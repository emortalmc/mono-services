package model

import (
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/relationship"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FriendConnection struct {
	Id          primitive.ObjectID `bson:"_id"`
	PlayerOneId uuid.UUID          `bson:"playerOneId"`
	PlayerTwoId uuid.UUID          `bson:"playerTwoId"`
}

// PendingFriendConnection equivalent to a friend request
type PendingFriendConnection struct {
	Id          primitive.ObjectID `bson:"_id"`
	RequesterId uuid.UUID          `bson:"requesterId"`
	TargetId    uuid.UUID          `bson:"targetId"`
}

type PlayerBlock struct {
	Id        primitive.ObjectID `bson:"_id"`
	BlockerId uuid.UUID          `bson:"blockerId"`
	BlockedId uuid.UUID          `bson:"blockedId"`
}

func (b *PlayerBlock) ContainsPlayer(playerId uuid.UUID) bool {
	return b.BlockerId == playerId || b.BlockedId == playerId
}

func (b *PlayerBlock) ToProto() *pbmodel.PlayerBlock {
	return &pbmodel.PlayerBlock{
		BlockerId: b.BlockerId.String(),
		BlockedId: b.BlockedId.String(),
	}
}
