package app

import (
	"context"
	"fmt"
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/config"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/director"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/kafka"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/repository"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/service"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/simplecontroller"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/utils/kubernetes"
	"github.com/emortalmc/proto-specs/gen/go/grpc/party"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func Run(cfg config.Config, logger *zap.SugaredLogger) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	repoCtx, repoCancel := context.WithCancel(ctx)
	repoWg := &sync.WaitGroup{}

	// Parse gamemode configs
	gameModeController, err := liveconfig.NewGameModeConfigController(logger)
	if err != nil {
		logger.Fatalw("failed to create game mode config controller", err)
	}

	gameModes := gameModeController.GetConfigs()

	modeNames := make([]string, 0)
	for id := range gameModes {
		modeNames = append(modeNames, id)
	}
	logger.Infow("loaded initial gamemodes", "modeCount", len(gameModes), "modes", modeNames)

	_, agonesClient := kubernetes.CreateClients()

	repo, err := repository.NewMongoRepository(repoCtx, repoWg, logger, cfg.MongoDB)
	if err != nil {
		logger.Fatalw("failed to connect to mongo", err)
	}

	notifier := kafka.NewKafkaNotifier(ctx, wg, cfg.Kafka, logger)

	kafka.NewConsumer(ctx, wg, cfg.Kafka, logger, repo)

	err = repo.HealthCheck(ctx, 5*time.Second)
	if err != nil {
		logger.Fatalw("failed to initiate mongodb", err)
	}

	pSConn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.PartyService.SettingsHost, cfg.PartyService.SettingsPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalw("failed to connect to party service", err)
	}
	partySettingsService := party.NewPartySettingsServiceClient(pSConn)

	allocationClient := agonesClient.AllocationV1().GameServerAllocations(cfg.Namespace)

	pConn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.PartyService.Host, cfg.PartyService.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalw("failed to connect to party service", err)
	}

	partyService := party.NewPartyServiceClient(pConn)

	// Simple controllers
	lobbyCfg := cfg.Lobby
	proxyCfg := cfg.Proxy

	lobbyCtrl := simplecontroller.NewJoinController(ctx, wg, logger, notifier, allocationClient,
		lobbyCfg.FleetName, "lobby", lobbyCfg.MatchRate, lobbyCfg.MatchSize)

	velocityCtrl := simplecontroller.NewJoinController(ctx, wg, logger, notifier, allocationClient,
		proxyCfg.FleetName, "proxy", proxyCfg.MatchRate, proxyCfg.MatchSize)

	service.RunServices(ctx, logger, wg, cfg, repo, notifier, gameModeController, lobbyCtrl, velocityCtrl, partyService, partySettingsService)

	directR := director.New(logger, repo, notifier, allocationClient, gameModeController)
	directR.Start(ctx)

	wg.Wait()

	logger.Info("shutting down repository")
	repoCancel()
	repoWg.Wait()
}
