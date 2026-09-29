package logx

import (
	"log"
	"os"
	"path/filepath"
)

func NewLogger(sessionID string) (*log.Logger, error) {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		return nil, err
	}

	path := filepath.Join("logs", sessionID+".log")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	logger := log.New(f, "", log.LstdFlags|log.Lmicroseconds)

	return logger, nil
}
