package notifier

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/permission-service/internal/config"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/model"
	"github.com/emortalmc/proto-specs/gen/go/message/permission"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/permission"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

const topic = "permission-manager"

type kafkaNotifier struct {
	logger *zap.SugaredLogger
	w      *kafka.Writer
}

func NewKafkaNotifier(ctx context.Context, wg *sync.WaitGroup, logger *zap.SugaredLogger, cfg config.KafkaConfig) Notifier {
	w := &kafka.Writer{
		Addr:        kafka.TCP(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		Topic:       topic,
		Async:       true,
		Balancer:    &kafka.LeastBytes{},
		ErrorLogger: zap.NewStdLog(zap.L()),
	}

	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Info("shutting down kafka writer")
		if err := w.Close(); err != nil {
			logger.Errorw("failed to close kafka writer", "error", err)
		}
	}()

	return &kafkaNotifier{
		logger: logger,
		w:      w,
	}
}

func (k *kafkaNotifier) RoleUpdate(ctx context.Context, role *model.Role, changeType permission.RoleUpdateMessage_ChangeType) error {
	var protoRole *pbmodel.Role
	if role != nil {
		protoRole = role.ToProto()
	}

	msg := &permission.RoleUpdateMessage{Role: protoRole, ChangeType: changeType}
	if err := k.publishMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (k *kafkaNotifier) PlayerRolesUpdate(ctx context.Context, playerId string, roleId string, changeType permission.PlayerRolesUpdateMessage_ChangeType) error {
	msg := &permission.PlayerRolesUpdateMessage{PlayerId: playerId, RoleId: roleId, ChangeType: changeType}
	if err := k.publishMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (k *kafkaNotifier) publishMessage(ctx context.Context, message proto.Message) error {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := k.w.WriteMessages(ctx, kafka.Message{
		Value:   bytes,
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(message.ProtoReflect().Descriptor().FullName())}},
	}); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}
