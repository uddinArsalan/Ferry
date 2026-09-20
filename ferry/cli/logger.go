package cli

import (
	"log"
	"os"
)

type Logger struct {
	log *log.Logger
}

func NewLogger() Logger {
	log := log.New(os.Stdout, "[FERRY]", log.LstdFlags|log.Lmicroseconds)
	return Logger{
		log: log,
	}
}

func (l Logger) GetLogger() *log.Logger {
	return l.log
}
