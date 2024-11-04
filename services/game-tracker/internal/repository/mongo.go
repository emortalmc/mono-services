package repository

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/config"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository/model"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository/registrytypes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	databaseName = "game-tracker"

	liveGameCollectionName     = "liveGame"
	historicGameCollectionName = "historicGame"
)

type mongoRepository struct {
	database *mongo.Database

	liveGameCollection     *mongo.Collection
	historicGameCollection *mongo.Collection
}

func NewMongoRepository(ctx context.Context, logger *zap.SugaredLogger, wg *sync.WaitGroup, cfg config.MongoDBConfig) (Repository, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetRegistry(registrytypes.CodecRegistry))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	database := client.Database(databaseName)
	repo := &mongoRepository{
		database:               database,
		liveGameCollection:     database.Collection(liveGameCollectionName),
		historicGameCollection: database.Collection(historicGameCollectionName),
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

// Note we create a lot more indexes for the game tracker because it's a debug data treasure trove.
// Not all indexes are used in code.
var (
	liveGameIndexes = []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "gameModeId", Value: 1}},
			Options: options.Index().SetName("gameModeId"),
		},
		{
			Keys:    bson.D{{Key: "serverId", Value: 1}},
			Options: options.Index().SetName("serverId"),
		},
	}
	historicGameIndexes = []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "gameModeId", Value: 1}},
			Options: options.Index().SetName("gameModeId"),
		},
		{
			Keys:    bson.D{{Key: "serverId", Value: 1}},
			Options: options.Index().SetName("serverId"),
		},
		{
			Keys:    bson.D{{Key: "startTime", Value: 1}},
			Options: options.Index().SetName("startTime"),
		},
		{
			Keys:    bson.D{{Key: "players.id", Value: 1}},
			Options: options.Index().SetName("players.id"),
		},
		{
			Keys:    bson.D{{Key: "endTime", Value: 1}},
			Options: options.Index().SetName("endTime"),
		},

		// todo we might want indexes for game data and winner data
	}
)

func (m *mongoRepository) createIndexes(ctx context.Context) {
	collIndexes := map[*mongo.Collection][]mongo.IndexModel{
		m.liveGameCollection:     liveGameIndexes,
		m.historicGameCollection: historicGameIndexes,
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
		return 0, fmt.Errorf("failed to create indexes: %w", err)
	}

	return len(result), nil
}

func (m *mongoRepository) GetLiveGame(ctx context.Context, id primitive.ObjectID) (*model.LiveGame, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var game model.LiveGame
	if err := m.liveGameCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&game); err != nil {
		return nil, fmt.Errorf("failed to get live game: %w", err)
	}

	if err := game.ParseGameData(); err != nil {
		return nil, fmt.Errorf("failed to parse game data: %w", err)
	}

	return &game, nil
}

var ErrIdNotSet = fmt.Errorf("id not set")

func (m *mongoRepository) SaveLiveGame(ctx context.Context, game *model.LiveGame) error {
	if game.Id.IsZero() {
		return ErrIdNotSet
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.liveGameCollection.UpdateByID(ctx, game.Id, bson.M{"$set": game}, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("failed to save live game: %w", err)
	}

	return nil
}

func (m *mongoRepository) DeleteLiveGame(ctx context.Context, id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.liveGameCollection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (m *mongoRepository) SaveHistoricGame(ctx context.Context, game *model.HistoricGame) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.historicGameCollection.InsertOne(ctx, game)
	return err
}

func (m *mongoRepository) GetHistoricGame(ctx context.Context, id primitive.ObjectID) (*model.HistoricGame, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var game model.HistoricGame
	if err := m.historicGameCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&game); err != nil {
		return nil, fmt.Errorf("failed to get historic game: %w", err)
	}

	if err := game.ParseGameData(); err != nil {
		return nil, fmt.Errorf("failed to parse game data: %w", err)
	}

	return &game, nil
}
