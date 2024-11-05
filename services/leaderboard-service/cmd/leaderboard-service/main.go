package main

import (
	"github.com/emortalmc/mono-services/services/leaderboard-service/internal/config"
	"github.com/emortalmc/mono-services/services/leaderboard-service/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		// Config
		fx.Provide(config.LoadGlobalConfig),

		// Logging
		fx.Provide(
			newZapLogger,
			newZapSugared,
		),

		// Storage - MongoDB
		fx.Provide(newRepoMongo),

		fx.Invoke(func(log *zap.SugaredLogger) {
			log.Info("Starting leaderboard service")
		}),

		// gRPC
	).Run()
}

func newZapLogger(conf config.Config) (*zap.Logger, error) {
	if conf.Development {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func newZapSugared(log *zap.Logger) *zap.SugaredLogger {
	zap.ReplaceGlobals(log)
	return log.Sugar()
}

func newRepoMongo(cfg config.Config, lc fx.Lifecycle) (repository.Repository, error) {
	c, err := repository.NewMongoRepository(cfg.MongoDB)

	lc.Append(fx.Hook{OnStart: c.Start, OnStop: c.Shutdown})
	return c, err
}
