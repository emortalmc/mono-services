package healthprovider

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"time"
)

type checker struct {
	srv *health.Server

	repo repository.Repository
}

func Create(ctx context.Context, repo repository.Repository) *health.Server {
	srv := health.NewServer()

	for _, service := range config.AllServices {
		srv.SetServingStatus(service, healthpb.HealthCheckResponse_UNKNOWN)
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()

	c := &checker{
		srv:  srv,
		repo: repo,
	}

	c.startHealthChecks(ctx)

	return srv
}

func (c *checker) startHealthChecks(ctx context.Context) {
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				c.performHealthCheck(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *checker) performHealthCheck(ctx context.Context) {
	// MongoDB
	if err := c.repo.Ping(ctx); err != nil {
		c.srv.SetServingStatus(config.MongoServiceName, healthpb.HealthCheckResponse_NOT_SERVING)
	} else {
		c.srv.SetServingStatus(config.MongoServiceName, healthpb.HealthCheckResponse_SERVING)
	}
}
