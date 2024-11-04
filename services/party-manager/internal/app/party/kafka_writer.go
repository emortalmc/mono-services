package party

import (
	"context"
	"github.com/emortalmc/mono-services/services/party-manager/internal/kafka/writer"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	_ KafkaWriter = &writer.Notifier{}
)

type KafkaWriter interface {
	PartyCreated(ctx context.Context, party *model.Party)
	PartyDeleted(ctx context.Context, party *model.Party)
	PartyEmptied(ctx context.Context, party *model.Party)
	PartyOpenChanged(ctx context.Context, partyId primitive.ObjectID, open bool)
	PartyInviteCreated(ctx context.Context, invite *model.PartyInvite)
	PartyPlayerJoined(ctx context.Context, partyId primitive.ObjectID, player *model.PartyMember)
	PartyPlayerLeft(ctx context.Context, partyId primitive.ObjectID, player *model.PartyMember)
	PartyPlayerKicked(ctx context.Context, partyId primitive.ObjectID, kicked *model.PartyMember, kicker *model.PartyMember)
	PartyLeaderChanged(ctx context.Context, partyId primitive.ObjectID, newLeader *model.PartyMember)

	PartySettingsChanged(ctx context.Context, playerId uuid.UUID, settings *model.PartySettings)
}
