package main

import (
	"github.com/emortalmc/mono-services/services/relationship-manager-service/internal/app"
	"github.com/emortalmc/mono-services/services/relationship-manager-service/internal/config"
	"go.uber.org/zap"
	"log"
)

func main() {
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := createLogger(cfg)
	if err != nil {
		log.Fatal(err)
	}

	app.Run(cfg, logger)
}

func createLogger(cfg *config.Config) (*zap.SugaredLogger, error) {
	var logger *zap.Logger
	var err error
	if cfg.Development {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}
