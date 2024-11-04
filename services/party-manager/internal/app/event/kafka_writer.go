package event

import (
	"context"
	"github.com/emortalmc/mono-services/services/party-manager/internal/kafka/writer"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
)

var (
	_ KafkaWriter = &writer.Notifier{}
)

type KafkaWriter interface {
	DisplayEvent(ctx context.Context, event *model.Event)
	StartEvent(ctx context.Context, event *model.Event)
	DeleteEvent(ctx context.Context, event *model.Event)
}
