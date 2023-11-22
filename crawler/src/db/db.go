package db

import (
	"database/sql"
	"fmt"
	"os"

	"mxshs/crawler/src/domain"

	"github.com/joho/godotenv"
	pq "github.com/lib/pq"
)

var (
    HOST string
    PORT string
    USER string
    PASS string
    DBNAME string
)

type DB struct {
    db *sql.DB
}

func init() {
    err := godotenv.Load(".env")
    if err != nil {
        panic("Could not locate .env file")
    }

    HOST, _ = os.LookupEnv("DB_HOST")
    PORT, _ = os.LookupEnv("DB_PORT")
    USER, _ = os.LookupEnv("DB_USER")
    PASS, _ = os.LookupEnv("DB_PASS")
    DBNAME, _ = os.LookupEnv("DB")
}

func GetDB() (*DB, error) {
    db := &DB{}

    conn_info := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        HOST,
        PORT,
        USER,
        PASS,
        DBNAME,
    )

    conn, err := sql.Open("postgres", conn_info)
    if err != nil {
        return nil, err
    }

    db.db = conn

    return db, nil
}

func (db *DB) InsertBet(game_id int, bet *domain.Bet) (int, error) {
    var bet_id int

    bet_arr := [][]string{}

    for _, opt := range bet.Opts {
        bet_arr = append(bet_arr, []string{opt.Name, opt.Value})
    }

    q, err := db.db.Query(
        `INSERT INTO bets (type, bet, game_id)
        VALUES ($1, $2, $3) RETURNING bet_id;`,
        bet.Type,
        pq.Array(bet_arr),
        game_id,
    )
    if err != nil {
        return bet_id, err
    }

    q.Next()

    err = q.Scan(&bet_id)
    if err != nil {
        return bet_id, err
    }

    err = q.Close()

    return bet_id, err
}

func (db *DB) InsertGame(game *domain.GameBets) (int, error) {
    var game_id int

    q, err := db.db.Query(
        `INSERT INTO games (date, tournament, radiant, dire) 
        VALUES ($1, $2, $3, $4) RETURNING game_id;`,
        game.Date,
        game.Tournament,
        game.TeamA,
        game.TeamB,
    )
    if err != nil {
        return game_id, err
    }

    q.Next()

    err = q.Scan(&game_id)
    if err != nil {
        return game_id, err
    }

    err = q.Close()

    return game_id, err 
}

