package repository

// TODO make tests not depend on other repository methods.
// Where they do, switch to direct mongo calls.
import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/emortalmc/mono-services/services/permission-service/internal/config"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/model"
	"github.com/emortalmc/mono-services/services/permission-service/internal/utils"
	"github.com/emortalmc/proto-specs/gen/go/model/permission"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	mongoDb "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	mongoUri = "mongodb://root:password@localhost:%s"
)

var (
	dbClient *mongoDb.Client
	database *mongoDb.Database
	repo     Repository
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not constuct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "6.0.3",
		Env: []string{
			"MONGO_INITDB_ROOT_USERNAME=root",
			"MONGO_INITDB_ROOT_PASSWORD=password",
		},
	}, func(cfg *docker.HostConfig) {
		cfg.AutoRemove = true
		cfg.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("could not start resource: %s", err)
	}

	uri := fmt.Sprintf(mongoUri, resource.GetPort("27017/tcp"))

	err = pool.Retry(func() (err error) {
		dbClient, err = mongoDb.Connect(context.Background(), options.Client().ApplyURI(uri).SetRegistry(createCodecRegistry()))
		if err != nil {
			return
		}
		err = dbClient.Ping(context.Background(), nil)
		if err != nil {
			return
		}

		ctx := context.Background()
		unsugared, _ := zap.NewDevelopment()
		logger := unsugared.Sugar()

		// Ping was successful, let's create the mongo repo
		repo, err = NewMongoRepository(ctx, logger, &sync.WaitGroup{}, config.MongoDBConfig{URI: uri})
		database = dbClient.Database(databaseName)
		return
	})

	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Exit(code)
}

var testRole = model.Role{
	Id:          "test1",
	Priority:    10,
	DisplayName: utils.PointerOf("testName"),
	Permissions: []model.PermissionNode{
		{
			Node:  "test1",
			State: permission.PermissionNode_DENY,
		},
	},
}

// minimumRole doesn't include displayName
var testMinimumRole = model.Role{
	Id:       "test2",
	Priority: 10,
	Permissions: []model.PermissionNode{
		{
			Node:  "test2",
			State: permission.PermissionNode_ALLOW,
		},
	},
}

var testUserIds = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

// todo this test fails since the default role is created in the mongo repository
func TestMongoRepository_GetRoles(t *testing.T) {
	// Setup
	many, err := database.Collection(roleCollectionName).InsertMany(context.Background(), []interface{}{testRole, testMinimumRole})
	assert.NoError(t, err)
	assert.Len(t, many.InsertedIDs, 2)

	// Test
	roles, err := repo.GetAllRoles(context.Background())
	assert.NoError(t, err)
	assert.Len(t, roles, 2)
	for _, role := range roles {
		assert.NotEmpty(t, role.Id)
		assert.NotEmpty(t, role.Priority)
		assert.NotEmpty(t, role.Permissions)

		valRole := *role
		switch role.Id {
		case testRole.Id:
			assert.Equal(t, testRole, valRole)
		case testMinimumRole.Id:
			assert.Equal(t, testMinimumRole, valRole)
		default:
			t.Errorf("unexpected role: %v", valRole)
		}
	}

	cleanup()

	roles, err = repo.GetAllRoles(context.Background())
	assert.NoError(t, err)
	assert.Len(t, roles, 0)
}

func TestMongoRepository_GetRole(t *testing.T) {
	// Setup
	many, err := database.Collection(roleCollectionName).InsertMany(context.Background(), []interface{}{testRole, testMinimumRole})
	assert.NoError(t, err)
	assert.Len(t, many.InsertedIDs, 2)

	// Test
	role, err := repo.GetRole(context.Background(), testRole.Id)
	assert.NoError(t, err)
	assert.Equal(t, testRole, *role)

	role, err = repo.GetRole(context.Background(), testMinimumRole.Id)
	assert.NoError(t, err)
	assert.Equal(t, testMinimumRole, *role)

	cleanup()

	role, err = repo.GetRole(context.Background(), testRole.Id)
	assert.Equal(t, mongoDb.ErrNoDocuments, err)
	assert.Nil(t, role)
}

