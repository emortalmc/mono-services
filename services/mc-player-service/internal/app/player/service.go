package player

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/utils/experience"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/webhook"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"time"
)

type Service interface {
	HandlePlayerConnect(ctx context.Context, time time.Time, playerID uuid.UUID, playerUsername string,
		proxyID string, playerSkin model.PlayerSkin, player model.Player)
	HandlePlayerDisconnect(ctx context.Context, time time.Time, playerID uuid.UUID, playerUsername string)
	HandlePlayerServerSwitch(ctx context.Context, pID uuid.UUID, newServerID string)

	AddExperienceByID(ctx context.Context, playerID uuid.UUID, reason string, amount int) (int, error)
}

type serviceImpl struct {
	log *zap.SugaredLogger

	repo    repository.PlayerReadWriter
	kafkaW  KafkaWriter
	webhook webhook.Webhook
}

func NewService(log *zap.SugaredLogger, cfg config.Config, repo repository.PlayerReadWriter, kafkaW KafkaWriter) Service {
	return &serviceImpl{
		log:     log,
		repo:    repo,
		kafkaW:  kafkaW,
		webhook: webhook.NewWebhook(cfg.DiscordWebhookUrl, log),
	}
}

func (s *serviceImpl) AddExperienceByID(ctx context.Context, playerID uuid.UUID, reason string, amount int) (int, error) {
	newXP, err := s.repo.AddExperienceToPlayer(ctx, playerID, amount)
	if err != nil {
		return 0, fmt.Errorf("failed to add experience to player: %w", err)
	}

	if err := s.repo.CreateExperienceTransaction(ctx, model.ExperienceTransaction{
		ID:       primitive.NewObjectID(),
		PlayerID: playerID,
		Amount:   int64(amount),
		Reason:   reason,
	}); err != nil {
		return 0, fmt.Errorf("failed to create experience transaction: %w", err)
	}

	oldXP := newXP - amount
	oldLevel := experience.XPToLevel(oldXP)
	newLevel := experience.XPToLevel(newXP)

	s.kafkaW.PlayerExperienceChange(ctx, playerID, reason, oldXP, newXP, oldLevel, newLevel)

	return newXP, nil
}
