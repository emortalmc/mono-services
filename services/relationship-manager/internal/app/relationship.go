package app

import (
	"context"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/clients"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/config"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/kafka"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/repository"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/service"
	"go.uber.org/zap"
	"os/signal"
	"sync"
	"syscall"
)

func Run(cfg *config.Config, logger *zap.SugaredLogger) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	// Mongo and Kafka get a delayed context to make sure they are shut down after requests are finished.
	delayedWg := &sync.WaitGroup{}
	delayedCtx, delayedCancel := context.WithCancel(ctx)

	repo, err := repository.NewMongoRepository(delayedCtx, logger, delayedWg, cfg.MongoDB)
	if err != nil {
		logger.Fatalw("failed to create repository", "error", err)
	}

	notif := kafka.NewKafkaNotifier(delayedCtx, wg, cfg.Kafka, logger)
	if err != nil {
		logger.Fatalw("failed to create notifier", "error", err)
	}

	service.RunServices(ctx, logger, wg, cfg, repo, notif)

	playerTracker, err := clients.NewPlayerTrackerClient(cfg.PlayerTrackerService)
	if err != nil {
		logger.Fatalw("failed to create player tracker client", "error", err)
	}

	kafka.NewConsumer(ctx, wg, cfg.Kafka, logger, repo, notif, playerTracker)

	wg.Wait()
	logger.Info("stopped services")

	logger.Info("shutting down repository and kafka")
	delayedCancel()
	delayedWg.Wait()
}
