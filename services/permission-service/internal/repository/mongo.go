package repository

import (
	"context"
	"errors"
	"github.com/emortalmc/mono-services/services/permission-service/internal/config"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/model"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/registrytypes"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	databaseName         = "permission-service"
	roleCollectionName   = "roles"
	playerCollectionName = "players"
)

type mongoRepository struct {
	Repository
	database *mongo.Database

	roleCollection   *mongo.Collection
	playerCollection *mongo.Collection
}

var (
	ErrAlreadyHasRole  = errors.New("player already has testRole")
	ErrDoesNotHaveRole = errors.New("player does not have testRole")
)

func NewMongoRepository(ctx context.Context, logger *zap.SugaredLogger, wg *sync.WaitGroup, cfg config.MongoDBConfig) (Repository, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetRegistry(createCodecRegistry()))
	if err != nil {
		return nil, err
	}

	database := client.Database(databaseName)
	repo := &mongoRepository{
		database:         database,
		roleCollection:   database.Collection(roleCollectionName),
		playerCollection: database.Collection(playerCollectionName),
	}

	err = repo.createDefaultRole(ctx)
	if err != nil {
		return nil, err
	}

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

func (m *mongoRepository) createDefaultRole(ctx context.Context) error {
	displayName := "{{.Username}}"
	role := &model.Role{
		Id:          model.DefaultRoleId,
		Priority:    0,
		DisplayName: &displayName,
		Permissions: make([]model.PermissionNode, 0),
	}

	// Create default role if it doesn't already exist
	_, err := m.roleCollection.UpdateOne(ctx, &bson.M{"_id": model.DefaultRoleId}, &bson.M{"$setOnInsert": &role}, options.Update().SetUpsert(true))
	return err
}

func (m *mongoRepository) GetAllRoles(ctx context.Context) ([]*model.Role, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.roleCollection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	var mongoResult []model.Role
	err = cursor.All(ctx, &mongoResult)

	slice := make([]*model.Role, len(mongoResult))
	for i := range mongoResult {
		slice[i] = &mongoResult[i]
	}

	return slice, err
}

func (m *mongoRepository) GetRole(ctx context.Context, roleId string) (*model.Role, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": roleId}

	var result *model.Role
	err := m.roleCollection.FindOne(ctx, filter).Decode(&result)

	return result, err
}

func (m *mongoRepository) DoesRoleExist(ctx context.Context, roleId string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": roleId}

	count, err := m.roleCollection.CountDocuments(ctx, filter)

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *mongoRepository) CreateRole(ctx context.Context, role *model.Role) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.roleCollection.InsertOne(ctx, role)
	return err
}

func (m *mongoRepository) UpdateRole(ctx context.Context, role *model.Role) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result := m.roleCollection.FindOneAndReplace(ctx, bson.M{"_id": role.Id}, role)
	return result.Err()
}

func (m *mongoRepository) GetPlayerRoleIds(ctx context.Context, playerId uuid.UUID) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": playerId}

	var result *model.Player
	err := m.playerCollection.FindOne(ctx, filter).Decode(&result)

	if err != nil {
		// insert into db if not exists
		if err == mongo.ErrNoDocuments {
			_, err := m.playerCollection.InsertOne(ctx, model.Player{Id: playerId, Roles: []string{model.DefaultRoleId}})

			if err != nil {
				return nil, err
			}

			return []string{model.DefaultRoleId}, nil
		} else {
			return nil, err
		}
	}

	return result.Roles, err
}

func (m *mongoRepository) AddRoleToPlayer(ctx context.Context, playerId uuid.UUID, roleId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{"$addToSet": bson.M{"roles": roleId}})

	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		if result.MatchedCount == 0 {
			// insert into db if not exists
			_, err = m.playerCollection.InsertOne(ctx, model.Player{Id: playerId, Roles: []string{model.DefaultRoleId, roleId}})
			if err != nil {
				return err
			}
			return nil
		}
		return ErrAlreadyHasRole
	}

	return err
}

func (m *mongoRepository) RemoveRoleFromPlayer(ctx context.Context, playerId uuid.UUID, roleId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{"$pull": bson.M{"roles": roleId}})

	if err != nil {
		return err
	}
	if result.ModifiedCount == 0 {
		return ErrDoesNotHaveRole
	}

	return err
}

func createCodecRegistry() *bsoncodec.Registry {
	r := bson.NewRegistry()

	r.RegisterTypeEncoder(registrytypes.UUIDType, bsoncodec.ValueEncoderFunc(registrytypes.UuidEncodeValue))
	r.RegisterTypeDecoder(registrytypes.UUIDType, bsoncodec.ValueDecoderFunc(registrytypes.UuidDecodeValue))

	return r
}
