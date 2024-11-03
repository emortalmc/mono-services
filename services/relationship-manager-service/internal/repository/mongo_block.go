package repository

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
"github.com/emortalmc/mono-services/services/relationship-manager-service/internal/repository/model"
"time"
)

var (
	AlreadyBlockedError = errors.New("player already blocked")
)

func (m *mongoRepository) CreatePlayerBlock(ctx context.Context, block *model.PlayerBlock) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.blockColl.InsertOne(ctx, block)

	if mongo.IsDuplicateKeyError(err) {
		return AlreadyBlockedError
	}

	return err
}

func (m *mongoRepository) DeletePlayerBlock(ctx context.Context, blockerId uuid.UUID, blockedId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.blockColl.DeleteOne(ctx, bson.M{"blockerId": blockerId, "blockedId": blockedId})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (m *mongoRepository) IsPlayerBlocked(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.blockColl.CountDocuments(ctx, bson.M{"$or": []bson.M{
		{"blockerId": playerOneId, "blockedId": playerTwoId},
		{"blockerId": playerTwoId, "blockedId": playerOneId},
	}})
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (m *mongoRepository) GetMutualBlocks(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) ([]*model.PlayerBlock, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.blockColl.Find(ctx, bson.M{"$or": []bson.M{
		{"blockerId": playerOneId, "blockedId": playerTwoId},
		{"blockerId": playerTwoId, "blockedId": playerOneId},
	}})
	if err != nil {
		return nil, err
	}

	result := make([]*model.PlayerBlock, 0)
	err = cursor.All(ctx, &result)
	return result, err
}

func (m *mongoRepository) GetPlayerBlocks(ctx context.Context, playerId uuid.UUID) ([]*model.PlayerBlock, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.blockColl.Find(ctx, bson.M{"blockerId": playerId})
	if err != nil {
		return nil, err
	}
	result := make([]*model.PlayerBlock, 0)
	err = cursor.All(ctx, &result)
	return result, err
}
