package domain

import "time"

type GameBets struct {
    TeamA string
    TeamB string
    Date time.Time
    Tournament string
    Bets []Bet
}

type Bet struct {
    Type string
    Opts []Option
}

type Option struct {
    Name string
    Value string
}

