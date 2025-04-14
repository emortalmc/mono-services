package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/config"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/repository/model"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/repository/registrytypes"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"sync"
	"time"
)

type mongoRepository struct {
	logger *zap.SugaredLogger

	database *mongo.Database

	friendColl        *mongo.Collection
	pendingFriendColl *mongo.Collection
	blockColl         *mongo.Collection
}

var (
	ErrNotFriends      = errors.New("players are not friends")
	ErrNoFriendRequest = errors.New("no friend request found")
)

func NewMongoRepository(ctx context.Context, logger *zap.SugaredLogger, wg *sync.WaitGroup, cfg *config.MongoDBConfig) (Repository, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetRegistry(createCodecRegistry()))
	if err != nil {
		return nil, err
	}

	database := client.Database("relationship-manager")

	repo := &mongoRepository{
		logger: logger,

		database: database,

		friendColl:        database.Collection("friend"),
		pendingFriendColl: database.Collection("pendingFriend"),
		blockColl:         database.Collection("block"),
	}

	repo.createIndexes(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err := client.Disconnect(ctx); err != nil {
			logger.Errorw("failed to disconnect from mongo", err)
		}
	}()

	return repo, nil
}

var (
	friendIndexes = []mongo.IndexModel{
		{ // Combined friend index
			Keys:    bson.D{{Key: "playerOneId", Value: 1}, {Key: "playerTwoId", Value: 1}},
			Options: options.Index().SetName("playerOneId_playerTwoId").SetUnique(true),
		},
		// Singular friend indexes
		{
			Keys:    bson.M{"playerOneId": 1},
			Options: options.Index().SetName("playerOneId"),
		},
		{
			Keys:    bson.M{"playerTwoId": 1},
			Options: options.Index().SetName("playerTwoId"),
		},
	}

	pendingFriendIndexes = []mongo.IndexModel{
		{ // Combined pending friend index
			Keys:    bson.D{{Key: "requesterId", Value: 1}, {Key: "targetId", Value: 1}},
			Options: options.Index().SetName("requesterId_targetId").SetUnique(true),
		},
		// Singular pending friend indexes
		{
			Keys:    bson.M{"requesterId": 1},
			Options: options.Index().SetName("requesterId"),
		},
		{
			Keys:    bson.M{"targetId": 1},
			Options: options.Index().SetName("targetId"),
		},
	}

	blockIndexes = []mongo.IndexModel{
		{ // Combined block index
			Keys:    bson.D{{Key: "blockerId", Value: 1}, {Key: "blockedId", Value: 1}},
			Options: options.Index().SetName("blockerId_blockedId").SetUnique(true),
		},
		// Singular block indexes
		{
			Keys:    bson.M{"blockerId": 1},
			Options: options.Index().SetName("blockerId"),
		},
		{
			Keys:    bson.M{"blockedId": 1},
			Options: options.Index().SetName("blockedId"),
		},
	}
)

func (m *mongoRepository) createIndexes(ctx context.Context) {
	collIndexes := map[*mongo.Collection][]mongo.IndexModel{
		m.friendColl:        friendIndexes,
		m.pendingFriendColl: pendingFriendIndexes,
		m.blockColl:         blockIndexes,
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

func (m *mongoRepository) createCollIndexes(ctx context.Context, coll *mongo.Collection, indexes []mongo.IndexModel) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return 0, err
	}

	return len(result), nil
}

func (m *mongoRepository) CreateFriendConnection(ctx context.Context, conn *model.FriendConnection) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.friendColl.InsertOne(ctx, conn)
	return err
}

func (m *mongoRepository) GetFriendConnections(ctx context.Context, playerId uuid.UUID) ([]*model.FriendConnection, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.friendColl.Find(ctx, bson.M{"$or": []interface{}{
		bson.M{"playerOneId": playerId},
		bson.M{"playerTwoId": playerId},
	}})
	if err != nil {
		return nil, err
	}

	var mongoResult []*model.FriendConnection
	err = cursor.All(ctx, &mongoResult)

	return mongoResult, err
}

func (m *mongoRepository) DeleteFriendConnection(ctx context.Context, playerId uuid.UUID, friendId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.friendColl.DeleteOne(ctx, bson.M{"$or": []interface{}{
		bson.M{"playerOneId": playerId, "playerTwoId": friendId},
		bson.M{"playerOneId": friendId, "playerTwoId": playerId},
	}})

	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFriends
	}
	return nil
}

func (m *mongoRepository) AreFriends(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.friendColl.CountDocuments(ctx, bson.M{"$or": []bson.M{
		{"playerOneId": playerOneId, "playerTwoId": playerTwoId},
		{"playerOneId": playerTwoId, "playerTwoId": playerOneId},
	}})
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (m *mongoRepository) CreatePendingFriendConnection(ctx context.Context, conn *model.PendingFriendConnection) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.pendingFriendColl.InsertOne(ctx, conn)
	// todo NOTE: already exists is mongo.IsDuplicateKeyError(err)
	return err
}

func (m *mongoRepository) DoesPendingFriendConnectionExist(ctx context.Context, requesterId uuid.UUID, targetId uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.pendingFriendColl.CountDocuments(ctx, bson.M{"requesterId": requesterId, "targetId": targetId})
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (m *mongoRepository) GetPendingFriendConnections(ctx context.Context, playerId uuid.UUID, opts DirectionOpts) ([]*model.PendingFriendConnection, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var orFilters = make([]bson.M, 0)
	if opts.Incoming {
		orFilters = append(orFilters, bson.M{"targetId": playerId})
	}
	if opts.Outgoing {
		orFilters = append(orFilters, bson.M{"requesterId": playerId})
	}

	if len(orFilters) == 0 {
		m.logger.Warnw("no direction options provided", "playerId", playerId)
		return nil, errors.New("no direction options provided")
	}

	cursor, err := m.pendingFriendColl.Find(ctx, bson.M{"$or": orFilters}, options.Find().SetSort(bson.M{"_id": -1}))
	if err != nil {
		return nil, err
	}

	var mongoResult []*model.PendingFriendConnection
	err = cursor.All(ctx, &mongoResult)
	return mongoResult, err
}

func (m *mongoRepository) DeletePendingFriendConnection(ctx context.Context, playerId uuid.UUID, friendId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.pendingFriendColl.DeleteOne(ctx, bson.M{"$or": []bson.M{
		{"requesterId": playerId, "targetId": friendId},
		{"requesterId": friendId, "targetId": playerId},
	}})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNoFriendRequest
	}
	return nil
}

func (m *mongoRepository) DeletePendingFriendConnections(ctx context.Context, playerId uuid.UUID, opts *DirectionOpts) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var orFilters = make([]bson.M, 0)
	if opts.Incoming {
		orFilters = append(orFilters, bson.M{"targetId": playerId})
	}
	if opts.Outgoing {
		orFilters = append(orFilters, bson.M{"requesterId": playerId})
	}

	if len(orFilters) == 0 {
		m.logger.Warnw("no direction options provided", "playerId", playerId)
		return 0, nil
	}

	result, err := m.pendingFriendColl.DeleteMany(ctx, bson.M{"$or": orFilters})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func createCodecRegistry() *bsoncodec.Registry {
	r := bson.NewRegistry()

	r.RegisterTypeEncoder(registrytypes.UUIDType, bsoncodec.ValueEncoderFunc(registrytypes.UuidEncodeValue))
	r.RegisterTypeDecoder(registrytypes.UUIDType, bsoncodec.ValueDecoderFunc(registrytypes.UuidDecodeValue))

	return r
}
