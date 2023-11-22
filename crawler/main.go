package main

import (
	"fmt"

	"mxshs/crawler/src/parser"
)

func main() {
    err := parser.Parse("https://leon.ru/bets/esports/1970324836975012-dota2")
    if err != nil {
        panic(err)
    }

    fmt.Println("[INFO] Successfully finished parsing")
}

