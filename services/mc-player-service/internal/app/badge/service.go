package badge

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	permmsg "github.com/emortalmc/proto-specs/gen/go/message/permission"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	// AddBadgeToPlayer adds a badge to a player and updates
	// their active badge if necessary
	AddBadgeToPlayer(ctx context.Context, playerId uuid.UUID, badgeId string) error

	// RemoveBadgeFromPlayer removes a player's badge and updates
	// their active badge if necessary
	RemoveBadgeFromPlayer(ctx context.Context, playerId uuid.UUID, badgeId string) error

	// UpdateActiveBadge updates a player's active badge based on the badge
	// they own and their priority. If the player no longer owns their active badge
	// and none are required, the active badge will be set to nil
	UpdateActiveBadge(ctx context.Context, playerId uuid.UUID) error

	// SetActiveBadge sets a player's active badge to the given badge
	SetActiveBadge(ctx context.Context, playerID uuid.UUID, badgeID string) error

	// Kafka Handlers

	HandlePlayerRolesUpdate(ctx context.Context, playerID uuid.UUID, roleID string,
		changeType permmsg.PlayerRolesUpdateMessage_ChangeType)
}

type serviceImpl struct {
	log *zap.SugaredLogger

	repo         repository.BadgeReadWriter
	playerReader repository.PlayerReader
	badgeCfg     config.BadgeConfig
}

func NewService(log *zap.SugaredLogger, badgeRepo repository.BadgeReadWriter, playerReader repository.PlayerReader,
	badgeCfg config.BadgeConfig) Service {
	return &serviceImpl{
		log: log,

		repo:         badgeRepo,
		playerReader: playerReader,
		badgeCfg:     badgeCfg,
	}
}

var (
	AlreadyHasBadgeErr = errors.New("player already has this badge")
	DoesntExistErr     = errors.New("badge does not exist")
)

func (s *serviceImpl) AddBadgeToPlayer(ctx context.Context, playerId uuid.UUID, badgeId string) error {
	badge, ok := s.badgeCfg.Badges[badgeId]
	if !ok {
		return DoesntExistErr
	}

	modCount, err := s.repo.AddPlayerBadge(ctx, playerId, badgeId)
	if err != nil {
		return fmt.Errorf("failed to add player badge: %w", err)
	}

	if modCount == 0 {
		return AlreadyHasBadgeErr
	}

	if badge.Required {
		if err := s.UpdateActiveBadge(ctx, playerId); err != nil {
			return fmt.Errorf("failed to update active badge: %w", err)
		}
	}

	return nil
}

var (
	DoesntHaveBadgeErr = errors.New("player does not have this badge")
)

func (s *serviceImpl) RemoveBadgeFromPlayer(ctx context.Context, playerId uuid.UUID, badgeId string) error {
	player, err := s.playerReader.GetPlayer(ctx, playerId)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	removed := false
	for i, playerBadgeId := range player.Badges {
		if playerBadgeId == badgeId {
			player.Badges = append(player.Badges[:i], player.Badges[i+1:]...)
			removed = true
			break
		}
	}
	if !removed {
		return DoesntHaveBadgeErr
	}

	s.log.Debugw("checking active badge", "activeBadge", player.ActiveBadge, "badgeId", badgeId)
	if player.ActiveBadge != nil && *player.ActiveBadge == badgeId {
		player.ActiveBadge = s.calculateActiveBadge(player.Badges)
	}

	s.log.Debugw("Updating badge", "playerId", playerId, "badges", player.Badges, "activeBadge", player.ActiveBadge)
	if err := s.repo.UpdatePlayerBadgesAndActive(ctx, playerId, player.Badges, player.ActiveBadge); err != nil {
		return fmt.Errorf("failed to update player: %w", err)
	}

	return nil
}

func (s *serviceImpl) UpdateActiveBadge(ctx context.Context, playerId uuid.UUID) error {
	player, err := s.repo.GetBadgePlayer(ctx, playerId)
	if err != nil {
		return fmt.Errorf("failed to get player badge: %w", err)
	}

	highestBadgeId := s.calculateActiveBadge(player.BadgeIDs)

	if err := s.repo.SetActivePlayerBadge(ctx, playerId, highestBadgeId); err != nil {
		return fmt.Errorf("failed to set active badge: %w", err)
	}

	return nil
}

var ErrPlayerDoesntHaveBadge = status.Error(codes.NotFound, "player does not have this badge")

func (s *serviceImpl) SetActiveBadge(ctx context.Context, playerID uuid.UUID, badgeID string) error {
	player, err := s.repo.GetBadgePlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player badge: %w", err)
	}

	for _, loopBadgeID := range player.BadgeIDs {
		if loopBadgeID == badgeID {
			if err := s.repo.SetActivePlayerBadge(ctx, playerID, &badgeID); err != nil {
				return fmt.Errorf("failed to set active badge: %w", err)
			}

			return nil
		}
	}

	return ErrPlayerDoesntHaveBadge
}

func (s *serviceImpl) calculateActiveBadge(badgeIDs []string) *string {
	var highestBadgeId *string
	var highestBadgePriority int

	for _, badgeId := range badgeIDs {
		badge, ok := s.badgeCfg.Badges[badgeId]
		if !ok {
			s.log.Warnw("player has badge that does not exist", "badgeId", badgeId)
			continue
		}

		if badge.Priority > highestBadgePriority {
			highestBadgeId = &badgeId
			highestBadgePriority = badge.Priority
		}
	}

	s.log.Debugw("Calculated active badge", "badgeIDs", badgeIDs, "activeBadge", highestBadgeId)

	return highestBadgeId
}
