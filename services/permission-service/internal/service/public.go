package service

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/permission-service/internal/config"
	"github.com/emortalmc/mono-services/services/permission-service/internal/kafka/notifier"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository"
	"github.com/emortalmc/mono-services/services/permission-service/internal/utils/grpczap"
	"github.com/emortalmc/proto-specs/gen/go/grpc/permission"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"sync"
)

func RunServices(ctx context.Context, logger *zap.SugaredLogger, wg *sync.WaitGroup, cfg config.Config,
	repo repository.Repository, notif notifier.Notifier) {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		logger.Fatalw("failed to listen", "error", err)
	}

	opts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(grpczap.InterceptorLogger(logger.Desugar()), opts...),
	))

	if cfg.Development {
		reflection.Register(s)
	}

	permission.RegisterPermissionServiceServer(s, newPermissionService(logger, repo, notif))
	logger.Infow("listening for gRPC requests", "port", cfg.GRPCPort)

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Fatalw("failed to serve", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		s.GracefulStop()
	}()

}
