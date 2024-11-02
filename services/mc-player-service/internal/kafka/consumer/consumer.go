package kafkaConsumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/app/badge"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/app/player"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	permmsg "github.com/emortalmc/proto-specs/gen/go/message/permission"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
)

const connectionsTopic = "mc-connections"
const permissionsTopic = "permission-manager"

type consumer struct {
	log       *zap.SugaredLogger
	repo      repository.PlayerReader
	badgeSvc  badge.Service
	playerSvc player.Service

	reader *kafka.Reader
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, config config.Config, log *zap.SugaredLogger, repo repository.Repository,
	badgeSvc badge.Service, playerSvc player.Service) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", config.Kafka.Host, config.Kafka.Port)},
		GroupID:     "mc-player-service",
		GroupTopics: []string{connectionsTopic, permissionsTopic},

		Logger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			log.Infow(fmt.Sprintf(format, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			log.Errorw(fmt.Sprintf(format, args...))
		}),
	})

	c := &consumer{
		log:  log,
		repo: repo,

		badgeSvc:  badgeSvc,
		playerSvc: playerSvc,

		reader: reader,
	}

	handler := kafkautils.NewConsumerHandler(log, reader)
	handler.RegisterHandler(&common.PlayerConnectMessage{}, c.handlePlayerConnectMessage)
	handler.RegisterHandler(&common.PlayerDisconnectMessage{}, c.handlePlayerDisconnectMessage)
	handler.RegisterHandler(&common.PlayerSwitchServerMessage{}, c.handlePlayerSwitchServerMessage)
	handler.RegisterHandler(&permmsg.PlayerRolesUpdateMessage{}, c.handlePlayerRolesUpdateMessage)

	log.Infow("starting listening for kafka messages", "topics", reader.Config().GroupTopics)

	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.Run(ctx) // Run is blocking until the context is cancelled
		if err := reader.Close(); err != nil {
			log.Errorw("error closing kafka reader", "error", err)
		}
	}()
}

func (c *consumer) handlePlayerConnectMessage(ctx context.Context, kafkaM *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*common.PlayerConnectMessage)

	pID, err := uuid.Parse(m.PlayerId)
	if err != nil {
		c.log.Errorw("error parsing player id", "error", err)
		return
	}

	p, err := c.repo.GetPlayer(ctx, pID)

	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		c.log.Errorw("error getting player", "error", err)
		return
	}
	// we ignore an ErrNoDocuments - `p` will be empty

	pSkin := model.PlayerSkinFromProto(m.PlayerSkin)

	c.playerSvc.HandlePlayerConnect(ctx, kafkaM.Time, pID, m.PlayerUsername, m.ServerId, pSkin, p)
}

func (c *consumer) handlePlayerDisconnectMessage(ctx context.Context, kafkaMsg *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*common.PlayerDisconnectMessage)

	pID, err := uuid.Parse(m.PlayerId)
	if err != nil {
		c.log.Errorw("error parsing player id", "error", err)
		return
	}

	c.playerSvc.HandlePlayerDisconnect(ctx, kafkaMsg.Time, pID, m.PlayerUsername)
}

func (c *consumer) handlePlayerSwitchServerMessage(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*common.PlayerSwitchServerMessage)

	pID, err := uuid.Parse(m.PlayerId)
	if err != nil {
		c.log.Errorw("error parsing player id", "error", err)
		return
	}

	c.playerSvc.HandlePlayerServerSwitch(ctx, pID, m.ServerId)
}

func (c *consumer) handlePlayerRolesUpdateMessage(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*permmsg.PlayerRolesUpdateMessage)

	pID, err := uuid.Parse(m.PlayerId)
	if err != nil {
		c.log.Errorw("error parsing player id", "error", err)
		return
	}

	c.badgeSvc.HandlePlayerRolesUpdate(ctx, pID, m.RoleId, m.ChangeType)
}
