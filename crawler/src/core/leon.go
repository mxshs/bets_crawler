package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mxshs/crawler/src/db"
	"mxshs/crawler/src/domain"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)


func GetLeonParser(db *db.DB) BetParser {
    parser := LeonParser{}
    parser.driverOpts = chromedp.DefaultExecAllocatorOptions[:]
    parser.DB = db

    return &parser
}

var ru2en = map[string]int{
    "Янв": 1,
    "Фев": 2,
    "Мар": 3,
    "Апр": 4,
    "Май": 5,
    "Июн": 6,
    "Июл": 7,
    "Авг": 8,
    "Сен": 9,
    "Окт": 10,
    "Ноя": 11,
    "Дек": 12,
}


type LeonParser struct {
    Parser
    driverOpts []func(*chromedp.ExecAllocator)
    DB *db.DB
}

func (lp *LeonParser) ParseMatchUrls(url string) ([]string, error) {
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), lp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate(url),
        chromedp.WaitVisible(`div .sport-event-region`, chromedp.ByQuery),
        chromedp.InnerHTML(`div .sport-event-region`, &domNode),
    )
    if err != nil {
        return nil, err
    }

	reader := strings.NewReader(domNode)

    doc, err := goquery.NewDocumentFromReader(reader)
    if err != nil {
        return nil, err
    }

    var urls []string

    doc.Find(`div[data-test-el="sportline-event-block"]`).Each(
        func(i int, s *goquery.Selection) {
            if url, ok := s.Children().Find("a").First().Attr("href"); ok {
                urls = append(urls, url)
            } else {
                err = fmt.Errorf(
                    "[ERROR] Failed to get match url on main page (possibly HTML changed)\n",
                )
            }
        },
    )

    return urls, nil
}

func (lp *LeonParser) ParseAll(url string) error {
    // Using different setup to ensure that the website does not fall back to mobile version
    ctx, cancel := chromedp.NewExecAllocator(
        context.Background(),
        chromedp.Headless,
        chromedp.Flag("force-device-scale-factor", "1"),
        chromedp.Flag("window-size", "1920,1080"),
    )
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate("https://leon.ru" + url),
        chromedp.WaitReady(`div .sport-event-details-market-list_pY0E1`, chromedp.ByQuery),
        chromedp.InnerHTML(`div .sport-event-details`, &domNode),
    )
    if err != nil {
        return err
    }

	reader := strings.NewReader(domNode)

    doc, err := goquery.NewDocumentFromReader(reader)
    if err != nil {
        return err
    }

    id, err := lp.ParseMatchData(doc.Selection)
    if err != nil {
        return err
    }

    err = lp.ParseMatchBets(id, doc.Find(`div .sport-event-details__markets_G3m4g`))

    return err 
}

func (lp *LeonParser) ParseMatchData(s *goquery.Selection) (int, error) {

	game := &domain.GameBets{}

    radiant := strings.TrimSpace(s.Find(`div .headline-info__team`).First().Text())
    dire := strings.TrimSpace(s.Find(`div .headline-info__team`).Last().Text())

    if len(radiant) == 0 || len(dire) == 0 {
        return 0, fmt.Errorf(
            "[ERROR] Could not parse team names (got zero-length values)\n",
        )
    }

	game.TeamA = radiant
	game.TeamB = dire

    var dateNode []string
    s.Find(`div .headline-info__date`).Children().Filter(`span`).Each(
        func(i int, s *goquery.Selection) {
            dateNode = append(dateNode, s.Text())
        },
    )

    dateField, err := lp.validateDate(dateNode)
    if err != nil {
        return 0, err
    }

	game.Date = *dateField

	tournament := strings.TrimSpace(s.Find(
        `div .breadcrumb__title`).Eq(-2).Text())
	game.Tournament = tournament

	id, err := lp.DB.InsertGame(game)

	return id, err
}

func (lp *LeonParser) ParseMatchBets(game_id int, s *goquery.Selection) (error) {

    var err error

	s.Children().First().Children().Each(func(i int, s *goquery.Selection) {
		bet := &domain.Bet{}

		bet.Type = (
            s.Find(`div .sport-event-details-market-group__title`).Text())

        s.Find(`div .sport-event-details-item__runner-holder`).Each(
            func(i int, s *goquery.Selection) {
                s = s.Children().First().Find(`span`)

                option := domain.Option{}
                option.Name = strings.TrimSpace(s.First().Text())
                option.Value = strings.TrimSpace(s.Last().Text())

                bet.Opts = append(bet.Opts, option)
            },
        )

		_, err = lp.DB.InsertBet(game_id, bet)
	})

	return err
}

func (lp *LeonParser) validateDate(d []string) (*time.Time, error) {

    if len(d) != 2 {
        fmt.Printf(
            "[INFO] Number of parsed date items: %d, expected: %d\n",
            len(d),
            2,
        )

        t := time.Now()
        return &t, nil
    }

    date := strings.Split(d[0], " ")
    timestamp := strings.Split(d[1], ":")

    dateField := time.Time{}

    days, err := strconv.Atoi(date[0])
    if err != nil {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert date (days): %s", err.Error(),
        )
    }

    months, ok := ru2en[date[1][:6]]
    if !ok {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert date (months): cannot find %s in translation map",
            date[1],
        )
    }

    years, err := strconv.Atoi(date[2])
    if err != nil {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert date (years): %s", err.Error(),
        )
    }

    dateField = dateField.AddDate(years - 1, months - 1, days - 1)

    hrs, err := strconv.Atoi(timestamp[0])
    if err != nil {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert timestamp (hrs): %s", err.Error(),
        )
    }

    mins, err := strconv.Atoi(timestamp[1])
    if err != nil {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert timestamp (minutes): %s", err.Error(),
        )
    }

    dateField = dateField.Add(
        time.Duration(hrs) * time.Hour + time.Duration(mins) * time.Minute,
    )
    
    return &dateField, nil
}

