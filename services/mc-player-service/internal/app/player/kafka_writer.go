package player

import (
	"context"
	kafkaWriter "github.com/emortalmc/mono-services/services/mc-player-service/internal/kafka/writer"
	"github.com/google/uuid"
)

var (
	_ KafkaWriter = &kafkaWriter.Notifier{}
)

type KafkaWriter interface {
	PlayerExperienceChange(ctx context.Context, playerID uuid.UUID, reason string, oldXP int, newXP int, oldLevel int, newLevel int)
}
