package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (m *mongoRepository) GetBadgePlayer(ctx context.Context, playerId uuid.UUID) (model.BadgePlayer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var player model.BadgePlayer
	if err := m.playerCollection.FindOne(ctx, bson.M{"_id": playerId}, options.FindOne().SetProjection(model.BadgePlayerProjection)).
		Decode(&player); err != nil {
		return model.BadgePlayer{}, err
	}

	return player, nil
}

func (m *mongoRepository) UpdatePlayerBadgesAndActive(ctx context.Context, playerId uuid.UUID, badges []string,
	activeBadge *string) error {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"badges": badges}}
	if activeBadge != nil {
		update["$set"].(bson.M)["activeBadge"] = activeBadge
	} else {
		update["$unset"] = bson.M{"activeBadge": ""}
	}

	if r, err := m.playerCollection.UpdateByID(ctx, playerId, update); err != nil {
		return err
	} else if r.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *mongoRepository) AddPlayerBadge(ctx context.Context, playerId uuid.UUID, badgeId string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{"$addToSet": bson.M{"badges": badgeId}})
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (m *mongoRepository) RemovePlayerBadge(ctx context.Context, playerId uuid.UUID, badgeId string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{"$pull": bson.M{"badges": badgeId}})
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (m *mongoRepository) GetActivePlayerBadge(ctx context.Context, playerId uuid.UUID) (*string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result := struct {
		ActiveBadge *string `bson:"activeBadge"`
	}{}

	opts := options.FindOne().SetProjection(bson.M{"activeBadge": 1})
	if err := m.playerCollection.FindOne(ctx, bson.M{"_id": playerId}, opts).Decode(&result); err != nil {
		return nil, err
	}

	return result.ActiveBadge, nil
}

func (m *mongoRepository) SetActivePlayerBadge(ctx context.Context, playerId uuid.UUID, badgeId *string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if badgeId == nil {
		_, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{"$unset": bson.M{"activeBadge": ""}})
		return err
	}

	_, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{"$set": bson.M{"activeBadge": badgeId}})
	return err
}
