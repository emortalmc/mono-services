package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/config"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/repository"
	"github.com/emortalmc/proto-specs/gen/go/message/party"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

// NOTE: We don't listen to player connections as player disconnect = party leave or disband - so it's handled by that.
const partyTopic = "party-manager"

type consumer struct {
	logger *zap.SugaredLogger

	reader *kafka.Reader

	repo repository.Repository
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, config config.KafkaConfig, logger *zap.SugaredLogger, repo repository.Repository) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		GroupID:     "matchmaker",
		GroupTopics: []string{partyTopic},

		Logger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Infow(fmt.Sprintf(format, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Errorw(fmt.Sprintf(format, args...))
		}),
	})

	c := &consumer{
		logger: logger,
		reader: reader,
		repo:   repo,
	}

	handler := kafkautils.NewConsumerHandler(logger, reader)
	handler.RegisterHandler(&party.PartyDeletedMessage{}, c.handlePartyDisband)
	handler.RegisterHandler(&party.PartyPlayerJoinedMessage{}, c.handlePartyPlayerJoined)
	handler.RegisterHandler(&party.PartyPlayerLeftMessage{}, c.handlePartyPlayerLeft)

	logger.Infow("starting listening for kafka messages", "topics", reader.Config().GroupTopics)

	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.Run(ctx) // Run is blocking until the context is cancelled
		if err := reader.Close(); err != nil {
			logger.Errorw("error closing kafka reader", "error", err)
		}
	}()
}

func (c *consumer) handlePartyDisband(ctx context.Context, _ *kafka.Message, uncast proto.Message) {
	pMsg := uncast.(*party.PartyDeletedMessage)

	// Dequeue if possible
	partyId, err := primitive.ObjectIDFromHex(pMsg.Party.Id)
	if err != nil {
		c.logger.Errorw("failed to parse party id", err)
		return
	}

	_, err = c.repo.AddTicketDequeueRequestByPartyId(ctx, partyId)
	if err != nil {
		c.logger.Errorw("failed to add ticket dequeue request", err)
		return
	}
}

// TODO we don't account for if a ticket size increases and there isn't enough space in their existing PendingMatch
func (c *consumer) handlePartyPlayerJoined(ctx context.Context, _ *kafka.Message, uncast proto.Message) {
	//pMsg := uncast.(*party.PartyPlayerJoinedMessage)
	// TODO
}

func (c *consumer) handlePartyPlayerLeft(ctx context.Context, _ *kafka.Message, uncast proto.Message) {
	pMsg := uncast.(*party.PartyPlayerLeftMessage)

	partyId, err := primitive.ObjectIDFromHex(pMsg.PartyId)
	if err != nil {
		c.logger.Errorw("failed to parse party id", err)
		return
	}

	playerId, err := uuid.Parse(pMsg.Member.Id)
	if err != nil {
		c.logger.Errorw("failed to parse player id", err)
		return
	}

	err = c.repo.AddPlayerDequeueRequestByPartyId(ctx, partyId, playerId)
	if err != nil {
		c.logger.Errorw("failed to add player dequeue request", err)
		return
	}
}
