package app

import (
	"context"
	"github.com/emortalmc/mono-services/services/permission-service/internal/config"
	"github.com/emortalmc/mono-services/services/permission-service/internal/kafka/notifier"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository"
	"github.com/emortalmc/mono-services/services/permission-service/internal/service"
	"go.uber.org/zap"
	"sync"
)

func Run(cfg config.Config, logger *zap.SugaredLogger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	delayedCtx, repoCancel := context.WithCancel(ctx)
	delayedWg := &sync.WaitGroup{}

	repo, err := repository.NewMongoRepository(delayedCtx, logger, delayedWg, cfg.MongoDB)
	if err != nil {
		logger.Fatalw("failed to create repository", "error", err)
	}

	notif := notifier.NewKafkaNotifier(delayedCtx, delayedWg, logger, cfg.Kafka)

	service.RunServices(ctx, logger, wg, cfg, repo, notif)

	wg.Wait()
	logger.Info("shutting down")

	logger.Info("shutting down delayed services")
	repoCancel()
	delayedWg.Wait()
}
