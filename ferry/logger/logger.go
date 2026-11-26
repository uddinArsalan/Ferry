package logger

import "log"

type Logger interface {
	GetLogger() *log.Logger
}
