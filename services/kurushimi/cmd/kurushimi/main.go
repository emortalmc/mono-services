package main

import (
	"github.com/emortalmc/mono-services/services/kurushimi/internal/app"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/config"
	"go.uber.org/zap"
	"log"
)

func main() {
	cfg := config.LoadGlobalConfig()

	unsugared, err := createLogger(cfg)
	if err != nil {
		log.Fatal(err)
	}
	logger := unsugared.Sugar()

	app.Run(cfg, logger)
}

func createLogger(cfg config.Config) (logger *zap.Logger, err error) {
	if cfg.Development {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	return
}
