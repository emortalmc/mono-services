package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/relationship-manager-service/internal/config"
	"github.com/emortalmc/mono-services/services/relationship-manager-service/internal/repository"
	"github.com/emortalmc/proto-specs/gen/go/grpc/mcplayer"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

const connectionsTopic = "mc-connections"

type consumer struct {
	logger *zap.SugaredLogger

	repo          repository.Repository
	notif         Notifier
	playerTracker mcplayer.PlayerTrackerClient

	reader *kafka.Reader
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, cfg *config.KafkaConfig, logger *zap.SugaredLogger, repo repository.Repository,
	notif Notifier, playerTracker mcplayer.PlayerTrackerClient) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		GroupID:     "relationship-manager",
		GroupTopics: []string{connectionsTopic},

		Logger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Infow(fmt.Sprintf(format, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Errorw(fmt.Sprintf(format, args...))
		}),

		MaxWait: 5 * time.Second,
	})

	c := &consumer{
		logger: logger,

		repo:          repo,
		notif:         notif,
		playerTracker: playerTracker,

		reader: reader,
	}

	handler := kafkautils.NewConsumerHandler(logger, reader)
	handler.RegisterHandler(&common.PlayerConnectMessage{}, c.handlePlayerConnectMessage)
	handler.RegisterHandler(&common.PlayerDisconnectMessage{}, c.handlePlayerDisconnectMessage)

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

func (c *consumer) handlePlayerConnectMessage(ctx context.Context, _ *kafka.Message, uncast proto.Message) {
	msg := uncast.(*common.PlayerConnectMessage)

	pId, err := uuid.Parse(msg.PlayerId)
	if err != nil {
		c.logger.Errorw("failed to parse player id", "error", err)
		return
	}

	c.handleConnectionMessage(ctx, pId, msg.PlayerUsername, true)
}

func (c *consumer) handlePlayerDisconnectMessage(ctx context.Context, _ *kafka.Message, uncast proto.Message) {
	msg := uncast.(*common.PlayerDisconnectMessage)

	pId, err := uuid.Parse(msg.PlayerId)
	if err != nil {
		c.logger.Errorw("failed to parse player id", "error", err)
		return
	}

	c.handleConnectionMessage(ctx, pId, msg.PlayerUsername, false)
}

func (c *consumer) handleConnectionMessage(ctx context.Context, pId uuid.UUID, username string, joined bool) {
	conns, err := c.repo.GetFriendConnections(ctx, pId)
	if err != nil {
		c.logger.Errorw("failed to get friend connections", "error", err)
		return
	}

	otherPlayerIds := make([]string, 0, len(conns))
	for _, conn := range conns {
		if conn.PlayerOneId != pId {
			otherPlayerIds = append(otherPlayerIds, conn.PlayerOneId.String())
		} else {
			otherPlayerIds = append(otherPlayerIds, conn.PlayerTwoId.String())
		}
	}

	resp, err := c.playerTracker.GetPlayerServers(ctx, &mcplayer.GetPlayerServersRequest{
		PlayerIds: otherPlayerIds,
	})
	if err != nil {
		c.logger.Errorw("failed to get player servers", "error", err)
		return
	}

	playerServers := resp.PlayerServers

	notifiablePlayerIds := make([]string, 0)
	for playerId := range playerServers {
		notifiablePlayerIds = append(notifiablePlayerIds, playerId)
	}

	if err := c.notif.FriendConnectStatus(ctx, pId, username, joined, notifiablePlayerIds); err != nil {
		c.logger.Errorw("failed to send friend connect status", "error", err)
		return
	}
}
