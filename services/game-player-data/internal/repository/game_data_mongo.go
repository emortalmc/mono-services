package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository/model"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/utils"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"time"
)

const (
	blockSumoCollection    = "blockSumo"
	towerDefenceCollection = "towerDefence"
	minesweeperCollection  = "minesweeper"
	marathonCollection     = "marathon"
)

var typeToCollection = map[reflect.Type]string{
	reflect.TypeOf(&model.BlockSumoData{}):    blockSumoCollection,
	reflect.TypeOf(&model.TowerDefenceData{}): towerDefenceCollection,
	reflect.TypeOf(&model.MinesweeperData{}):  minesweeperCollection,
	reflect.TypeOf(&model.MarathonData{}):     marathonCollection,
}

type mongoGameDataRepository[T model.GameData] struct {
	coll *mongo.Collection

	example T
}

func NewMongoGameDataRepository[T model.GameData](db *mongo.Database, example T) GameDataRepository[T] {
	collName, ok := typeToCollection[reflect.TypeOf(example)]
	if !ok {
		panic(fmt.Sprintf("no collection found for type %T", example))
	}

	return &mongoGameDataRepository[T]{
		coll: db.Collection(collName),
	}
}

func (m *mongoGameDataRepository[T]) Get(ctx context.Context, playerID uuid.UUID) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	example := reflect.New(reflect.TypeOf(m.example).Elem()).Interface().(T)

	res := m.coll.FindOne(ctx, bson.M{"_id": playerID})
	if err := res.Decode(&example); err != nil {
		return example, fmt.Errorf("failed to decode data: %w", err)
	}

	return example, nil
}

func (m *mongoGameDataRepository[T]) GetOrDefault(ctx context.Context, playerID uuid.UUID, defaultData T) (T, error) {
	res, err := m.Get(ctx, playerID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return defaultData, nil
		}

		return m.example, err
	}

	return res, nil
}

func (m *mongoGameDataRepository[T]) GetMultiple(ctx context.Context, playerIds []uuid.UUID) ([]T, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.coll.Find(ctx, bson.M{"_id": bson.M{"$in": playerIds}})
	if err != nil {
		return nil, err
	}

	exampleType := reflect.TypeOf(m.example)
	sliceType := reflect.SliceOf(exampleType)
	data := reflect.MakeSlice(sliceType, 0, 0).Interface().([]T)
	if err := cursor.All(ctx, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (m *mongoGameDataRepository[T]) Save(ctx context.Context, data T) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.coll.ReplaceOne(ctx, bson.M{"_id": data.PlayerID()}, data, &options.ReplaceOptions{Upsert: utils.PointerOf(true)})
	if err != nil {
		return fmt.Errorf("failed to insert data: %w", err)
	}

	return nil
}
