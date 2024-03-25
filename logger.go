package migrate

import (
	"log"
)

type Logger interface {
	Printf(format string, args ...interface{})
}

type DefaultLogger struct{}

func (l DefaultLogger) Printf(msg string, args ...interface{}) {
	log.Printf(msg, args...)
}
