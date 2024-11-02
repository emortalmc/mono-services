package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/app/badge"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/badge"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/badge"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type badgeService struct {
	pb.BadgeManagerServer

	repo     repository.BadgeReader
	badgeSvc badge.Service
	badgeCfg config.BadgeConfig
}

func newBadgeService(repo repository.Repository, badgeSvc badge.Service, badgeCfg config.BadgeConfig) pb.BadgeManagerServer {
	return &badgeService{
		repo:     repo,
		badgeSvc: badgeSvc,
		badgeCfg: badgeCfg,
	}
}

var setActivePlayerBadgeDoesntHaveBadgeErr = panicIfErr(status.New(codes.NotFound, "player does not have this badge").
	WithDetails(&pb.SetActivePlayerBadgeErrorResponse{Reason: pb.SetActivePlayerBadgeErrorResponse_PLAYER_DOESNT_HAVE_BADGE})).Err()

func (s *badgeService) SetActivePlayerBadge(ctx context.Context, request *pb.SetActivePlayerBadgeRequest) (*pb.SetActivePlayerBadgeResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	if err := s.badgeSvc.SetActiveBadge(ctx, playerId, request.BadgeId); err != nil {
		if errors.Is(err, badge.DoesntHaveBadgeErr) {
			return nil, setActivePlayerBadgeDoesntHaveBadgeErr
		} else {
			return nil, status.Error(codes.Internal, "failed to set active badge: "+err.Error())
		}
	}

	return &pb.SetActivePlayerBadgeResponse{}, nil
}

var removeBadgeFromPlayerDoesntHaveBadgeErr = panicIfErr(status.New(codes.NotFound, "player does not have this badge").
	WithDetails(&pb.RemoveBadgeFromPlayerErrorResponse{Reason: pb.RemoveBadgeFromPlayerErrorResponse_PLAYER_DOESNT_HAVE_BADGE})).Err()

func (s *badgeService) RemoveBadgeFromPlayer(ctx context.Context, request *pb.RemoveBadgeFromPlayerRequest) (*pb.RemoveBadgeFromPlayerResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	err = s.badgeSvc.RemoveBadgeFromPlayer(ctx, playerId, request.BadgeId)
	if err != nil {
		if errors.Is(err, badge.DoesntHaveBadgeErr) {
			return nil, removeBadgeFromPlayerDoesntHaveBadgeErr
		} else {
			return nil, status.Error(codes.Internal, "failed to remove badge from player")
		}
	}

	return &pb.RemoveBadgeFromPlayerResponse{}, nil
}

func (s *badgeService) GetActivePlayerBadge(ctx context.Context, request *pb.GetActivePlayerBadgeRequest) (*pb.GetActivePlayerBadgeResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	badgeId, err := s.repo.GetActivePlayerBadge(ctx, playerId)
	if err != nil {
		return nil, fmt.Errorf("failed to get active player badge: %w", err)
	}
	if badgeId == nil {
		return nil, status.Error(codes.NotFound, "player does not have any badge")
	}

	badge, ok := s.badgeCfg.Badges[*badgeId]
	if !ok {
		return nil, fmt.Errorf("failed to resolve badgeId to config")
	}

	return &pb.GetActivePlayerBadgeResponse{
		Badge: badge.ToProto(),
	}, nil
}

func (s *badgeService) GetPlayerBadges(ctx context.Context, request *pb.GetPlayerBadgesRequest) (*pb.GetPlayerBadgesResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	player, err := s.repo.GetBadgePlayer(ctx, playerId)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	badges := make([]*pbmodel.Badge, len(player.BadgeIDs))
	for i, badgeId := range player.BadgeIDs {
		b, ok := s.badgeCfg.Badges[badgeId]
		if !ok {
			return nil, fmt.Errorf("failed to resolve badgeId to config")
		}

		badges[i] = b.ToProto()
	}

	return &pb.GetPlayerBadgesResponse{
		Badges:        badges,
		ActiveBadgeId: player.ActiveBadge,
	}, nil
}

var addBadgeToPlayerAlreadyHasBadgeErr = panicIfErr(status.New(codes.AlreadyExists, "player already has this badge").
	WithDetails(&pb.AddBadgeToPlayerErrorResponse{Reason: pb.AddBadgeToPlayerErrorResponse_PLAYER_ALREADY_HAS_BADGE})).Err()

func (s *badgeService) AddBadgeToPlayer(ctx context.Context, request *pb.AddBadgeToPlayerRequest) (*pb.AddBadgeToPlayerResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	err = s.badgeSvc.AddBadgeToPlayer(ctx, playerId, request.BadgeId)
	if err != nil {
		if errors.Is(err, badge.AlreadyHasBadgeErr) {
			return nil, addBadgeToPlayerAlreadyHasBadgeErr
		} else if errors.Is(err, badge.DoesntExistErr) {
			return nil, status.Error(codes.InvalidArgument, "badge does not exist")
		} else {
			return nil, status.Error(codes.Internal, "failed to add badge to player")
		}
	}

	return &pb.AddBadgeToPlayerResponse{}, nil
}

func (s *badgeService) GetBadges(context.Context, *pb.GetBadgesRequest) (*pb.GetBadgesResponse, error) {
	badges := make([]*pbmodel.Badge, 0, len(s.badgeCfg.Badges))
	for _, badge := range s.badgeCfg.Badges {
		badges = append(badges, badge.ToProto())
	}

	return &pb.GetBadgesResponse{
		Badges: badges,
	}, nil
}

func panicIfErr[T any](thing T, err error) T {
	if err != nil {
		panic(err)
	}
	return thing
}
