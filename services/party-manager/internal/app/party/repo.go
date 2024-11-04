package party

import (
	"context"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	_ ReadWriter = &repository.MongoRepository{}
)

type Reader interface {
	IsInParty(ctx context.Context, playerId uuid.UUID) (bool, error)

	GetPartyByID(ctx context.Context, partyId primitive.ObjectID) (*model.Party, error)
	GetPartyByMemberID(ctx context.Context, playerId uuid.UUID) (*model.Party, error)
	GetPartyIdByMemberId(ctx context.Context, playerId uuid.UUID) (primitive.ObjectID, error)
	GetPartyLeaderIdByMemberId(ctx context.Context, playerId uuid.UUID) (uuid.UUID, error)
	GetPartyLeaderByPartyId(ctx context.Context, partyId primitive.ObjectID) (uuid.UUID, error)
	// GetPartyInvitesByPartyId returns all party invites for a given party.
	// Note that this does not check if the party exists. If the party does not exist, an empty slice is returned.
	GetPartyInvitesByPartyId(ctx context.Context, partyId primitive.ObjectID) ([]*model.PartyInvite, error)
	DoesPartyInviteExist(ctx context.Context, partyId primitive.ObjectID, playerId uuid.UUID) (bool, error)

	GetPartySettings(ctx context.Context, playerId uuid.UUID) (*model.PartySettings, error)
}

type Writer interface {
	// CreateParty creates a new party and sets the ID of the party for the reference.
	CreateParty(ctx context.Context, party *model.Party) error
	DeleteParty(ctx context.Context, partyId primitive.ObjectID) error

	SetPartyMembers(ctx context.Context, partyId primitive.ObjectID, members []*model.PartyMember) error
	AddPartyMember(ctx context.Context, partyId primitive.ObjectID, member *model.PartyMember) error

	SetPartyLeader(ctx context.Context, partyId primitive.ObjectID, leaderId uuid.UUID) error

	// SetPartyOpen sets the party to open or closed.
	// If no party exists, mongo.ErrNoDocuments is thrown.
	SetPartyOpen(ctx context.Context, partyId primitive.ObjectID, open bool) error

	SetPartyEventID(ctx context.Context, partyId primitive.ObjectID, eventId string) error
	RemovePartyEventID(ctx context.Context, partyId primitive.ObjectID) error

	// RemoveMemberFromParty removes from party.
	// Throws mongo.ErrNoDocuments if party does not exist.
	// Throws repository.ErrNotInParty if player is not in the party.
	RemoveMemberFromParty(ctx context.Context, partyId primitive.ObjectID, playerId uuid.UUID) error
	// RemoveMemberFromSelfParty removes a player from their party.
	// Throws mongo.ErrNoDocuments if the player is not in a party.
	RemoveMemberFromSelfParty(ctx context.Context, playerId uuid.UUID) error

	// CreatePartyInvite creates a new invite and sets the ID of the invite for the reference.
	CreatePartyInvite(ctx context.Context, invite *model.PartyInvite) error
	DeletePartyInvite(ctx context.Context, partyId primitive.ObjectID, targetId uuid.UUID) error
	DeletePartyInvitesByPartyId(ctx context.Context, partyId primitive.ObjectID) error

	UpdatePartySettings(ctx context.Context, settings *model.PartySettings) error
}

type ReadWriter interface {
	Reader
	Writer
}
