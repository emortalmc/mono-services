package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/config"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/parsers"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository/model"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/utils"
	"github.com/emortalmc/proto-specs/gen/go/message/gametracker"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"strings"
	"sync"
	"time"
)

const gamesTopic = "game-tracker"

type consumer struct {
	logger *zap.SugaredLogger
	repo   repository.Repository

	reader *kafka.Reader

	liveHandler     *parserHandler[model.LiveGame]
	historicHandler *parserHandler[model.HistoricGame]
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, cfg config.KafkaConfig, logger *zap.SugaredLogger,
	repo repository.Repository) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.Host},
		GroupID:     "game-tracker",
		GroupTopics: []string{gamesTopic},

		Logger:      kafkautils.CreateLogger(logger),
		ErrorLogger: kafkautils.CreateErrorLogger(logger),
	})

	c := &consumer{
		logger: logger,
		repo:   repo,

		reader: reader,

		liveHandler:     &parserHandler[model.LiveGame]{logger: logger, parsers: parsers.LiveParsers},
		historicHandler: &parserHandler[model.HistoricGame]{logger: logger, parsers: parsers.HistoricParsers},
	}

	handler := kafkautils.NewConsumerHandler(logger, reader)
	handler.RegisterHandler(&gametracker.GameStartMessage{}, c.handleGameStartMessage)
	handler.RegisterHandler(&gametracker.GameUpdateMessage{}, c.handleGameUpdateMessage)
	handler.RegisterHandler(&gametracker.GameFinishMessage{}, c.handleGameFinishMessage)

	logger.Infow("started listening for kafka messages", "topics", reader.Config().GroupTopics)

	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.Run(ctx) // Run is blocking until the context is cancelled
		if err := reader.Close(); err != nil {
			logger.Errorw("failed to close kafka reader", err)
		}
	}()
}

func (c *consumer) handleGameStartMessage(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*gametracker.GameStartMessage)
	commonData := m.CommonData

	id, err := primitive.ObjectIDFromHex(commonData.GameId)
	if err != nil {
		c.logger.Errorw("failed to parse game id", "gameId", commonData.GameId)
		return
	}

	players, err := model.BasicPlayersFromProto(commonData.Players)
	if err != nil {
		c.logger.Errorw("failed to parse players", "players", commonData.Players)
		return
	}

	liveGame := &model.LiveGame{
		Game: &model.Game{
			Id:         id,
			GameModeId: commonData.GameModeId,
			ServerId:   commonData.ServerId,
			StartTime:  utils.Pointer(m.StartTime.AsTime()),
			Players:    players,
		},
		LastUpdated: time.Now(),
	}

	if err := c.liveHandler.handle(m.Content, liveGame); err != nil {
		c.logger.Errorw("failed to handle game content", "game", commonData.GameId, "content", m.Content)
		return
	}

	if err := c.repo.SaveLiveGame(ctx, liveGame); err != nil {
		c.logger.Errorw("failed to save live game", "game", liveGame, "error", err)
		return
	}
}

func (c *consumer) handleGameUpdateMessage(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*gametracker.GameUpdateMessage)
	commonData := m.CommonData

	id, err := primitive.ObjectIDFromHex(commonData.GameId)
	if err != nil {
		c.logger.Errorw("failed to parse game id", "gameId", commonData.GameId)
		return
	}

	liveGame, err := c.repo.GetLiveGame(ctx, id)
	if err != nil {
		c.logger.Errorw("failed to get live game", "gameId", id.Hex(), err)
		return
	}

	// common data start

	players, err := model.BasicPlayersFromProto(commonData.Players)
	if err != nil {
		c.logger.Errorw("failed to parse players", "players", commonData.Players)
		return
	}

	liveGame.Players = players
	liveGame.LastUpdated = time.Now()

	// common data end

	if err := c.liveHandler.handle(m.Content, liveGame); err != nil {
		c.logger.Errorw("failed to handle game content", "game", commonData.GameId, "content", m.Content)
		return
	}

	if err := c.repo.SaveLiveGame(ctx, liveGame); err != nil {
		c.logger.Errorw("failed to save live game", "game", liveGame, "error", err)
	}
}

func (c *consumer) handleGameFinishMessage(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	m := uncastMsg.(*gametracker.GameFinishMessage)
	commonData := m.CommonData

	id, err := primitive.ObjectIDFromHex(commonData.GameId)
	if err != nil {
		c.logger.Errorw("failed to parse game id", "gameId", commonData.GameId)
		return
	}

	liveGame, err := c.repo.GetLiveGame(ctx, id)
	if err != nil {
		c.logger.Errorw("failed to get live game", "game", id, "error", err)
		return
	}

	if err := c.repo.DeleteLiveGame(ctx, id); err != nil {
		c.logger.Errorw("failed to delete live game", "game", id, "error", err)
		return
	}

	players, err := model.BasicPlayersFromProto(commonData.Players)
	if err != nil {
		c.logger.Errorw("failed to parse players", "players", commonData.Players)
		return
	}

	game := &model.HistoricGame{
		Game: &model.Game{
			Id:         id,
			GameModeId: commonData.GameModeId,
			ServerId:   commonData.ServerId,
			StartTime:  liveGame.StartTime,
			Players:    players,
		},
		EndTime: m.EndTime.AsTime(),
	}

	if err := c.historicHandler.handle(m.Content, game); err != nil {
		c.logger.Errorw("failed to handle game content", "error", err, "game", commonData.GameId, "content", m.Content)
		return
	}

	if err := c.repo.SaveHistoricGame(ctx, game); err != nil {
		c.logger.Errorw("failed to save historic game", "game", game, "error", err)
	}
}

type parserHandler[T model.IGame] struct {
	logger *zap.SugaredLogger

	parsers map[proto.Message]func(data proto.Message, game *T) error
}

func (h *parserHandler[T]) handle(content []*anypb.Any, g *T) error {
	unhandledIndexes := make([]bool, len(content)) // every index is false by default

	for i, anyPb := range content {
		anyFullName := strings.SplitAfter(anyPb.TypeUrl, "type.googleapis.com/")[1]

		for key, parser := range h.parsers {
			if anyFullName == string(key.ProtoReflect().Descriptor().FullName()) {
				unmarshaled := key
				if err := anyPb.UnmarshalTo(unmarshaled); err != nil {
					return fmt.Errorf("failed to unmarshal game content: %w", err)
				}

				if err := parser(unmarshaled, g); err != nil {
					return fmt.Errorf("failed to parse game content: %w", err)
				}

				unhandledIndexes[i] = true
				break
			}
		}

		for key, parser := range parsers.DualParsers {
			if anyFullName == string(key.ProtoReflect().Descriptor().FullName()) {
				unmarshaled := key
				if err := anyPb.UnmarshalTo(unmarshaled); err != nil {
					return fmt.Errorf("failed to unmarshal game content: %w", err)
				}

				if err := parser(unmarshaled, (*g).GetGame()); err != nil {
					return fmt.Errorf("failed to parse game content: %w", err)
				}

				unhandledIndexes[i] = true
				break
			}
		}
	}

	// check every index has been processed and warn if not
	for i, b := range unhandledIndexes {
		if !b {
			h.logger.Warnw("unhandled game content index", "index", i, "game", g, "content", content)
		}
	}

	return nil
}
