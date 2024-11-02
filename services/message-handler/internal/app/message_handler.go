package app

import (
	"context"
	"github.com/emortalmc/mono-services/services/message-handler/internal/clients"
	"github.com/emortalmc/mono-services/services/message-handler/internal/config"
	"github.com/emortalmc/mono-services/services/message-handler/internal/kafka"
	"github.com/emortalmc/mono-services/services/message-handler/internal/service"
	"go.uber.org/zap"
	"os/signal"
	"sync"
	"syscall"
)

func Run(cfg *config.Config, logger *zap.SugaredLogger) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	relClient, err := clients.NewRelationshipClient(cfg.RelationshipService)
	if err != nil {
		logger.Fatalw("failed to connect to relationship service", err)
	}

	playerTrackerClient, err := clients.NewPlayerTrackerClient(cfg.PlayerTrackerService)
	if err != nil {
		logger.Fatalw("failed to connect to player tracker service", err)
	}

	permClient, err := clients.NewPermissionClient(cfg.PermissionService)
	if err != nil {
		logger.Fatalw("failed to connect to permission service", err)
	}

	badgeClient, err := clients.NewBadgeClient(cfg.BadgeService)
	if err != nil {
		logger.Fatalw("failed to connect to badge service", err)
	}

	notif := kafka.NewKafkaNotifier(cfg.Kafka, logger)

	kafka.NewConsumer(ctx, wg, cfg.Kafka, logger, notif, permClient, badgeClient)

	service.RunServices(ctx, logger, wg, cfg, notif, relClient, playerTrackerClient)

	wg.Wait()
	logger.Info("shutting down")
}