func TestMongoRepository_DoesRoleExist(t *testing.T) {
	// Setup
	many, err := database.Collection(roleCollectionName).InsertMany(context.Background(), []interface{}{testRole, testMinimumRole})
	assert.NoError(t, err)
	assert.Len(t, many.InsertedIDs, 2)

	// Test
	exists, err := repo.DoesRoleExist(context.Background(), testRole.Id)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.DoesRoleExist(context.Background(), testMinimumRole.Id)
	assert.NoError(t, err)
	assert.True(t, exists)

	cleanup()

	exists, err = repo.DoesRoleExist(context.Background(), testRole.Id)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestMongoRepository_CreateRole(t *testing.T) {
	// Test
	err := repo.CreateRole(context.Background(), &testRole)
	assert.NoError(t, err)

	// Verify
	role, err := repo.GetRole(context.Background(), testRole.Id)
	assert.NoError(t, err)
	assert.Equal(t, testRole, *role)

	cleanup()

	// Test that duplicate documents error
	// Setup
	_, err = database.Collection(roleCollectionName).InsertOne(context.Background(), testRole)
	assert.NoError(t, err)

	// Create a duplicate role
	err = repo.CreateRole(context.Background(), &testRole)
	dupeErr := mongoDb.IsDuplicateKeyError(err)
	assert.True(t, dupeErr)

	cleanup()
}

func TestMongoRepository_UpdateRole(t *testing.T) {
	// Setup
	_, err := database.Collection(roleCollectionName).InsertOne(context.Background(), testRole)
	assert.NoError(t, err)

	// Test
	tempRole := testMinimumRole
	tempRole.Id = testRole.Id // Ensure the same ID when updating

	err = repo.UpdateRole(context.Background(), &tempRole)
	assert.NoError(t, err)

	// Verify
	role, err := repo.GetRole(context.Background(), tempRole.Id)
	assert.NoError(t, err)
	assert.Equal(t, tempRole, *role)

	cleanup()

	err = repo.UpdateRole(context.Background(), &testRole)
	assert.Equal(t, mongoDb.ErrNoDocuments, err)
}

func TestMongoRepository_GetPlayerRoleIds(t *testing.T) {
	// Test default behaviour when user is not present
	roleIds, err := repo.GetPlayerRoleIds(context.Background(), testUserIds[0])
	assert.NoError(t, err)
	assert.Len(t, roleIds, 1)
	assert.Equal(t, model.DefaultRoleId, roleIds[0])

	// Test that return is still the same as default should persist the user
	roleIds, err = repo.GetPlayerRoleIds(context.Background(), testUserIds[0])
	assert.NoError(t, err)
	assert.Len(t, roleIds, 1)
	assert.Equal(t, model.DefaultRoleId, roleIds[0])

	cleanup()

	_, err = database.Collection(playerCollectionName).InsertOne(context.Background(), model.Player{
		Id:    testUserIds[1],
		Roles: []string{"default", "test1"},
	})
	assert.NoError(t, err)

	roleIds, err = repo.GetPlayerRoleIds(context.Background(), testUserIds[1])

	assert.NoError(t, err)
	assert.Len(t, roleIds, 2)
	assert.Contains(t, roleIds, "default")
	assert.Contains(t, roleIds, "test1")

	cleanup()
}

func TestMongoRepository_AddRoleToPlayer(t *testing.T) {
	// Test when the user does not exist. A default user with the additional role should be created.
	err := repo.AddRoleToPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.NoError(t, err)

	// Verify
	roleIds, err := repo.GetPlayerRoleIds(context.Background(), testUserIds[0])
	assert.NoError(t, err)
	assert.Len(t, roleIds, 2)
	assert.Contains(t, roleIds, model.DefaultRoleId)
	assert.Contains(t, roleIds, testRole.Id)

	cleanup()

	_, err = database.Collection(playerCollectionName).InsertOne(context.Background(), model.Player{
		Id:    testUserIds[0],
		Roles: []string{model.DefaultRoleId},
	})
	assert.NoError(t, err)

	// Test a valid case with a default user
	err = repo.AddRoleToPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.NoError(t, err)

	// Verify
	roleIds, err = repo.GetPlayerRoleIds(context.Background(), testUserIds[0])
	assert.NoError(t, err)
	assert.Len(t, roleIds, 2)
	assert.Contains(t, roleIds, model.DefaultRoleId)
	assert.Contains(t, roleIds, testRole.Id)

	// Test that duplicates error, so no cleanup is done.
	err = repo.AddRoleToPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.Equal(t, ErrAlreadyHasRole, err)

	cleanup()
}

// Test when user doesn't yet exist
func TestMongoRepository_AddRoleToPlayer2(t *testing.T) {
	err := repo.AddRoleToPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.NoError(t, err)

	// Verify
	roleIds, err := repo.GetPlayerRoleIds(context.Background(), testUserIds[0])
	assert.NoError(t, err)
	assert.Len(t, roleIds, 2)
	assert.Contains(t, roleIds, model.DefaultRoleId)
	assert.Contains(t, roleIds, testRole.Id)

	cleanup()
}

func TestMongoRepository_RemoveRoleFromPlayer(t *testing.T) {
	// Test when the user does not exist. ErrDoesNotHaveRole should be returned.
	err := repo.RemoveRoleFromPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.Equal(t, ErrDoesNotHaveRole, err)

	// Test a valid case with a default user
	_, err = database.Collection(playerCollectionName).InsertOne(context.Background(), model.Player{
		Id:    testUserIds[0],
		Roles: []string{model.DefaultRoleId, testRole.Id},
	})
	assert.NoError(t, err)

	err = repo.RemoveRoleFromPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.NoError(t, err)

	// Verify
	roleIds, err := repo.GetPlayerRoleIds(context.Background(), testUserIds[0])
	assert.NoError(t, err)
	assert.Len(t, roleIds, 1)
	assert.Contains(t, roleIds, model.DefaultRoleId)

	cleanup()

	// Test with an existing user that has only the default role
	_, err = database.Collection(playerCollectionName).InsertOne(context.Background(), model.Player{
		Id:    testUserIds[0],
		Roles: []string{model.DefaultRoleId},
	})
	assert.NoError(t, err)

	err = repo.RemoveRoleFromPlayer(context.Background(), testUserIds[0], testRole.Id)
	assert.Equal(t, ErrDoesNotHaveRole, err)

	cleanup()
}

func cleanup() {
	if err := database.Drop(context.Background()); err != nil {
		log.Panicf("could not drop database: %s", err)
	}
}
