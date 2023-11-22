package core

import (
    "github.com/PuerkitoBio/goquery"
)

type BetParser interface {
    ParseMatchUrls(url string) ([]string, error)
    ParseAll(url string) error
    ParseMatchData(s *goquery.Selection) (int, error)
    ParseMatchBets(id int, s *goquery.Selection) error
}

type Parser struct {
}

type Selector struct {
    TeamName string
    MatchDate string
    MatchTournament string
    BetTitle string
    BetDiv string
    BetOpt string
    BetCoef string
}

