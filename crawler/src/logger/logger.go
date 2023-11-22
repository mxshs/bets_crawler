package logger

import (
    "log"
	"os"
)
        
var Logger *log.Logger

func init() {
    Logger = log.New(os.Stdout, "crawler ", log.LstdFlags)
}

