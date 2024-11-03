package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func (m *mongoRepository) CreateTicket(ctx context.Context, ticket *model.Ticket) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.ticketCollection.InsertOne(ctx, ticket)
	return err
}

func (m *mongoRepository) DeleteTicket(ctx context.Context, ticketId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.ticketCollection.DeleteOne(ctx, bson.M{"_id": ticketId})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return err
}

func (m *mongoRepository) MassUpdateTicketInPendingMatch(ctx context.Context, values map[primitive.ObjectID]bool) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var updates []mongo.WriteModel
	for ticketId, inPendingMatch := range values {
		updates = append(updates, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": ticketId}).SetUpdate(bson.M{"$set": bson.M{"inPendingMatch": inPendingMatch}}))
	}

	res, err := m.ticketCollection.BulkWrite(ctx, updates)
	if err != nil {
		return 0, err
	}

	return res.ModifiedCount, nil
}

func (m *mongoRepository) DeleteAllTicketsById(ctx context.Context, ticketIds []primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.ticketCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ticketIds}})
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

func (m *mongoRepository) GetTicketByPlayerId(ctx context.Context, playerId uuid.UUID) (*model.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ticket model.Ticket
	err := m.ticketCollection.FindOne(ctx, bson.M{"playerIds": playerId}).Decode(&ticket)
	if err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (m *mongoRepository) GetTicketsByGameMode(ctx context.Context, gameModeId string) ([]*model.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.ticketCollection.Find(ctx, bson.M{"gameModeId": gameModeId})
	if err != nil {
		return nil, err
	}

	var tickets []*model.Ticket
	err = cursor.All(ctx, &tickets)

	return tickets, err
}

func (m *mongoRepository) GetUnmatchedTicketsByGameMode(ctx context.Context, gameModeId string) ([]*model.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.ticketCollection.Find(ctx, bson.M{"gameModeId": gameModeId, "inPendingMatch": false})
	if err != nil {
		return nil, err
	}

	var tickets []*model.Ticket
	err = cursor.All(ctx, &tickets)
	if err != nil {
		return nil, err
	}

	return tickets, nil
}

func (m *mongoRepository) AddTicketDequeueRequest(ctx context.Context, ticketId primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.ticketCollection.UpdateOne(ctx, bson.M{"_id": ticketId}, bson.M{"$set": bson.M{"removals.markedForRemoval": true}})
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (m *mongoRepository) AddPlayerDequeueRequest(ctx context.Context, ticketId primitive.ObjectID, playerId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.ticketCollection.UpdateOne(ctx, bson.M{"_id": ticketId}, bson.M{"$addToSet": bson.M{"removals.playersForRemoval": playerId}})
	return err
}

func (m *mongoRepository) ResetAllDequeueRequestsById(ctx context.Context, ticketIds []primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.ticketCollection.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": ticketIds}}, bson.M{"$unset": bson.M{"removals": ""}})
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (m *mongoRepository) GetTicketsWithDequeueRequest(ctx context.Context, gameModeId string) ([]*model.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.ticketCollection.Find(ctx, bson.M{"gameModeId": gameModeId, "removals": bson.M{"$exists": true}})
	if err != nil {
		return nil, err
	}

	var tickets []*model.Ticket
	err = cursor.All(ctx, &tickets)
	if err != nil {
		return nil, err
	}

	return tickets, nil
}

func (m *mongoRepository) IsPartyQueued(ctx context.Context, partyId primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	count, err := m.ticketCollection.CountDocuments(ctx, bson.M{"partyId": partyId})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *mongoRepository) AddTicketDequeueRequestByPartyId(ctx context.Context, partyId primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"partyId": partyId}
	update := bson.M{"$set": bson.M{"removals.markedForRemoval": true}}

	result, err := m.ticketCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (m *mongoRepository) AddPlayerDequeueRequestByPartyId(ctx context.Context, partyId primitive.ObjectID, playerId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"partyId": partyId}
	update := bson.M{"$addToSet": bson.M{"removals.playersForRemoval": playerId}}

	_, err := m.ticketCollection.UpdateOne(ctx, filter, update)
	return err
}
