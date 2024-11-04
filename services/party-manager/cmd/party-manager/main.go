package main

import (
	"github.com/emortalmc/mono-services/services/party-manager/internal/app"
	"github.com/emortalmc/mono-services/services/party-manager/internal/config"
	"go.uber.org/zap"
	"log"
)

func main() {
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		log.Fatal("failed to load config", err)
	}

	unsugared, err := createLogger(cfg)
	if err != nil {
		log.Fatal(err)
	}
	logger := unsugared.Sugar()

	app.Run(cfg, logger)
}

func createLogger(cfg *config.Config) (logger *zap.Logger, err error) {
	if cfg.Development {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}
	return logger, nil
}
