package app

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/app/badge"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/app/player"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/grpc"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/kafka/consumer"
	kafkaWriter "github.com/emortalmc/mono-services/services/mc-player-service/internal/kafka/writer"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	"go.uber.org/zap"
	"os/signal"
	"sync"
	"syscall"
)

func Run(cfg config.Config, log *zap.SugaredLogger) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	badgeCfg, err := config.LoadBadgeConfig()
	if err != nil {
		log.Fatalw("failed to load badge config", err)
	}
	log.Infow("loaded badge config", "badgeCount", len(badgeCfg.Badges))

	repoWg := &sync.WaitGroup{}
	repoCtx, repoCancel := context.WithCancel(ctx)

	repo, err := repository.NewMongoRepository(repoCtx, log, repoWg, cfg.MongoDB)
	if err != nil {
		log.Fatalw("failed to create repository", err)
	}

	notifier := kafkaWriter.NewKafkaNotifier(ctx, wg, cfg.Kafka, log)

	badgeSvc := badge.NewService(log, repo, repo, badgeCfg)
	playerSvc := player.NewService(log, cfg, repo, notifier)

	kafkaConsumer.NewConsumer(ctx, wg, cfg, log, repo, badgeSvc, playerSvc)

	grpc.RunServices(ctx, log, wg, cfg, badgeSvc, badgeCfg, playerSvc, repo)

	wg.Wait()
	log.Info("shutting down")

	log.Info("shutting down repository")
	repoCancel()
	repoWg.Wait()
}
