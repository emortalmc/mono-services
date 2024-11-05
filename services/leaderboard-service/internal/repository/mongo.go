package repository

import (
	"context"
	"fmt"
	mongoUtils "github.com/emortalmc/mono-services/libraries/mongo/pkg/utils"
	"github.com/emortalmc/mono-services/services/leaderboard-service/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName = "leaderboard-service"
)

type MongoRepository struct {
	URI string

	client   *mongo.Client
	database *mongo.Database

	collection *mongo.Collection
}

func NewMongoRepository(cfg config.MongoDBConfig) (*MongoRepository, error) {
	return &MongoRepository{
		URI: cfg.URI,
	}, nil
}

func (r *MongoRepository) Start(ctx context.Context) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(r.URI).SetRegistry(mongoUtils.NewRegistry(mongoUtils.UUIDType)))
	if err != nil {
		return err
	}

	r.client = client
	r.database = client.Database(databaseName)

	return nil
}

func (r *MongoRepository) Shutdown(ctx context.Context) error {
	if err := r.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from mongo: %w", err)
	}

	return nil
}
