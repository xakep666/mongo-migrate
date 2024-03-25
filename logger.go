package migrate

import (
	"log"
)

type Logger interface {
	Printf(format string, args ...any)
}

type DefaultLogger struct{}

func (l DefaultLogger) Printf(msg string, args ...any) {
	log.Printf(msg, args...)
}
