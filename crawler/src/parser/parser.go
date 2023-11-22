package parser

import (
    "fmt"
    "sync"

	"mxshs/crawler/src/core"
	"mxshs/crawler/src/db"
)

func Parse(url string) error {
    db, err := db.GetDB()
    if err != nil {
        return err
    }

    p := core.GetLeonParser(db)

    urls, err := p.ParseMatchUrls(url)
    if err != nil {
        return err
    }


    i := len(urls) - 1
   
    for i >= 0 {

        counter := 2

        var wg sync.WaitGroup

        for i >= 0 && counter >= 0 {

            wg.Add(1)
            url := urls[i]

            go func() {
                defer wg.Done()

                err := p.ParseAll(url)
                if err != nil {
                    fmt.Println(err.Error())
                }
            } ()

            counter -= 1
            i -= 1
        }

        wg.Wait()
    }

    return nil
}

