package main

import (
	"log"

	"github.com/Viswesh934/blast-radius/internal/tui"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("starting Blast Radius TUI")

	if err := tui.Run(); err != nil {
		log.Fatal(err)
	}
}
