package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/party-manager/internal/config"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/registrytypes"
	"github.com/google/uuid"
	"github.com/zakshearman/go-grpc-health/pkg/health"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	databaseName = "party-manager"

	partyCollectionName         = "party"
	partyInviteCollectionName   = "partyInvite"
	partySettingsCollectionName = "partySettings"
	eventCollectionName         = "event"
)

var (
	ErrAlreadyInParty = errors.New("player already in party")
	ErrNotInParty     = errors.New("player not in party")
	ErrIdMustBeNil    = errors.New("id must be nil")
	ErrIdIsNil        = errors.New("id must not be nil")
	ErrAlreadyLeader  = errors.New("player already leader") // TODO this isn't handled yet
)

type MongoRepository struct {
	logger *zap.SugaredLogger

	database *mongo.Database

	partyCollection         *mongo.Collection
	partyInviteCollection   *mongo.Collection
	partySettingsCollection *mongo.Collection
	eventCollection         *mongo.Collection
}

func NewMongoRepository(ctx context.Context, logger *zap.SugaredLogger, wg *sync.WaitGroup, cfg config.MongoDBConfig) (*MongoRepository, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetRegistry(createCodecRegistry()))
	if err != nil {
		return nil, err
	}

	database := client.Database(databaseName)
	repo := &MongoRepository{
		logger: logger,

		database: database,

		partyCollection:         database.Collection(partyCollectionName),
		partyInviteCollection:   database.Collection(partyInviteCollectionName),
		partySettingsCollection: database.Collection(partySettingsCollectionName),
		eventCollection:         database.Collection(eventCollectionName),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err := client.Disconnect(ctx); err != nil {
			logger.Errorw("failed to disconnect from mongo", err)
		}
	}()

	repo.createIndexes(ctx)
	logger.Infow("created mongo indexes")

	return repo, nil
}

var (
	partyIndexes = []mongo.IndexModel{
		{
			Keys:    bson.M{"leaderId": 1},
			Options: options.Index().SetName("leaderId").SetUnique(true),
		},
		{
			Keys:    bson.M{"members.playerId": 1},
			Options: options.Index().SetName("members_playerId").SetUnique(true),
		},
	}

	partyInviteIndexes = []mongo.IndexModel{
		{
			Keys:    bson.M{"partyId": 1},
			Options: options.Index().SetName("partyId"),
		},
		{
			Keys:    bson.D{{Key: "partyId", Value: 1}, {Key: "targetId", Value: 1}},
			Options: options.Index().SetName("partyId_targetId").SetUnique(true),
		},
		{
			Keys:    bson.M{"expiresAt": 1},
			Options: options.Index().SetName("expiresAt_TTL").SetExpireAfterSeconds(0),
		},
	}
)

func (m *MongoRepository) createIndexes(ctx context.Context) {
	collIndexes := map[*mongo.Collection][]mongo.IndexModel{
		m.partyCollection:       partyIndexes,
		m.partyInviteCollection: partyInviteIndexes,
	}

	wg := sync.WaitGroup{}
	wg.Add(len(collIndexes))

	for coll, indexes := range collIndexes {
		go func(coll *mongo.Collection, indexes []mongo.IndexModel) {
			defer wg.Done()
			_, err := m.createCollIndexes(ctx, coll, indexes)
			if err != nil {
				panic(fmt.Sprintf("failed to create indexes for collection %s: %s", coll.Name(), err))
			}
		}(coll, indexes)
	}

	wg.Wait()
}

func (m *MongoRepository) createCollIndexes(ctx context.Context, coll *mongo.Collection, indexes []mongo.IndexModel) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return 0, err
	}

	return len(result), nil
}

func (m *MongoRepository) HealthCheck(ctx context.Context) health.HealthStatus {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.database.Client().Ping(ctx, nil)
	if err != nil {
		m.logger.Errorw("failed to health check repository", err)
		return health.HealthStatusUnhealthy
	}
	return health.HealthStatusHealthy
}

func (m *MongoRepository) IsInParty(ctx context.Context, playerId uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	count, err := m.partyCollection.CountDocuments(ctx, bson.M{"members.playerId": playerId})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *MongoRepository) CreateParty(ctx context.Context, party *model.Party) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if party.ID != primitive.NilObjectID {
		return ErrIdMustBeNil
	}
	party.ID = primitive.NewObjectID()

	_, err := m.partyCollection.InsertOne(ctx, party)
	if err != nil {
		return err
	}

	return nil
}

func (m *MongoRepository) SetPartyMembers(ctx context.Context, partyId primitive.ObjectID, members []*model.PartyMember) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$set": bson.M{"members": members}})
	return err
}

