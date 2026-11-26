package ferry_logger

import (
	"log"
	"os"
)

// ferry logger
type FerryLogger struct {
	log *log.Logger
}

func NewFerryLogger() *FerryLogger {
	log := log.New(os.Stdout, "[FERRY]", 0)
	return &FerryLogger{
		log: log,
	}
}

func (f *FerryLogger) GetLogger() *log.Logger {
	return f.log
}
