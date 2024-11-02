package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/message-handler/internal/config"
	pbmsg "github.com/emortalmc/proto-specs/gen/go/message/messagehandler"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/messagehandler"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"time"
)

const writeTopic = "message-handler"

type Notifier interface {
	PrivateMessageCreated(ctx context.Context, msg *pbmodel.PrivateMessage) error
	ChatMessageCreated(ctx context.Context, msg *pbmodel.ChatMessage) error
}

type kafkaNotifier struct {
	w *kafka.Writer
}

func NewKafkaNotifier(cfg *config.KafkaConfig, logger *zap.SugaredLogger) Notifier {
	w := &kafka.Writer{
		Addr:            kafka.TCP(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		Topic:           writeTopic,
		Balancer:        &kafka.LeastBytes{},
		Async:           true,
		WriteBackoffMin: 10 * time.Millisecond,
		BatchSize:       100,
		BatchTimeout:    100 * time.Millisecond,
		ErrorLogger:     kafka.LoggerFunc(logger.Errorw),
	}

	return &kafkaNotifier{w: w}
}

func (n *kafkaNotifier) PrivateMessageCreated(ctx context.Context, pm *pbmodel.PrivateMessage) error {
	message := &pbmsg.PrivateMessageCreatedMessage{PrivateMessage: pm}

	if err := n.writeMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}
	return nil
}

func (n *kafkaNotifier) ChatMessageCreated(ctx context.Context, cm *pbmodel.ChatMessage) error {
	message := &pbmsg.ChatMessageCreatedMessage{Message: cm}

	if err := n.writeMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}
	return nil
}

func (n *kafkaNotifier) writeMessage(ctx context.Context, msg proto.Message) error {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal proto to bytes: %s", err)
	}

	return n.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(msg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})
}
