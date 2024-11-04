package repository

//
//import (
//	"context"
//	"fmt"
//	"github.com/google/uuid"
//	"github.com/ory/dockertest/v3"
//	"github.com/ory/dockertest/v3/docker"
//	"github.com/stretchr/testify/assert"
//	"go.mongodb.org/mongo-driver/bson"
//	"go.mongodb.org/mongo-driver/bson/primitive"
//	"go.mongodb.org/mongo-driver/mongo"
//	"go.mongodb.org/mongo-driver/mongo/options"
//	"go.uber.org/zap"
//	"log"
//	"os"
//	"github.com/emortalmc/mono-services/services/party-manager/internal/config"
//	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
//	"sync"
//	"testing"
//	"time"
//)
//
//const (
//	mongoUri = "mongodb://root:password@localhost:%s"
//)
//
//var (
//	dbClient *mongo.Client
//	database *mongo.Database
//	repo     Repository
//)
//
//// TODO some of the tests here don't check the database after operations are performed.
//// TODO This should be done.
//
//func TestMain(m *testing.M) {
//	pool, err := dockertest.NewPool("")
//	if err != nil {
//		log.Fatalf("could not constuct pool: %s", err)
//	}
//
//	err = pool.Client.Ping()
//	if err != nil {
//		log.Fatalf("could not connect to docker: %s", err)
//	}
//
//	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
//		Repository: "mongo",
//		Tag:        "6.0.5",
//		Env: []string{
//			"MONGO_INITDB_ROOT_USERNAME=root",
//			"MONGO_INITDB_ROOT_PASSWORD=password",
//		},
//	}, func(cfg *docker.HostConfig) {
//		cfg.AutoRemove = true
//		cfg.RestartPolicy = docker.RestartPolicy{
//			Name: "no",
//		}
//	})
//	if err != nil {
//		log.Fatalf("could not start resource: %s", err)
//	}
//
//	unsugaredLogger, err := zap.NewDevelopment()
//	if err != nil {
//		log.Fatalf("could not create logger: %s", err)
//	}
//	logger := unsugaredLogger.Sugar()
//
//	uri := fmt.Sprintf(mongoUri, resource.GetPort("27017/tcp"))
//
//	err = pool.Retry(func() (err error) {
//		dbClient, err = mongo.Connect(context.Background(), options.Client().ApplyURI(uri).SetRegistry(createCodecRegistry()))
//		if err != nil {
//			return
//		}
//		err = dbClient.Ping(context.Background(), nil)
//		if err != nil {
//			return
//		}
//
//		// Ping was successful, let's create the mongo repo
//		repo, err = NewMongoRepository(context.Background(), logger, &sync.WaitGroup{}, &config.MongoDBConfig{URI: uri})
//		database = dbClient.Database(databaseName)
//
//		return
//	})
//
//	if err != nil {
//		log.Fatalf("could not connect to docker: %s", err)
//	}
//
//	code := m.Run()
//
//	if err := pool.Purge(resource); err != nil {
//		log.Fatalf("could not purge resource: %s", err)
//	}
//
//	os.Exit(code)
//}
//
//func TestMongoRepository_IsInParty(t *testing.T) {
//	playerId := uuid.New()
//	playerUsername := "test"
//
//	tests := []struct {
//		name string
//
//		dbModel *model.Party
//
//		player uuid.UUID
//
//		want    bool
//		wantErr error
//	}{
//		{
//			name: "in_party",
//			dbModel: &model.Party{
//				LeaderId: playerId,
//				Members:  []*model.PartyMember{{PlayerID: playerId, Username: playerUsername}},
//			},
//
//			player: playerId,
//
//			want:    true,
//			wantErr: nil,
//		},
//		{
//			name: "not_in_party",
//			dbModel: &model.Party{
//				LeaderId: playerId,
//				Members:  []*model.PartyMember{{PlayerID: playerId, Username: playerUsername}},
//			},
//
//			player: uuid.New(),
//
//			want:    false,
//			wantErr: nil,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbModel)
//			assert.NoError(t, err)
//
//			got, err := repo.IsInParty(ctx, tt.player)
//			assert.Equal(t, tt.wantErr, err)
//			assert.Equal(t, tt.want, got)
//		})
//	}
//}
//
//func TestMongoRepository_CreateParty(t *testing.T) {
//	playerId := uuid.New()
//	playerUsername := "test"
//
//	tests := []struct {
//		name string
//
//		party *model.Party
//
//		wantErr error
//	}{
//		{
//			name: "success",
//			party: &model.Party{
//				LeaderId: playerId,
//				Members:  []*model.PartyMember{{PlayerID: playerId, Username: playerUsername}},
//			},
//
//			wantErr: nil,
//		},
//		{
//			name: "failure_id_set",
//			party: &model.Party{
//				ID:       primitive.NewObjectID(),
//				LeaderId: playerId,
//				Members:  []*model.PartyMember{{PlayerID: playerId, Username: playerUsername}},
//			},
//			wantErr: ErrIdMustBeNil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			originalId := tt.party.ID
//
//			err := repo.CreateParty(ctx, tt.party)
//			assert.Equal(t, tt.wantErr, err)
//
//			// Check the ID has been set
//			if originalId == primitive.NilObjectID {
//				assert.NotEmpty(t, tt.party.ID)
//			}
//
//			if tt.wantErr != nil {
//				return
//			}
//			// Check if the party was created and is valid
//			collection := database.Collection(partyCollectionName)
//			var party model.Party
//
//			err = collection.FindOne(ctx, bson.M{"_id": tt.party.ID}).Decode(&party)
//			assert.NoError(t, err)
//			assert.Equal(t, tt.party, &party)
//		})
//	}
//}
//
//// todo
////func TestMongoRepository_SetPartyMembers(t *testing.T) {
////	party := &model.Party{
////		ID:       primitive.NewObjectID(),
////		LeaderId: uuid.New(),
////		Members:  []*model.PartyMember{{PlayerID: uuid.New(), Username: "test"}},
////	}
////}
//
//func TestMongoRepository_DeleteParty(t *testing.T) {
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: uuid.New(),
//		Members:  []*model.PartyMember{{PlayerID: uuid.New(), Username: "test"}},
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		partyId primitive.ObjectID
//
//		wantErr error
//	}{
//		{
//			name:    "success",
//			dbParty: party,
//			partyId: party.ID,
//			wantErr: nil,
//		},
//		{
//			name:    "failure_not_exists",
//			dbParty: party,
//			partyId: primitive.NewObjectID(),
//			wantErr: mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			err = repo.DeleteParty(ctx, tt.partyId)
//			assert.Equal(t, tt.wantErr, err)
//
//			// Check the party was deleted in the database
//			if tt.wantErr == nil {
//				count, err := collection.CountDocuments(ctx, bson.M{"_id": tt.partyId})
//				assert.NoError(t, err)
//				assert.Equal(t, int64(0), count)
//			}
//		})
//	}
//}
//
//func TestMongoRepository_AddPartyMember(t *testing.T) {
//	partyMembers := []model.PartyMember{
//		{PlayerID: uuid.New(), Username: "test1"},
//		{PlayerID: uuid.New(), Username: "test2"},
//	}
//
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: partyMembers[0].PlayerID,
//		Members:  []*model.PartyMember{&partyMembers[0]},
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		partyId     primitive.ObjectID
//		addedMember *model.PartyMember
//
//		wantErr error
//	}{
//		{
//			name: "success",
//
//			dbParty: party,
//
//			partyId:     party.ID,
//			addedMember: &partyMembers[1],
//
//			wantErr: nil,
//		},
//		{
//			name: "failure_party_not_exists",
//
//			dbParty: party,
//
//			partyId:     primitive.NewObjectID(),
//			addedMember: &partyMembers[1],
//
//			wantErr: mongo.ErrNoDocuments,
//		},
//		{
//			name: "failure_member_already_in_party",
//
//			dbParty: party,
//
//			partyId:     party.ID,
//			addedMember: &partyMembers[0],
//
//			wantErr: ErrAlreadyInParty,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			err = repo.AddPartyMember(ctx, tt.partyId, tt.addedMember)
//			assert.Equal(t, tt.wantErr, err)
//
//			// Check the party was updated in the database
//			if tt.wantErr == nil {
//				party := model.Party{}
//				err = collection.FindOne(ctx, bson.M{"_id": tt.partyId}).Decode(&party)
//				assert.NoError(t, err)
//				assert.Contains(t, party.Members, tt.addedMember)
//			}
//		})
//	}
//}
//
//func TestMongoRepository_SetPartyLeader(t *testing.T) {
//	partyMembers := []*model.PartyMember{
//		{PlayerID: uuid.New(), Username: "test1"},
//		{PlayerID: uuid.New(), Username: "test2"},
//	}
//
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: partyMembers[0].PlayerID,
//		Members:  partyMembers,
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		partyId     primitive.ObjectID
//		setLeaderId uuid.UUID
//
//		wantErr error
//	}{
//		{
//			name: "success",
//
//			dbParty: party,
//
//			partyId:     party.ID,
//			setLeaderId: partyMembers[1].PlayerID,
//
//			wantErr: nil,
//		},
//		{
//			name: "failure_party_not_exists",
//
//			dbParty: party,
//
//			partyId:     primitive.NewObjectID(),
//			setLeaderId: partyMembers[1].PlayerID,
//
//			wantErr: mongo.ErrNoDocuments,
//		},
//		{
//			name: "failure_already_leader",
//
//			dbParty: party,
//
//			partyId:     party.ID,
//			setLeaderId: partyMembers[0].PlayerID,
//
//			wantErr: ErrAlreadyLeader,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			err = repo.SetPartyLeader(ctx, tt.partyId, tt.setLeaderId)
//			assert.Equal(t, tt.wantErr, err)
//
//			// Check the party was updated in the database
//			if tt.wantErr == nil {
//				party := model.Party{}
//				err = collection.FindOne(ctx, bson.M{"_id": tt.partyId}).Decode(&party)
//				assert.NoError(t, err)
//				assert.Equal(t, tt.setLeaderId, party.LeaderId)
//			}
//		})
//	}
//}
//
//func TestMongoRepository_GetPartyById(t *testing.T) {
//	member := &model.PartyMember{PlayerID: uuid.New(), Username: "test"}
//
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: member.PlayerID,
//		Members:  []*model.PartyMember{member},
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		partyId primitive.ObjectID
//
//		wantParty *model.Party
//		wantErr   error
//	}{
//		{
//			name:    "success",
//			dbParty: party,
//			partyId: party.ID,
//
//			wantParty: party,
//		},
//		{
//			name:    "failure_party_not_exists",
//			dbParty: party,
//			partyId: primitive.NewObjectID(),
//
//			wantErr: mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			party, err := repo.GetPartyByID(ctx, tt.partyId)
//			assert.Equal(t, tt.wantErr, err)
//			assert.Equal(t, tt.wantParty, party)
//		})
//	}
//}
//
//func TestMongoRepository_GetPartyByMemberId(t *testing.T) {
//	member := &model.PartyMember{PlayerID: uuid.New(), Username: "test"}
//
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: member.PlayerID,
//		Members:  []*model.PartyMember{member},
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		memberId uuid.UUID
//
//		wantParty *model.Party
//		wantErr   error
//	}{
//		{
//			name:     "success",
//			dbParty:  party,
//			memberId: member.PlayerID,
//
//			wantParty: party,
//		},
//		{
//			name:     "failure_party_not_exists",
//			dbParty:  party,
//			memberId: uuid.New(),
//
//			wantErr: mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			party, err := repo.GetPartyByMemberID(ctx, tt.memberId)
//			assert.Equal(t, tt.wantErr, err)
//			assert.Equal(t, tt.wantParty, party)
//		})
//	}
//}
//
//func TestMongoRepository_GetPartyIdByMemberId(t *testing.T) {
//	member := &model.PartyMember{PlayerID: uuid.New(), Username: "test"}
//
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: member.PlayerID,
//		Members:  []*model.PartyMember{member},
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		memberId uuid.UUID
//
//		wantPartyId primitive.ObjectID
//		wantErr     error
//	}{
//		{
//			name:     "success",
//			dbParty:  party,
//			memberId: member.PlayerID,
//
//			wantPartyId: party.ID,
//		},
//		{
//			name:     "failure_party_not_exists",
//			dbParty:  party,
//			memberId: uuid.New(),
//
//			wantErr: mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			partyId, err := repo.GetPartyIdByMemberId(ctx, tt.memberId)
//			assert.Equal(t, tt.wantErr, err)
//			assert.Equal(t, tt.wantPartyId, partyId)
//		})
//	}
//}
//
//func TestMongoRepository_GetPartyLeaderIdByMemberId(t *testing.T) {
//	members := []*model.PartyMember{{PlayerID: uuid.New(), Username: "test1"}, {PlayerID: uuid.New(), Username: "test2"}}
//
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: members[0].PlayerID,
//		Members:  members,
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		targetId uuid.UUID
//
//		wantLeaderId uuid.UUID
//		wantErr      error
//	}{
//		{
//			name:     "success",
//			dbParty:  party,
//			targetId: members[0].PlayerID,
//
//			wantLeaderId: party.LeaderId,
//		},
//		{
//			name:     "success_not_leader",
//			dbParty:  party,
//			targetId: members[1].PlayerID,
//
//			wantLeaderId: party.LeaderId,
//		},
//		{
//			name:     "failure_party_not_exists",
//			dbParty:  party,
//			targetId: uuid.New(),
//
//			wantErr: mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			leaderId, err := repo.GetPartyLeaderIdByMemberId(ctx, tt.targetId)
//			assert.Equal(t, tt.wantErr, err)
//			assert.Equal(t, tt.wantLeaderId, leaderId)
//		})
//	}
//}
//
//func TestMongoRepository_RemoveMemberFromParty(t *testing.T) {
//	partyMembers := []*model.PartyMember{{PlayerID: uuid.New(), Username: "test1"}, {PlayerID: uuid.New(), Username: "test2"}}
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: partyMembers[0].PlayerID,
//		Members:  partyMembers,
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		partyId  primitive.ObjectID
//		memberId uuid.UUID
//
//		wantParty *model.Party
//		wantErr   error
//	}{
//		{
//			name:     "success",
//			dbParty:  party,
//			partyId:  party.ID,
//			memberId: partyMembers[1].PlayerID,
//
//			wantParty: &model.Party{
//				ID:       party.ID,
//				LeaderId: party.LeaderId,
//				Members:  []*model.PartyMember{partyMembers[0]},
//			},
//		},
//		{
//			name:     "failure_party_not_exists",
//			dbParty:  party,
//			partyId:  primitive.NewObjectID(),
//			memberId: partyMembers[1].PlayerID,
//
//			wantParty: party,
//			wantErr:   mongo.ErrNoDocuments,
//		},
//		{
//			name:     "failure_member_not_in_party",
//			dbParty:  party,
//			partyId:  party.ID,
//			memberId: uuid.New(),
//
//			wantParty: party,
//			wantErr:   ErrNotInParty,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			err = repo.RemoveMemberFromParty(ctx, tt.partyId, tt.memberId)
//			assert.Equal(t, tt.wantErr, err)
//
//			var party *model.Party
//			err = collection.FindOne(ctx, bson.M{"_id": tt.dbParty.ID}).Decode(&party)
//			assert.NoError(t, err)
//			assert.Equal(t, tt.wantParty, party)
//		})
//	}
//}
//
//func TestMongoRepository_RemoveMemberFromSelfParty(t *testing.T) {
//	members := []*model.PartyMember{{PlayerID: uuid.New(), Username: "test1"}, {PlayerID: uuid.New(), Username: "test2"}}
//	party := &model.Party{
//		ID:       primitive.NewObjectID(),
//		LeaderId: members[0].PlayerID,
//		Members:  members,
//	}
//
//	tests := []struct {
//		name string
//
//		dbParty *model.Party
//
//		memberId uuid.UUID
//
//		wantParty *model.Party
//		wantErr   error
//	}{
//		{
//			name:     "success",
//			dbParty:  party,
//			memberId: members[1].PlayerID,
//
//			wantParty: &model.Party{
//				ID:       party.ID,
//				LeaderId: party.LeaderId,
//				Members:  []*model.PartyMember{members[0]},
//			},
//		},
//		{
//			name:     "failure_member_not_in_party",
//			dbParty:  party,
//			memberId: uuid.New(),
//
//			wantParty: party,
//			wantErr:   mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbParty)
//			assert.NoError(t, err)
//
//			err = repo.RemoveMemberFromSelfParty(ctx, tt.memberId)
//			assert.Equal(t, tt.wantErr, err)
//
//			var party *model.Party
//			err = collection.FindOne(ctx, bson.M{"_id": tt.dbParty.ID}).Decode(&party)
//			assert.NoError(t, err)
//			assert.Equal(t, tt.wantParty, party)
//		})
//	}
//}
//
//func TestMongoRepository_CreatePartyInvite(t *testing.T) {
//	invite := &model.PartyInvite{
//		InviterID:       uuid.New(),
//		InviterUsername: "test",
//
//		TargetID:       uuid.New(),
//		TargetUsername: "test2",
//
//		ExpiresAt: time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//	}
//
//	tests := []struct {
//		name    string
//		invite  *model.PartyInvite
//		wantErr error
//	}{
//		{
//			name:   "success",
//			invite: invite,
//		},
//		{
//			name: "failure_id_specified",
//			invite: &model.PartyInvite{
//				ID: primitive.NewObjectID(),
//			},
//			wantErr: ErrIdMustBeNil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			err := repo.CreatePartyInvite(ctx, tt.invite)
//			assert.Equal(t, tt.wantErr, err)
//			assert.NotEqual(t, primitive.NilObjectID, tt.invite.ID)
//
//			if tt.wantErr == nil {
//				// Retrieve the invite from the database and compare it to the one we created
//				collection := database.Collection(partyInviteCollectionName)
//				var dbInvite *model.PartyInvite
//				err = collection.FindOne(ctx, bson.M{"_id": tt.invite.ID}).Decode(&dbInvite)
//				assert.NoError(t, err)
//				assert.Equal(t, tt.invite, dbInvite)
//			}
//		})
//	}
//}
//
//func TestMongoRepository_DeletePartyInvite(t *testing.T) {
//	invite := &model.PartyInvite{
//		InviterID:       uuid.New(),
//		InviterUsername: "test",
//
//		TargetID:       uuid.New(),
//		TargetUsername: "test2",
//
//		ExpiresAt: time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//	}
//
//	tests := []struct {
//		name string
//
//		dbInvite *model.PartyInvite
//
//		partyId  primitive.ObjectID
//		targetId uuid.UUID
//
//		wantErr     error
//		wantPresent bool
//	}{
//		{
//			name: "success",
//
//			dbInvite: invite,
//
//			partyId:  invite.PartyID,
//			targetId: invite.TargetID,
//
//			wantPresent: false,
//		},
//		{
//			name: "failure_wrong_party_id",
//
//			dbInvite: invite,
//
//			partyId:  primitive.NewObjectID(),
//			targetId: invite.TargetID,
//
//			wantErr:     mongo.ErrNoDocuments,
//			wantPresent: true,
//		},
//		{
//			name: "failure_wrong_target_id",
//
//			dbInvite: invite,
//
//			partyId:  invite.PartyID,
//			targetId: uuid.New(),
//
//			wantErr:     mongo.ErrNoDocuments,
//			wantPresent: true,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyInviteCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbInvite)
//			assert.NoError(t, err)
//
//			err = repo.DeletePartyInvite(ctx, tt.partyId, tt.targetId)
//			assert.Equal(t, tt.wantErr, err)
//
//			count, err := collection.CountDocuments(ctx, bson.M{"_id": tt.dbInvite.ID})
//			assert.NoError(t, err)
//			assert.Equal(t, tt.wantPresent, count > 0)
//		})
//	}
//}
//
//func TestMongoRepository_GetPartyInvitesByPartyId(t *testing.T) {
//	partyId := primitive.NewObjectID()
//
//	invites := []*model.PartyInvite{
//		{
//			ID:              primitive.NewObjectID(),
//			PartyID:         partyId,
//			InviterID:       uuid.New(),
//			InviterUsername: "test",
//			TargetID:        uuid.New(),
//			TargetUsername:  "test2",
//			ExpiresAt:       time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//		},
//		{
//			ID:              primitive.NewObjectID(),
//			PartyID:         partyId,
//			InviterID:       uuid.New(),
//			InviterUsername: "test",
//			TargetID:        uuid.New(),
//			TargetUsername:  "test3",
//			ExpiresAt:       time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//		},
//		{
//			ID:              primitive.NewObjectID(),
//			PartyID:         primitive.NewObjectID(),
//			InviterID:       uuid.New(),
//			InviterUsername: "otherPartyTest",
//			TargetID:        uuid.New(),
//			TargetUsername:  "otherPartyTest2",
//			ExpiresAt:       time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//		},
//	}
//
//	tests := []struct {
//		name string
//
//		dbInvites []*model.PartyInvite
//
//		partyId primitive.ObjectID
//
//		wantInvites []*model.PartyInvite
//	}{
//		{
//			name:      "success",
//			dbInvites: invites,
//			partyId:   partyId,
//
//			wantInvites: invites[:2],
//		},
//		{
//			name:      "failure_party_not_found",
//			dbInvites: invites,
//			partyId:   primitive.NewObjectID(),
//
//			wantInvites: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyInviteCollectionName)
//			for _, invite := range tt.dbInvites {
//				_, err := collection.InsertOne(ctx, invite)
//				assert.NoError(t, err)
//			}
//
//			invites, err := repo.GetPartyInvitesByPartyId(ctx, tt.partyId)
//			assert.Nil(t, err)
//			assert.ElementsMatch(t, tt.wantInvites, invites)
//		})
//	}
//}
//
//func TestMongoRepository_DoesPartyInviteExist(t *testing.T) {
//	invites := []*model.PartyInvite{
//		{
//			ID:              primitive.NewObjectID(),
//			PartyID:         primitive.NewObjectID(),
//			InviterID:       uuid.New(),
//			InviterUsername: "test",
//			TargetID:        uuid.New(),
//			TargetUsername:  "test2",
//			ExpiresAt:       time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//		},
//		{
//			ID:              primitive.NewObjectID(),
//			PartyID:         primitive.NewObjectID(),
//			InviterID:       uuid.New(),
//			InviterUsername: "test3",
//			TargetID:        uuid.New(),
//			TargetUsername:  "test4",
//			ExpiresAt:       time.UnixMilli(time.Now().UnixMilli()).In(time.UTC),
//		},
//	}
//
//	tests := []struct {
//		name string
//
//		dbInvites []*model.PartyInvite
//
//		partyId  primitive.ObjectID
//		targetId uuid.UUID
//
//		wantExists bool
//	}{
//		{
//			name: "success",
//
//			dbInvites: invites,
//
//			partyId:  invites[0].PartyID,
//			targetId: invites[0].TargetID,
//
//			wantExists: true,
//		},
//		{
//			name: "failure_party_not_found",
//
//			dbInvites: invites,
//
//			partyId:  primitive.NewObjectID(),
//			targetId: invites[0].TargetID,
//
//			wantExists: false,
//		},
//		{
//			name: "failure_target_not_invited",
//
//			dbInvites: invites,
//
//			partyId:  invites[0].PartyID,
//			targetId: uuid.New(),
//
//			wantExists: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partyInviteCollectionName)
//			for _, invite := range tt.dbInvites {
//				_, err := collection.InsertOne(ctx, invite)
//				assert.NoError(t, err)
//			}
//
//			exists, err := repo.DoesPartyInviteExist(ctx, tt.partyId, tt.targetId)
//			assert.NoError(t, err)
//			assert.Equal(t, tt.wantExists, exists)
//		})
//	}
//}
//
//func TestMongoRepository_GetPartySettings(t *testing.T) {
//	settings := model.NewPartySettings(uuid.New())
//
//	tests := []struct {
//		name string
//
//		dbSettings *model.PartySettings
//
//		playerId uuid.UUID
//
//		wantSettings *model.PartySettings
//		wantErr      error
//	}{
//		{
//			name:       "success",
//			dbSettings: settings,
//
//			playerId: settings.PlayerID,
//
//			wantSettings: settings,
//		},
//		{
//			name:       "failure_not_found",
//			dbSettings: settings,
//
//			playerId: uuid.New(),
//
//			wantSettings: nil,
//			wantErr:      mongo.ErrNoDocuments,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partySettingsCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbSettings)
//			assert.NoError(t, err)
//
//			settings, err := repo.GetPartySettings(ctx, tt.playerId)
//			assert.Equal(t, tt.wantErr, err)
//			assert.Equal(t, tt.wantSettings, settings)
//		})
//	}
//}
//
//func TestMongoRepository_UpdatePartySettings(t *testing.T) {
//	settings := model.NewPartySettings(uuid.New())
//
//	settings.AllowMemberDequeue = !settings.AllowMemberDequeue
//	settings.AllowMemberInvite = !settings.AllowMemberInvite
//
//	tests := []struct {
//		name string
//
//		dbSettings *model.PartySettings
//
//		updateSettings *model.PartySettings
//
//		wantErr bool
//	}{
//		{
//			name:       "success",
//			dbSettings: settings,
//
//			updateSettings: settings,
//			wantErr:        false,
//		},
//		{
//			name:       "success_no_changes",
//			dbSettings: settings,
//
//			updateSettings: settings,
//			wantErr:        false,
//		},
//		{
//			name:       "success_no_found",
//			dbSettings: settings,
//
//			updateSettings: model.NewPartySettings(uuid.New()),
//			wantErr:        false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Cleanup(cleanup)
//			ctx := context.Background()
//
//			collection := database.Collection(partySettingsCollectionName)
//			_, err := collection.InsertOne(ctx, tt.dbSettings)
//			assert.NoError(t, err)
//
//			err = repo.UpdatePartySettings(ctx, tt.updateSettings)
//			if tt.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//			}
//
//			if !tt.wantErr {
//				settings, err := repo.GetPartySettings(ctx, tt.updateSettings.PlayerID)
//				assert.NoError(t, err)
//				assert.Equal(t, tt.updateSettings, settings)
//
//				// If the player IDs are different, ensure both settings are in the database
//				if tt.updateSettings.PlayerID != tt.dbSettings.PlayerID {
//					settings, err := repo.GetPartySettings(ctx, tt.dbSettings.PlayerID)
//					assert.NoError(t, err)
//					assert.Equal(t, tt.dbSettings, settings)
//				}
//			}
//		})
//	}
//}
//
//func cleanup() {
//	ctx := context.Background()
//	if err := database.Drop(ctx); err != nil {
//		log.Panicf("could not drop database: %s", err)
//	}
//}
