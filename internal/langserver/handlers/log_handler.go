package handlers

import (
	"log"
)

// logHandler provides handlers logger
type logHandler struct {
	logger *log.Logger
}

func LogHandler(logger *log.Logger) *logHandler {
	return &logHandler{logger}
}
