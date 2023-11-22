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


func GetLsParser(db *db.DB) BetParser {
    parser := LSParser{}
    parser.driverOpts = chromedp.DefaultExecAllocatorOptions[:]
    parser.DB = db

    return &parser
}

type LSParser struct {
    Parser
    driverOpts []func(*chromedp.ExecAllocator)
    DB *db.DB
}

func (lp *LSParser) ParseMatchUrls(url string) ([]string, error) {
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), lp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    fmt.Println(url)
    err := chromedp.Run(
        ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady(`body`, chromedp.ByQuery),
        chromedp.InnerHTML(`body`, &domNode),
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

    doc.Find(`div .bui-event-row-dfbc70`).Each(
        func(i int, s *goquery.Selection) {
            if url, ok := s.Find("a").First().Attr("href"); ok {
                urls = append(urls, url)
            } else {
                err = fmt.Errorf(
                    "[ERROR] Failed to get match url on main page (possibly HTML changed)\n",
                )
            }
        },
    )

    return urls, err
}

func (lp *LSParser) ParseAll(url string) error {
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), lp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate("https://www.ligastavok.ru" + url),
        chromedp.WaitReady(`div #content`, chromedp.ByQuery),
        chromedp.InnerHTML(`div #content`, &domNode),
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

    err = lp.ParseMatchBets(id, doc.Find(`div .part__markets-86eb26`))

    return err 
}

func (lp *LSParser) ParseMatchData(s *goquery.Selection) (int, error) {

	game := &domain.GameBets{}

    var teams []string
    s.Find(`div[itemprop="performer"]`).Each(
        func(i int, s *goquery.Selection) {
            teams = append(teams, s.Text())
        },
    )

    if len(teams) != 2 {
        return 0, fmt.Errorf(
            "[ERROR] Number of parsed teams: %d, expected: %d\n",
            len(teams),
            2,
        )
    }

	game.TeamA = strings.TrimSpace(teams[0])
	game.TeamB = strings.TrimSpace(teams[1])

    var dateNode []string

    s.Find(`div .event-header__time-wrapper-1eccdf`).Children().First().Children().Each(
        func(i int, s *goquery.Selection) {
            dateNode = append(dateNode, s.Text())
        },
    )

    dateField, err := lp.validateDate(dateNode)
    if err != nil {
        return 0, err
    }

	game.Date = *dateField

	tournament := strings.TrimSpace(s.Find(`a #event__breadcrumbs-tournament`).First().Text())
	game.Tournament = tournament

	id, err := lp.DB.InsertGame(game)

	return id, err
}

func (lp *LSParser) ParseMatchBets(game_id int, s *goquery.Selection) (error) {

    var err error

	s.Children().Each(func(i int, s *goquery.Selection) {
		bet := &domain.Bet{}

		bet.Type = strings.TrimSpace(s.Find(`span .market__title-0ff163`).First().Text())

        s.Find(`div .market__outcomes-96e4e5`).Children().Each(func(i int, s *goquery.Selection) {
			option := domain.Option{}
			option.Name = strings.TrimSpace(s.Children().First().Text())
			option.Value = strings.TrimSpace(s.Children().Last().Text())
			bet.Opts = append(bet.Opts, option)
		})

		_, err = lp.DB.InsertBet(game_id, bet)
	})

	return err
}

func (lp *LSParser) validateDate(d []string) (*time.Time, error) {

    if len(d) != 2 {
        fmt.Printf(
            "[INFO] Number of parsed date items: %d, expected: %d\n",
            len(d),
            2,
        )

        t := time.Now()
        return &t, nil
    }

    date := strings.Split(d[1], "/")

    dateField := time.Time{}

    if _, err := strconv.Atoi(date[0]); err != nil {
        timestamp := strings.Split(d[0], " ")
        if len(timestamp) < 4 {
            fmt.Printf(
                "[ERROR] Number of parsed items in game timestamp: %d, expected: %d\n",
                len(d),
                4,
            )

            t := time.Now()
            return &t, nil
        }

        d := time.Now()
        dateField = dateField.AddDate(d.Year() - 1, int(d.Month()) - 1, d.Day() - 1)

        hrs, err := strconv.Atoi(timestamp[0])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert timestamp (hours): %s", err.Error(),
            )
        }

        mins, err := strconv.Atoi(timestamp[2])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert timestamp (minutes): %s", err.Error(),
            )
        }

        dateField.Add(time.Duration(hrs) * time.Hour + time.Duration(mins) * time.Minute)
    } else {
        days, err := strconv.Atoi(date[1])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert date (days): %s", err.Error(),
            )
        }

        months, err := strconv.Atoi(date[0]) 
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert date (months): %s", err.Error(),
            )
        }

        dateField = dateField.AddDate(time.Now().Year() - 1, months - 1, days - 1)

        timestamp := strings.Split(d[0], ":")
        if len(timestamp) < 2 {
            fmt.Printf(
                "[ERROR] Number of parsed items in game timestamp: %d, expected: %d\n",
                len(d),
                2,
            )

            t := time.Now()
            return &t, nil
        }

        hrs, err := strconv.Atoi(timestamp[0])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert timestamp (hours): %s", err.Error(),
            )
        }

        mins, err := strconv.Atoi(timestamp[1])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert timestamp (minutes): %s", err.Error(),
            )
        }

        dateField.Add(time.Duration(hrs) * time.Hour + time.Duration(mins) * time.Minute)
    }

    return &dateField, nil
}

