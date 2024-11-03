package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/repository/model"
	"github.com/google/uuid"
)

type Repository interface {
	CreateFriendConnection(ctx context.Context, connection *model.FriendConnection) error
	GetFriendConnections(ctx context.Context, playerId uuid.UUID) ([]*model.FriendConnection, error)
	DeleteFriendConnection(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) error
	AreFriends(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) (bool, error)

	CreatePendingFriendConnection(ctx context.Context, connection *model.PendingFriendConnection) error
	DoesPendingFriendConnectionExist(ctx context.Context, requesterId uuid.UUID, targetId uuid.UUID) (bool, error)
	GetPendingFriendConnections(ctx context.Context, playerId uuid.UUID, options DirectionOpts) ([]*model.PendingFriendConnection, error)

	DeletePendingFriendConnection(ctx context.Context, requesterId uuid.UUID, targetId uuid.UUID) error
	DeletePendingFriendConnections(ctx context.Context, playerId uuid.UUID, opts *DirectionOpts) (int64, error)

	CreatePlayerBlock(ctx context.Context, block *model.PlayerBlock) error
	DeletePlayerBlock(ctx context.Context, blockerId uuid.UUID, blockedId uuid.UUID) error
	// IsPlayerBlocked returns if either player has blocked the other
	IsPlayerBlocked(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) (bool, error)
	GetMutualBlocks(ctx context.Context, playerOneId uuid.UUID, playerTwoId uuid.UUID) ([]*model.PlayerBlock, error)
	GetPlayerBlocks(ctx context.Context, blockerId uuid.UUID) ([]*model.PlayerBlock, error)
}

type DirectionOpts struct {
	Incoming bool
	Outgoing bool
}
