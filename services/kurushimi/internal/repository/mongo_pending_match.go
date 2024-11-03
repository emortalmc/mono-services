package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func (m *mongoRepository) CreatePendingMatch(ctx context.Context, match *model.PendingMatch) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.pendingMatchCollection.InsertOne(ctx, match)
	return err
}

func (m *mongoRepository) CreatePendingMatches(ctx context.Context, matches []*model.PendingMatch) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	converted := make([]interface{}, len(matches))
	for i, match := range matches {
		converted[i] = match
	}
	_, err := m.pendingMatchCollection.InsertMany(ctx, converted)
	return err
}

func (m *mongoRepository) UpdatePendingMatches(ctx context.Context, matches []*model.PendingMatch) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	models := make([]mongo.WriteModel, len(matches))
	for i, match := range matches {
		models[i] = mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": match.Id}).
			SetUpdate(bson.M{"$set": match})
	}

	_, err := m.pendingMatchCollection.BulkWrite(ctx, models)
	return err
}

func (m *mongoRepository) DeletePendingMatches(ctx context.Context, matchIds []primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": bson.M{"$in": matchIds}}
	_, err := m.pendingMatchCollection.DeleteMany(ctx, filter)
	return err
}

func (m *mongoRepository) GetPendingMatchesByGameMode(ctx context.Context, gameModeId string) ([]*model.PendingMatch, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.pendingMatchCollection.Find(ctx, bson.M{"gameModeId": gameModeId})
	if err != nil {
		return nil, err
	}

	var matches []*model.PendingMatch
	err = cursor.All(ctx, &matches)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (m *mongoRepository) GetPendingMatchByTicketId(ctx context.Context, ticketId primitive.ObjectID) (*model.PendingMatch, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var match model.PendingMatch

	if err := m.pendingMatchCollection.FindOne(ctx, bson.M{"ticketIds": ticketId}).Decode(&match); err != nil {
		return nil, err
	}

	return &match, nil
}

func (m *mongoRepository) RemoveTicketsFromPendingMatchesById(ctx context.Context, ticketIds []primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"ticketIds": bson.M{"$in": ticketIds}}
	update := bson.M{"$pull": bson.M{"ticketIds": bson.M{"$in": ticketIds}}}

	result, err := m.pendingMatchCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}
