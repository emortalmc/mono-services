package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

const (
	databaseName = "matchmaker"

	queuedPlayerCollectionName = "queuedPlayer"
	ticketCollectionName       = "ticket"
	pendingMatchCollectionName = "pendingMatch"
	backfillCollectionName     = "backfill"
)

type Repository interface {
	HealthCheck(ctx context.Context, timeout time.Duration) error

	ExecuteTransaction(ctx context.Context, fn func(ctx mongo.SessionContext) error) error

	// QueuedPlayer

	CreateQueuedPlayers(ctx context.Context, players []*model.QueuedPlayer) error
	DeleteQueuedPlayer(ctx context.Context, playerId uuid.UUID) error
	GetQueuedPlayerById(ctx context.Context, playerId uuid.UUID) (*model.QueuedPlayer, error)

	GetAllQueuedPlayersByIds(ctx context.Context, playerIds []uuid.UUID) ([]*model.QueuedPlayer, error)

	// DeleteAllQueuedPlayersById deletes all queued players with the given ids.
	// returns: int64, the modified count.
	DeleteAllQueuedPlayersById(ctx context.Context, playerIds []uuid.UUID) (int64, error)

	SetMapIdOfQueuedPlayer(ctx context.Context, playerId uuid.UUID, mapId string) error

	// Ticket

	CreateTicket(ctx context.Context, ticket *model.Ticket) error
	DeleteTicket(ctx context.Context, ticketId primitive.ObjectID) error

	MassUpdateTicketInPendingMatch(ctx context.Context, values map[primitive.ObjectID]bool) (int64, error)

	// DeleteAllTicketsById deletes all tickets with the given ids.
	// returns: int64, the modified count.
	DeleteAllTicketsById(ctx context.Context, ticketIds []primitive.ObjectID) (int64, error)

	GetTicketByPlayerId(ctx context.Context, playerId uuid.UUID) (*model.Ticket, error)
	GetTicketsByGameMode(ctx context.Context, gameModeId string) ([]*model.Ticket, error)

	// GetUnmatchedTicketsByGameMode TODO Might be unused
	GetUnmatchedTicketsByGameMode(ctx context.Context, gameModeId string) ([]*model.Ticket, error)

	// AddTicketDequeueRequest requests that a ticket is void and removed from any PendingMatch.
	// returns: int64, the modified count. Throws mongo.ErrNoDocuments if no matches are found.
	AddTicketDequeueRequest(ctx context.Context, ticketId primitive.ObjectID) (int64, error)

	AddTicketDequeueRequestByPartyId(ctx context.Context, partyId primitive.ObjectID) (int64, error)

	// AddPlayerDequeueRequest requests that a player is removed from a ticket.
	AddPlayerDequeueRequest(ctx context.Context, ticketId primitive.ObjectID, playerId uuid.UUID) error
	AddPlayerDequeueRequestByPartyId(ctx context.Context, partyId primitive.ObjectID, playerId uuid.UUID) error

	ResetAllDequeueRequestsById(ctx context.Context, ticketIds []primitive.ObjectID) (int64, error)

	GetTicketsWithDequeueRequest(ctx context.Context, gameModeId string) ([]*model.Ticket, error)

	IsPartyQueued(ctx context.Context, partyId primitive.ObjectID) (bool, error)

	// PendingMatch

	CreatePendingMatch(ctx context.Context, match *model.PendingMatch) error
	CreatePendingMatches(ctx context.Context, matches []*model.PendingMatch) error
	UpdatePendingMatches(ctx context.Context, matches []*model.PendingMatch) error
	DeletePendingMatches(ctx context.Context, matchIds []primitive.ObjectID) error
	GetPendingMatchesByGameMode(ctx context.Context, gameModeId string) ([]*model.PendingMatch, error)
	GetPendingMatchByTicketId(ctx context.Context, ticketId primitive.ObjectID) (*model.PendingMatch, error)

	// RemoveTicketsFromPendingMatchesById removes ticket IDs from PendingMatches they are present in.
	// returns: int64, the modified count.
	RemoveTicketsFromPendingMatchesById(ctx context.Context, ticketIds []primitive.ObjectID) (int64, error)
}
