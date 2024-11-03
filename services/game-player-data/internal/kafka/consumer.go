package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/config"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository/model"
	pbmsg "github.com/emortalmc/proto-specs/gen/go/message/gameplayerdata"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/gameplayerdata"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"sync"
)

const GamePlayerDataTopic = "game-player-data"

type consumer struct {
	logger *zap.SugaredLogger
	repos  *repository.GameDataRepoColl

	reader *kafka.Reader
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, cfg *config.KafkaConfig, logger *zap.SugaredLogger,
	repos *repository.GameDataRepoColl) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		GroupID:     "game-player-data",
		GroupTopics: []string{GamePlayerDataTopic},

		Logger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Infow(fmt.Sprintf(format, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Errorw(fmt.Sprintf(format, args...))
		}),
	})

	c := &consumer{
		logger: logger,
		repos:  repos,

		reader: reader,
	}

	handler := kafkautils.NewConsumerHandler(logger, reader)
	handler.RegisterHandler(&pbmsg.UpdateGamePlayerDataMessage{}, c.handleUpdateGamePlayerDataMessage)

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

func (c *consumer) handleUpdateGamePlayerDataMessage(ctx context.Context, _ *kafka.Message, uncast proto.Message) {
	msg := uncast.(*pbmsg.UpdateGamePlayerDataMessage)

	pId, err := uuid.Parse(msg.PlayerId)
	if err != nil {
		c.logger.Errorw("failed to parse player id", "error", err)
		return
	}

	switch msg.GameMode {
	case pbmodel.GameDataGameMode_BLOCK_SUMO:
		err = c.handleBlockSumoUpdate(ctx, pId, msg)
	case pbmodel.GameDataGameMode_MARATHON:
		err = c.handleMarathonUpdate(ctx, pId, msg)

	default:
		c.logger.Errorw("unsupported game mode", "gameMode", msg.GameMode)
	}

	if err != nil {
		c.logger.Errorw("failed to handle update", "error", err, "playerId", pId, "gameMode", msg.GameMode)
		return
	}
}

func (c *consumer) handleMarathonUpdate(ctx context.Context, pId uuid.UUID, msg *pbmsg.UpdateGamePlayerDataMessage) error {
	gameData, err := c.repos.Marathon.GetOrDefault(ctx, pId, &model.MarathonData{BaseGameData: model.BaseGameData{PlayerId: pId}})
	if err != nil {
		return fmt.Errorf("failed to get block sumo data: %w", err)

	}

	msgData := &pbmodel.V1MarathonData{}

	if err := anypb.UnmarshalTo(msg.Data, msgData, proto.UnmarshalOptions{}); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	for _, path := range msg.DataMask.Paths {
		switch path {
		case "block_palette":
			gameData.BlockPalette = msgData.BlockPalette
		case "time":
			gameData.Time = msgData.Time
		case "animation":
			gameData.Animation = *msgData.Animation
		}
	}

	return c.repos.Marathon.Save(ctx, gameData)
}

func (c *consumer) handleBlockSumoUpdate(ctx context.Context, pId uuid.UUID, msg *pbmsg.UpdateGamePlayerDataMessage) error {
	gameData, err := c.repos.BlockSumo.GetOrDefault(ctx, pId, &model.BlockSumoData{BaseGameData: model.BaseGameData{PlayerId: pId}})
	if err != nil {
		return fmt.Errorf("failed to get block sumo data: %w", err)
	}

	msgData := &pbmodel.V1BlockSumoPlayerData{}

	if err := anypb.UnmarshalTo(msg.Data, msgData, proto.UnmarshalOptions{}); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	for _, path := range msg.DataMask.Paths {
		switch path {
		case "block_slot":
			gameData.BlockSlot = msgData.BlockSlot
		case "shears_slot":
			gameData.ShearsSlot = msgData.ShearsSlot
		}
	}

	return c.repos.BlockSumo.Save(ctx, gameData)
}
