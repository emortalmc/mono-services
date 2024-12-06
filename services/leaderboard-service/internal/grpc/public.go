package grpc

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/leaderboard-service/internal/config"
	"github.com/emortalmc/mono-services/services/leaderboard-service/internal/repository"
	"github.com/emortalmc/proto-specs/gen/go/grpc/leaderboard"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"net"
)

type GrpcRunner struct {
	log *zap.SugaredLogger
	cfg config.Config
	s   *grpc.Server
}

func RunServices(log *zap.SugaredLogger, cfg config.Config, repo repository.Repository) (*GrpcRunner, error) {
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		grpczap.UnaryServerInterceptor(log.Desugar(), grpczap.WithLevels(func(code codes.Code) zapcore.Level {
			if code != codes.Internal && code != codes.Unavailable && code != codes.Unknown {
				return zapcore.DebugLevel
			} else {
				return zapcore.ErrorLevel
			}
		})),
	))

	if cfg.Development {
		reflection.Register(s)
	}

	leaderboard.RegisterLeaderboardServer(s, newLeaderboardService(repo))

	log.Infow("listening for gRPC requests", "port", cfg.Port)

	return &GrpcRunner{
		log: log,
		cfg: cfg,
		s:   s,
	}, nil
}

func (r *GrpcRunner) Start(_ context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.cfg.Port))
	if err != nil {
		return err
	}

	go func() {
		if err := r.s.Serve(lis); err != nil {
			r.log.Fatalw("failed to serve", err)
		}
	}()

	return nil
}

func (r *GrpcRunner) Shutdown() error {
	r.s.GracefulStop()
	return nil
}
