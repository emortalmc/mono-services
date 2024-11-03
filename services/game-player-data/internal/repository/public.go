package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

type GameDataRepository[T model.GameData] interface {
	Get(ctx context.Context, playerID uuid.UUID) (T, error)
	GetOrDefault(ctx context.Context, playerID uuid.UUID, defaultData T) (T, error)
	GetMultiple(ctx context.Context, playerIds []uuid.UUID) ([]T, error)
	Save(ctx context.Context, data T) error
}

type GameDataRepoColl struct {
	BlockSumo GameDataRepository[*model.BlockSumoData]
	Marathon  GameDataRepository[*model.MarathonData]
}

func NewGamePlayerDataRepoColl(db *mongo.Database) *GameDataRepoColl {
	return &GameDataRepoColl{
		BlockSumo: NewMongoGameDataRepository(db, &model.BlockSumoData{}),
		Marathon:  NewMongoGameDataRepository(db, &model.MarathonData{}),
	}
}
