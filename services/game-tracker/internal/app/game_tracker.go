package app

import (
	"context"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/config"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/kafka"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository"
	"go.uber.org/zap"
	"os/signal"
	"sync"
	"syscall"
)

func Run(cfg config.Config, logger *zap.SugaredLogger) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	repoWg := &sync.WaitGroup{}
	repoCtx, repoCancel := context.WithCancel(ctx)

	repo, err := repository.NewMongoRepository(repoCtx, logger, repoWg, cfg.MongoDB)
	if err != nil {
		logger.Fatalw("failed to create repository", err)
	}

	kafka.NewConsumer(ctx, wg, cfg.Kafka, logger, repo)

	//service.RunServices(ctx, logger, wg, cfg, repo) todo: add services

	wg.Wait()
	logger.Info("shutting down")

	logger.Info("shutting down repository")
	repoCancel()
	repoWg.Wait()
}