func (m *MongoRepository) DeleteParty(ctx context.Context, partyId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.partyCollection.FindOneAndDelete(ctx, bson.M{"_id": partyId}).Err()
	return err
}

func (m *MongoRepository) AddPartyMember(ctx context.Context, partyId primitive.ObjectID, member *model.PartyMember) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$addToSet": bson.M{"members": member}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	if res.ModifiedCount == 0 {
		return ErrAlreadyInParty
	}

	return nil
}

func (m *MongoRepository) SetPartyLeader(ctx context.Context, partyId primitive.ObjectID, leaderId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$set": bson.M{"leaderId": leaderId}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	if res.ModifiedCount == 0 {
		return ErrAlreadyLeader
	}

	return nil
}

func (m *MongoRepository) SetPartyOpen(ctx context.Context, partyId primitive.ObjectID, open bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$set": bson.M{"open": open}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *MongoRepository) SetPartyEventID(ctx context.Context, partyId primitive.ObjectID, eventId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$set": bson.M{"eventId": eventId}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *MongoRepository) RemovePartyEventID(ctx context.Context, partyId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$unset": bson.M{"eventId": ""}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *MongoRepository) GetPartyByID(ctx context.Context, partyId primitive.ObjectID) (*model.Party, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var party model.Party
	err := m.partyCollection.FindOne(ctx, bson.M{"_id": partyId}).Decode(&party)
	if err != nil {
		return nil, err
	}

	return &party, nil
}

func (m *MongoRepository) GetPartyByMemberID(ctx context.Context, playerId uuid.UUID) (*model.Party, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var party model.Party
	err := m.partyCollection.FindOne(ctx, bson.M{"members.playerId": playerId}).Decode(&party)
	if err != nil {
		return nil, err
	}

	return &party, nil
}

func (m *MongoRepository) GetPartyIdByMemberId(ctx context.Context, playerId uuid.UUID) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get only the id of the party
	var result struct {
		Id primitive.ObjectID `bson:"_id"`
	}
	err := m.partyCollection.FindOne(ctx, bson.M{"members.playerId": playerId}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&result)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return result.Id, nil
}

func (m *MongoRepository) GetPartyLeaderIdByMemberId(ctx context.Context, playerId uuid.UUID) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get only the leaderId of the party
	var result struct {
		LeaderId uuid.UUID `bson:"leaderId"`
	}
	err := m.partyCollection.FindOne(ctx, bson.M{"members.playerId": playerId}, options.FindOne().SetProjection(bson.M{"leaderId": 1})).Decode(&result)
	if err != nil {
		return uuid.Nil, err
	}

	return result.LeaderId, nil
}

func (m *MongoRepository) GetPartyLeaderByPartyId(ctx context.Context, partyId primitive.ObjectID) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get only the leaderId of the party
	var result struct {
		LeaderId uuid.UUID `bson:"leaderId"`
	}
	err := m.partyCollection.FindOne(ctx, bson.M{"_id": partyId}, options.FindOne().SetProjection(bson.M{"leaderId": 1})).Decode(&result)
	if err != nil {
		return uuid.Nil, err
	}

	return result.LeaderId, nil
}

