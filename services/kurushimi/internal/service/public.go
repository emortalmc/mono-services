package service

import (
	"context"
	"fmt"
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/config"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/kafka"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/simplecontroller"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/utils/grpczap"
	"github.com/emortalmc/proto-specs/gen/go/grpc/matchmaker"
	"github.com/emortalmc/proto-specs/gen/go/grpc/party"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"sync"
)

func RunServices(ctx context.Context, logger *zap.SugaredLogger, wg *sync.WaitGroup, cfg config.Config,
	repo repository.Repository, notifier kafka.Notifier, gameModeController liveconfig.GameModeConfigController,
	lobbyCtrl simplecontroller.SimpleController, velocityCtrl simplecontroller.SimpleController,
	partyService party.PartyServiceClient, partySettingsService party.PartySettingsServiceClient) {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GrpcPort))
	if err != nil {
		logger.Fatalw("failed to listen", err)
	}

	opts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
		// Add any other option (check functions starting with logging.With).
	}

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(grpczap.InterceptorLogger(logger.Desugar()), opts...),
	))

	if cfg.Development {
		reflection.Register(s)
	}

	matchmaker.RegisterMatchmakerServer(s, newMatchmakerService(logger, repo, notifier, gameModeController, lobbyCtrl,
		velocityCtrl, partyService, partySettingsService))
	logger.Infow("listening for gRPC requests", "port", cfg.GrpcPort)

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Fatalw("failed to serve", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		s.GracefulStop()
	}()
}
