package kafkaWriter

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/proto-specs/gen/go/model/mcplayer"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

const experienceWriterTopic = "player-experience"

type Notifier struct {
	logger *zap.SugaredLogger
	w      *kafka.Writer
}

func NewKafkaNotifier(ctx context.Context, wg *sync.WaitGroup, cfg config.KafkaConfig, logger *zap.SugaredLogger) *Notifier {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Host),
		Topic:        experienceWriterTopic,
		Balancer:     &kafka.LeastBytes{},
		Async:        true,
		BatchTimeout: 500 * time.Millisecond,
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

func (n *Notifier) PlayerExperienceChange(ctx context.Context, playerID uuid.UUID, reason string, oldXP int, newXP int, oldLevel int, newLevel int) {
	msg := &mcplayer.PlayerExperienceChangeMessage{
		PlayerId:           playerID.String(),
		Reason:             reason,
		PreviousExperience: int64(oldXP),
		NewExperience:      int64(newXP),
		PreviousLevel:      int32(oldLevel),
		NewLevel:           int32(newLevel),
	}

	if err := n.writeMessage(ctx, msg); err != nil {
		n.logger.Errorw("failed to write message", "err", err)
		return
	}
}

func (n *Notifier) writeMessage(ctx context.Context, msg proto.Message) error {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal proto to bytes: %s", err)
	}

	return n.w.WriteMessages(ctx, kafka.Message{
		Topic:   experienceWriterTopic,
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(msg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})
}
