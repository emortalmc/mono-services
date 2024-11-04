package model

import (
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/party"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"
)

func TestParty_ToProto(t *testing.T) {
	partyId := primitive.NewObjectID()
	playerIds := []uuid.UUID{uuid.New(), uuid.New()}
	playerUsernames := []string{"test", "test2"}

	tests := []struct {
		name  string
		party *Party
		want  *pbmodel.Party
	}{
		{
			name: "party to proto",
			party: &Party{
				ID:       partyId,
				LeaderId: playerIds[0],
				Members:  []*PartyMember{{PlayerID: playerIds[0], Username: playerUsernames[0]}, {PlayerID: playerIds[1], Username: playerUsernames[1]}},
			},
			want: &pbmodel.Party{
				Id:       partyId.Hex(),
				LeaderId: playerIds[0].String(),
				Members:  []*pbmodel.PartyMember{{Id: playerIds[0].String(), Username: playerUsernames[0]}, {Id: playerIds[1].String(), Username: playerUsernames[1]}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := tt.party.ToProto()
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestParty_ContainsMember(t *testing.T) {
	playerId := uuid.New()
	tests := []struct {
		name         string
		party        *Party
		testMemberId uuid.UUID
		want         bool
	}{
		{
			name: "party contains member",
			party: &Party{
				Members: []*PartyMember{
					{
						PlayerID: playerId,
					},
					{
						PlayerID: uuid.New(),
					},
				},
			},
			testMemberId: playerId,
			want:         true,
		},
		{
			name: "party does not contain member",
			party: &Party{
				Members: []*PartyMember{
					{
						PlayerID: uuid.New(),
					},
					{
						PlayerID: playerId,
					},
				},
			},
			testMemberId: uuid.New(),
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := tt.party.ContainsMember(tt.testMemberId)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestPartyInvite_ToProto(t *testing.T) {
	partyId := primitive.NewObjectID()
	inviteId := primitive.NewObjectID()

	inviterId := uuid.New()
	inviterUsername := "test"

	targetId := uuid.New()
	targetUsername := "test2"

	expiresAt := time.Now().Add(time.Minute * 5)

	tests := []struct {
		name   string
		invite *PartyInvite
		want   *pbmodel.PartyInvite
	}{
		{
			name: "party invite to proto",
			invite: &PartyInvite{
				ID:      inviteId,
				PartyID: partyId,

				InviterID:       inviterId,
				InviterUsername: inviterUsername,

				TargetID:       targetId,
				TargetUsername: targetUsername,
				ExpiresAt:      expiresAt,
			},
			want: &pbmodel.PartyInvite{
				PartyId:        partyId.Hex(),
				SenderId:       inviterId.String(),
				SenderUsername: inviterUsername,
				TargetId:       targetId.String(),
				TargetUsername: targetUsername,
				ExpiresAt:      timestamppb.New(expiresAt),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := tt.invite.ToProto()
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestPartySettings_ToProto(t *testing.T) {
	playerId := uuid.New()

	tests := []struct {
		name     string
		settings *PartySettings
		want     *pbmodel.PartySettings
	}{
		{
			name: "party settings to proto",
			settings: &PartySettings{
				PlayerID: playerId,

				DequeueOnDisconnect: false,
				AllowMemberDequeue:  false,
				AllowMemberInvite:   false,
			},
			want: &pbmodel.PartySettings{
				DequeueOnDisconnect: false,
				AllowMemberDequeue:  false,
				AllowMemberInvite:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := tt.settings.ToProto()
			assert.Equal(t, tt.want, val)
		})
	}
}
