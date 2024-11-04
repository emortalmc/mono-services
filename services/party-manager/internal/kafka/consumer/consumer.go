package consumer

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/party-manager/internal/app/party"
	"github.com/emortalmc/mono-services/services/party-manager/internal/config"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

const connectionTopic = "mc-connections"

type Consumer struct {
	logger *zap.SugaredLogger

	partySvc *party.Service

	reader *kafka.Reader
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, config config.KafkaConfig, logger *zap.SugaredLogger, partySvc *party.Service) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		GroupID:     "party-manager",
		GroupTopics: []string{connectionTopic},

		ErrorLogger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Errorw(fmt.Sprintf(format, args...))
		}),
	})

	c := &Consumer{
		logger:   logger,
		partySvc: partySvc,
		reader:   reader,
	}

	handler := kafkautils.NewConsumerHandler(logger, reader)
	handler.RegisterHandler(&common.PlayerConnectMessage{}, c.handlePlayerConnect)
	handler.RegisterHandler(&common.PlayerDisconnectMessage{}, c.handlePlayerDisconnect)

	logger.Infow("started listening for kafka messages", "topics", reader.Config().GroupTopics)

	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.Run(ctx) // run is blocking until the context is cancelled
		if err := reader.Close(); err != nil {
			logger.Errorw("failed to close kafka reader", err)
		}
	}()
}

// handlePlayerConnect creates a new party for a player
// when they join the server.
func (c *Consumer) handlePlayerConnect(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*common.PlayerConnectMessage)

	playerId, err := uuid.Parse(m.PlayerId)
	if err != nil {
		c.logger.Errorw("failed to parse player id", err)
		return
	}

	c.partySvc.HandlePlayerConnect(ctx, playerId, m.PlayerUsername)
}

func (c *Consumer) handlePlayerDisconnect(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*common.PlayerDisconnectMessage)

	playerId, err := uuid.Parse(m.PlayerId)
	if err != nil {
		c.logger.Errorw("failed to parse player id", err)
		return
	}

	c.partySvc.HandlePlayerDisconnect(ctx, playerId)
}
