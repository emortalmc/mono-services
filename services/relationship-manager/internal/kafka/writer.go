package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/config"
	pbmsg "github.com/emortalmc/proto-specs/gen/go/message/relationship"
	"github.com/emortalmc/proto-specs/gen/go/model/relationship"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

const writeTopic = "relationship-manager"

// Notifier is an interface for sending Kafka messages into the relationship-manager topic.
//   - The context used should be independent of a request as kafka messages are sent lazily. Using a request context
//     will result in the context being cancelled before the message is sent.
type Notifier interface {
	FriendRequest(ctx context.Context, request *relationship.FriendRequest) error
	FriendAdded(ctx context.Context, senderId uuid.UUID, targetId uuid.UUID, senderUsername string) error
	FriendRemoved(ctx context.Context, senderId uuid.UUID, targetId uuid.UUID) error

	FriendConnectStatus(ctx context.Context, playerId uuid.UUID, playerUsername string, joined bool, notifyPlayers []string) error
}

type kafkaNotifier struct {
	w *kafka.Writer
}

func NewKafkaNotifier(ctx context.Context, wg *sync.WaitGroup, cfg *config.KafkaConfig, logger *zap.SugaredLogger) Notifier {
	w := &kafka.Writer{
		Addr:         kafka.TCP(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		Topic:        writeTopic,
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
			return
		}
	}()

	return &kafkaNotifier{w: w}
}

func (k *kafkaNotifier) FriendRequest(ctx context.Context, fReq *relationship.FriendRequest) error {
	msg := &pbmsg.FriendRequestReceivedMessage{
		Request: fReq,
	}

	if err := k.writeMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}

	return nil
}

func (k *kafkaNotifier) FriendAdded(ctx context.Context, senderId uuid.UUID, targetId uuid.UUID, senderUsername string) error {
	msg := &pbmsg.FriendAddedMessage{
		SenderId:       senderId.String(),
		SenderUsername: senderUsername,
		RecipientId:    targetId.String(),
	}

	if err := k.writeMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}

	return nil
}

func (k *kafkaNotifier) FriendRemoved(ctx context.Context, senderId uuid.UUID, targetId uuid.UUID) error {
	msg := &pbmsg.FriendRemovedMessage{
		SenderId:    senderId.String(),
		RecipientId: targetId.String(),
	}

	if err := k.writeMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}

	return nil
}

func (k *kafkaNotifier) FriendConnectStatus(ctx context.Context, playerId uuid.UUID, playerUsername string, joined bool,
	notifyPlayers []string) error {

	msg := &pbmsg.FriendConnectionMessage{
		MessageTargetIds: notifyPlayers,
		PlayerId:         playerId.String(),
		Username:         playerUsername,
		Joined:           joined,
	}

	if err := k.writeMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}

	return nil
}

func (k *kafkaNotifier) writeMessage(ctx context.Context, msg proto.Message) error {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal proto to bytes: %s", err)
	}

	return k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(msg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})
}
