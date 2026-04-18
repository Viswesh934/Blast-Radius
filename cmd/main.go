package main

import (
	"log"

	"github.com/Viswesh934/blast-radius/cmd/commands"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("starting Blast Radius CLI")
	if err := commands.Execute(); err != nil {
		log.Fatal(err)
	}
}