func (m *MongoRepository) RemoveMemberFromParty(ctx context.Context, partyId primitive.ObjectID, playerId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateByID(ctx, partyId, bson.M{"$pull": bson.M{"members": bson.M{"playerId": playerId}}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	if res.ModifiedCount == 0 {
		return ErrNotInParty
	}

	return nil
}

func (m *MongoRepository) RemoveMemberFromSelfParty(ctx context.Context, playerId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.partyCollection.UpdateOne(ctx, bson.M{"members.playerId": playerId}, bson.M{"$pull": bson.M{"members": bson.M{"playerId": playerId}}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *MongoRepository) CreatePartyInvite(ctx context.Context, invite *model.PartyInvite) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if invite.ID != primitive.NilObjectID {
		return ErrIdMustBeNil
	}
	invite.ID = primitive.NewObjectID()

	_, err := m.partyInviteCollection.InsertOne(ctx, invite)
	if err != nil {
		return err
	}

	return nil
}

func (m *MongoRepository) DeletePartyInvite(ctx context.Context, partyId primitive.ObjectID, targetId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.partyInviteCollection.DeleteOne(ctx, bson.M{"partyId": partyId, "targetId": targetId})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *MongoRepository) DeletePartyInvitesByPartyId(ctx context.Context, partyId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.partyInviteCollection.DeleteMany(ctx, bson.M{"partyId": partyId})
	return err
}

func (m *MongoRepository) GetPartyInvitesByPartyId(ctx context.Context, partyId primitive.ObjectID) ([]*model.PartyInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var invites []*model.PartyInvite
	cursor, err := m.partyInviteCollection.Find(ctx, bson.M{"partyId": partyId})
	if err != nil {
		return nil, err
	}

	err = cursor.All(ctx, &invites)
	if err != nil {
		return nil, err
	}

	return invites, nil
}

func (m *MongoRepository) DoesPartyInviteExist(ctx context.Context, partyId primitive.ObjectID, playerId uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	count, err := m.partyInviteCollection.CountDocuments(ctx, bson.M{"partyId": partyId, "targetId": playerId})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *MongoRepository) GetPartySettings(ctx context.Context, playerId uuid.UUID) (*model.PartySettings, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var settings model.PartySettings
	err := m.partySettingsCollection.FindOne(ctx, bson.M{"_id": playerId}).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (m *MongoRepository) UpdatePartySettings(ctx context.Context, settings *model.PartySettings) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.partySettingsCollection.ReplaceOne(ctx, bson.M{"_id": settings.PlayerID}, settings, options.Replace().SetUpsert(true))
	if err != nil {
		return err
	}

	// NOTE: We don't care if the document was modified or not, because the settings are the same
	return nil
}

func (m *MongoRepository) CreateEvent(ctx context.Context, event *model.Event) error {
	if event.ID == "" {
		return ErrIdIsNil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.eventCollection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

var ErrNoUpdateParams = errors.New("no update parameters")

func (m *MongoRepository) UpdateEvent(ctx context.Context, eventId string, displayTime *time.Time, startTime *time.Time) (*model.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{}
	if displayTime != nil {
		update["displayTime"] = displayTime
	}
	if startTime != nil {
		update["startTime"] = startTime
	}

	if len(update) == 0 {
		return nil, ErrNoUpdateParams
	}

	res := m.eventCollection.FindOneAndUpdate(ctx, bson.M{"_id": eventId}, bson.M{"$set": update})
	if res.Err() != nil {
		return nil, fmt.Errorf("failed to update event: %w", res.Err())
	}

	var event model.Event
	if err := res.Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to decode updated event: %w", err)
	}

	return &event, nil
}

func (m *MongoRepository) DeleteEvent(ctx context.Context, eventId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.eventCollection.DeleteOne(ctx, bson.M{"_id": eventId})
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (m *MongoRepository) GetEventByID(ctx context.Context, eventId string) (*model.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var event model.Event
	if err := m.eventCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to get event by id: %w", err)
	}

	return &event, nil
}

func (m *MongoRepository) GetLiveEvent(ctx context.Context) (*model.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var event model.Event
	if err := m.eventCollection.FindOne(ctx, bson.M{"started": true}).Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to get live event: %w", err)
	}

	return &event, nil
}

func (m *MongoRepository) ListEvents(ctx context.Context) ([]*model.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.eventCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []*model.Event
	err = cursor.All(ctx, &events)
	if err != nil {
		return nil, fmt.Errorf("failed to extract listed events: %w", err)
	}

	return events, nil
}

func (m *MongoRepository) SetEventPartyID(ctx context.Context, eventId string, partyId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.eventCollection.UpdateByID(ctx, eventId, bson.M{"$set": bson.M{"partyId": partyId}})
	if err != nil {
		return fmt.Errorf("failed to set event party id: %w", err)
	}

	return nil
}

func (m *MongoRepository) GetEventToDisplay(ctx context.Context) (*model.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"$and": []bson.M{{"displayTime": bson.M{"$lte": time.Now()}}, {"displayed": bson.M{"$exists": false}}}}
	res := m.eventCollection.FindOneAndUpdate(ctx, filter, bson.M{"$set": bson.M{"displayed": true}})
	if res.Err() != nil {
		return nil, fmt.Errorf("failed to get event to display: %w", res.Err())
	}

	var event model.Event
	if err := res.Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to decode event to display: %w", err)
	}

	return &event, nil
}

func (m *MongoRepository) GetEventToStart(ctx context.Context) (*model.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"$and": []bson.M{{"startTime": bson.M{"$lte": time.Now()}}, {"started": bson.M{"$exists": false}}}}
	res := m.eventCollection.FindOneAndUpdate(ctx, filter, bson.M{"$set": bson.M{"started": true}})
	if res.Err() != nil {
		return nil, fmt.Errorf("failed to get event to start: %w", res.Err())
	}

	var event model.Event
	if err := res.Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to decode event to start: %w", err)
	}

	return &event, nil
}

func createCodecRegistry() *bsoncodec.Registry {
	r := bson.NewRegistry()

	r.RegisterTypeEncoder(registrytypes.UUIDType, bsoncodec.ValueEncoderFunc(registrytypes.UuidEncodeValue))
	r.RegisterTypeDecoder(registrytypes.UUIDType, bsoncodec.ValueDecoderFunc(registrytypes.UuidDecodeValue))

	return r
}
