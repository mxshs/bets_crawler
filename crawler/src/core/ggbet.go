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


func GetGgbetParser(db *db.DB) BetParser {
    parser := GgbetParser{}
    parser.driverOpts = chromedp.DefaultExecAllocatorOptions[:]
    parser.DB = db

    return &parser
}

type GgbetParser struct {
    Parser
    driverOpts []func(*chromedp.ExecAllocator)
    DB *db.DB
}

func (gp *GgbetParser) ParseMatchUrls(url string) ([]string, error) {
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), gp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady(`div[data-test="sport-event-list"]`, chromedp.ByQuery),
        chromedp.InnerHTML(`div[data-test="sport-event-list"]`, &domNode),
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

    doc.Find(`div[data-test="sport-event-in-view-subscription"]`).Each(
        func(i int, s *goquery.Selection) {
            if url, ok := s.Children().Filter("a").First().Attr("href"); ok {
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

func (gp *GgbetParser) ParseAll(url string) error {
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), gp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate("https://the-ggbet.com" + url),
        chromedp.WaitReady(`div[data-tab="All"]`, chromedp.ByQuery),
        chromedp.Click(`div[data-tab="All"]`, chromedp.ByQuery),
        chromedp.Sleep(1 * time.Second),
        chromedp.WaitReady(`body`, chromedp.ByQuery),
        chromedp.InnerHTML(`body`, &domNode),
    )
    if err != nil {
        return err
    }

	reader := strings.NewReader(domNode)

    doc, err := goquery.NewDocumentFromReader(reader)
    if err != nil {
        return err
    }

    id, err := gp.ParseMatchData(doc.Selection)
    if err != nil {
        return err
    }

    err = gp.ParseMatchBets(id, doc.Find(`div[data-test="markets"]`))

    return err 
}

func (gp *GgbetParser) ParseMatchData(s *goquery.Selection) (int, error) {

	game := &domain.GameBets{}

    var teams []string
    s.Find(`span[data-test="competitor-title"]`).Each(
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
    s.Find(`div[data-test="competitors"]`).Children().First().Children().Each(
        func(i int, s *goquery.Selection) {
            dateNode = append(dateNode, s.Text())
        },
    )

    dateField, err := gp.validateDate(dateNode)
    if err != nil {
        return 0, err
    }

	game.Date = *dateField

	tournament := strings.TrimSpace(
        s.Find(`span[data-test="match-helper-top-bar__tournament-name"]`).First().Text())
	game.Tournament = tournament

	id, err := gp.DB.InsertGame(game)

	return id, err
}

func (gp *GgbetParser) ParseMatchBets(game_id int, s *goquery.Selection) (error) {

    var err error

	s.Children().First().Children().Each(func(i int, s *goquery.Selection) {
		bet := &domain.Bet{}

		bet.Type = strings.TrimSpace(s.Find(`div[data-test="market-name"]`).First().Text())

        s.Find(`div[data-test="market-group"]`).Children().Each(
            func(i int, s *goquery.Selection) {
                option := domain.Option{}
                option.Name = strings.TrimSpace(
                    s.Find(`div[data-test="odd-button__title"]`).First().Text())
                option.Value = strings.TrimSpace(
                    s.Find(`div[data-test="odd-button__result"]`).First().Text())

                bet.Opts = append(bet.Opts, option)
            },
        )

		_, err = gp.DB.InsertBet(game_id, bet)
	})

	return err
}

func (gp *GgbetParser) validateDate(d []string) (*time.Time, error) {

    if len(d) != 2 {
        fmt.Printf(
            "[INFO] Number of parsed date items: %d, expected: %d\n",
            len(d),
            2,
        )

        t := time.Now()
        return &t, nil
    }

    hrs := strings.Split(d[0], ":")
    date := strings.Split(d[1], " ")

    dateField := time.Time{}

    switch date[0] {
    case "Today":
        d := time.Now()
        dateField = dateField.AddDate(d.Year() - 1, int(d.Month()) - 1, d.Day() - 1)
    case "Tomorrow":
        d := time.Now()
        dateField = dateField.AddDate(d.Year() - 1, int(d.Month()) - 1, d.Day())
    default:
        d, err := strconv.Atoi(date[0])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert date (days): %s", err.Error(),
            )
        }

        m, err := strconv.Atoi(date[1]) 
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert date (months): %s", err.Error(),
            )
        }

        y, err := strconv.Atoi(date[2])
        if err != nil {
            return nil, fmt.Errorf(
                "[ERROR] Failed to convert date (years): %s", err.Error(),
            )
        }

        dateField = dateField.AddDate(y, m, d)
    }

    h, err := strconv.Atoi(hrs[0])
    if err != nil {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert timestamp (hours): %s", err.Error(),
        )
    }

    m, err := strconv.Atoi(hrs[1])
    if err != nil {
        return nil, fmt.Errorf(
            "[ERROR] Failed to convert timestamp (minutes): %s", err.Error(),
        )
    }

    dateField = dateField.Add(time.Duration(h * int(time.Hour) + m * int(time.Minute)))

    return &dateField, nil
}

