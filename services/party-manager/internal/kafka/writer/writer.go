package writer

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/party-manager/internal/config"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	partymsg "github.com/emortalmc/proto-specs/gen/go/message/party"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

const partyWriteTopic = "party-manager"
const eventWriteTopic = "event-manager"

type Notifier struct {
	logger *zap.SugaredLogger
	w      *kafka.Writer
}

func NewKafkaNotifier(ctx context.Context, wg *sync.WaitGroup, cfg config.KafkaConfig, logger *zap.SugaredLogger) *Notifier {
	w := &kafka.Writer{
		Addr:         kafka.TCP(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		Balancer:     &kafka.LeastBytes{},
		Async:        true,
		BatchTimeout: 100 * time.Millisecond,
		ErrorLogger:  kafka.LoggerFunc(logger.Errorw),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err := w.Close(); err != nil {
			logger.Errorw("failed to close kafka writer", "err", err)
		}
	}()

	return &Notifier{
		logger: logger,
		w:      w,
	}
}

func (k *Notifier) PartyCreated(ctx context.Context, party *model.Party) {
	msg := &partymsg.PartyCreatedMessage{Party: party.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyDeleted(ctx context.Context, party *model.Party) {
	msg := &partymsg.PartyDeletedMessage{Party: party.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyEmptied(ctx context.Context, party *model.Party) {
	msg := &partymsg.PartyEmptiedMessage{Party: party.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyOpenChanged(ctx context.Context, partyId primitive.ObjectID, open bool) {
	msg := &partymsg.PartyOpenChangedMessage{PartyId: partyId.Hex(), Open: open}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyInviteCreated(ctx context.Context, invite *model.PartyInvite) {
	msg := &partymsg.PartyInviteCreatedMessage{Invite: invite.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyPlayerJoined(ctx context.Context, partyId primitive.ObjectID, player *model.PartyMember) {
	msg := &partymsg.PartyPlayerJoinedMessage{PartyId: partyId.Hex(), Member: player.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyPlayerLeft(ctx context.Context, partyId primitive.ObjectID, player *model.PartyMember) {
	msg := &partymsg.PartyPlayerLeftMessage{PartyId: partyId.Hex(), Member: player.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyPlayerKicked(ctx context.Context, partyId primitive.ObjectID, kicked *model.PartyMember, kicker *model.PartyMember) {
	msg := &partymsg.PartyPlayerLeftMessage{PartyId: partyId.Hex(), Member: kicked.ToProto(), KickedBy: kicker.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartyLeaderChanged(ctx context.Context, partyId primitive.ObjectID, newLeader *model.PartyMember) {
	msg := &partymsg.PartyLeaderChangedMessage{PartyId: partyId.Hex(), NewLeader: newLeader.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) PartySettingsChanged(ctx context.Context, playerId uuid.UUID, settings *model.PartySettings) {
	msg := &partymsg.PartySettingsChangedMessage{PlayerId: playerId.String(), Settings: settings.ToProto()}

	if err := k.writeMessage(ctx, msg, true); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) DisplayEvent(ctx context.Context, event *model.Event) {
	msg := &partymsg.EventDisplayMessage{Event: event.ToProto()}

	if err := k.writeMessage(ctx, msg, false); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) StartEvent(ctx context.Context, event *model.Event) {
	msg := &partymsg.EventStartMessage{Event: event.ToProto()}

	if err := k.writeMessage(ctx, msg, false); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) DeleteEvent(ctx context.Context, event *model.Event) {
	msg := &partymsg.EventDeleteMessage{Event: event.ToProto()}

	if err := k.writeMessage(ctx, msg, false); err != nil {
		k.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (k *Notifier) writeMessage(ctx context.Context, msg proto.Message, partyMsg bool) error {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal proto to bytes: %s", err)
	}

	var topic string
	if partyMsg {
		topic = partyWriteTopic
	} else {
		topic = eventWriteTopic
	}

	return k.w.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(msg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})
}
