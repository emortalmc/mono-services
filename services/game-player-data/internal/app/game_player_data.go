package app

import (
	"context"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/config"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/kafka"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/service"
	"go.uber.org/zap"
	"os/signal"
	"sync"
	"syscall"
)

func Run(cfg *config.Config, logger *zap.SugaredLogger) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	repoCtx, repoCancel := context.WithCancel(ctx)
	repoWg := &sync.WaitGroup{}

	mongoDB, err := repository.CreateDatabase(repoCtx, cfg.MongoDB, repoWg, logger)
	if err != nil {
		logger.Fatalw("failed to create database", err)
	}

	repoColl := repository.NewGamePlayerDataRepoColl(mongoDB)

	kafka.NewConsumer(ctx, wg, cfg.Kafka, logger, repoColl)

	service.RunServices(ctx, logger, wg, cfg, repoColl)

	wg.Wait()
	logger.Info("shutting down")

	logger.Info("shutting down repository")
	repoCancel()
	repoWg.Wait()
}
