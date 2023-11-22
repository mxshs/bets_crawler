package core

import (
	"context"
	"strings"
	"time"
    "fmt"

	"mxshs/crawler/src/db"
	"mxshs/crawler/src/domain"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

func GetD2lParser(db *db.DB) BetParser {
    p := D2lParser{}
    p.driverOpts = chromedp.DefaultExecAllocatorOptions[:]
    p.DB = db

    return &p
}

type D2lParser struct {
    Parser
    driverOpts []func(*chromedp.ExecAllocator)
    DB *db.DB
}

func (lp *D2lParser) ParseMatchUrls(url string) ([]string, error) {
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), lp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady(`div .match_page`, chromedp.ByQuery),
        chromedp.InnerHTML(`div .match_page`, &domNode),
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

	doc.Find(`.lounge-bets-items__item`).Each(func(i int, s *goquery.Selection) {
		if url, ok := s.Find(`a`).First().Attr("href"); ok {
			urls = append(urls, "https://dota2lounge.com" + url)
		} else {
            err = fmt.Errorf(
                "[ERROR] Failed to get match url on main page (possibly HTML changed)\n",
            )
		}
	})

	return urls, err
}

func (lp *D2lParser) ParseAll(url string) error {

    ctx, cancel := chromedp.NewExecAllocator(context.Background(), lp.driverOpts...)
    defer cancel()

    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()

    var domNode string

    err := chromedp.Run(
        ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady(`div .match_page`, chromedp.ByQuery),
        chromedp.InnerHTML(`div .match_page`, &domNode),
    )
    if err != nil {
        return err
    }

	reader := strings.NewReader(domNode)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	gameId, err := lp.ParseMatchData(doc.Find(`div .lounge-match lounge-match_on-page`))
	if err != nil {
		return err
	}

	err = lp.ParseMatchBets(gameId, doc.Find(`div .lounge-events`))

	return err 
}

func (lp *D2lParser) ParseMatchData(s *goquery.Selection) (int, error) {

	game := &domain.GameBets{}

	teamA := strings.TrimSpace(s.Find(`.lounge-match__team_right`).Find(
		".lounge-team__title").First().Text())
	teamB := strings.TrimSpace(s.Find(`.lounge-match__team_left`).Find(
		".lounge-team__title").First().Text())

	game.TeamA = teamA
	game.TeamB = teamB

	date := strings.TrimSpace(s.Find(".lounge-match-date__date").First().Text())
	tournament := strings.TrimSpace(s.Find(".lounge-match__tournament").First().Text())

	datetime, err := time.Parse("2.1.2006, 15:04 MST", date)
	if err != nil {
		return 0, err
	}

	game.Date = datetime
	game.Tournament = tournament

	id, err := lp.DB.InsertGame(game)

	return id, err
}

func (lp *D2lParser) ParseMatchBets(game_id int, s *goquery.Selection) error {
	var err error

	s.Find(".lounge-event").Each(func(i int, s *goquery.Selection) {
		bet := &domain.Bet{}

		bet.Type = strings.TrimSpace(s.Find(".lounge-event__title").First().Text())

		s.Find(`.lounge-event__button`).Each(func(i int, s *goquery.Selection) {
			option := domain.Option{}
			option.Name = strings.TrimSpace(s.Find(".lounge-event-button__text").First().Text())
			option.Value = strings.TrimSpace(s.Find(".lounge-event-button__coeff").First().Text())
			bet.Opts = append(bet.Opts, option)
		})

		_, err = lp.DB.InsertBet(game_id, bet)
	})

	return err
}
