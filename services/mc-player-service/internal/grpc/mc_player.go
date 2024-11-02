package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/app/player"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/utils"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/mcplayer"
	"github.com/emortalmc/proto-specs/gen/go/model/common"
	"github.com/emortalmc/proto-specs/gen/go/model/mcplayer"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mcPlayerService struct {
	pb.McPlayerServer

	repo repository.PlayerReader
	svc  player.Service
}

func newMcPlayerService(repo repository.PlayerReader, svc player.Service) pb.McPlayerServer {
	return &mcPlayerService{
		repo: repo,
		svc:  svc,
	}
}

func (s *mcPlayerService) GetPlayer(ctx context.Context, req *pb.GetPlayerRequest) (*pb.GetPlayerResponse, error) {
	pID, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}

	mcPlayer, err := s.getOrCreateMcPlayer(ctx, pID)
	if err != nil {
		return nil, err
	}

	return &pb.GetPlayerResponse{
		Player: mcPlayer,
	}, nil
}

func (s *mcPlayerService) GetPlayers(ctx context.Context, req *pb.GetPlayersRequest) (*pb.GetPlayersResponse, error) {
	var ids []uuid.UUID
	for _, id := range req.PlayerIds {
		pID, err := uuid.Parse(id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", id))
		}
		ids = append(ids, pID)
	}

	players, err := s.repo.GetPlayers(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get players: %w", err)
	}

	var protoPlayers = make([]*mcplayer.McPlayer, len(players))
	for i, p := range players {
		protoPlayers[i], err = s.createMcPlayerFromPlayer(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("error creating player proto: %w", err)
		}
	}

	return &pb.GetPlayersResponse{
		Players: protoPlayers,
	}, nil
}

func (s *mcPlayerService) GetPlayerByUsername(ctx context.Context, req *pb.PlayerUsernameRequest) (*pb.GetPlayerByUsernameResponse, error) {
	p, err := s.repo.GetPlayerByUsername(ctx, req.Username, true)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("player with username %s not found", req.Username))
		}
		return nil, err
	}

	mcPlayer, err := s.createMcPlayerFromPlayer(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("error creating player proto: %w", err)
	}

	return &pb.GetPlayerByUsernameResponse{
		Player: mcPlayer,
	}, nil
}

func (s *mcPlayerService) SearchPlayersByUsername(ctx context.Context, req *pb.SearchPlayersByUsernameRequest) (*pb.SearchPlayersByUsernameResponse, error) {
	filter := &repository.UsernameSearchFilter{
		OnlineOnly: req.FilterMethod == pb.SearchPlayersByUsernameRequest_ONLINE,
		Friends:    req.FilterMethod == pb.SearchPlayersByUsernameRequest_FRIENDS,
	}

	if req.Pageable == nil {
		req.Pageable = &common.Pageable{
			Page: 0,
			Size: utils.PointerOf(uint64(20)),
		}
	} else if req.Pageable.Size == nil || *req.Pageable.Size == 0 {
		req.Pageable.Size = utils.PointerOf(uint64(20))
	}

	var excludedPlayerIDs []uuid.UUID
	if req.ExcludedPlayerIds != nil {
		excludedPlayerIDs = make([]uuid.UUID, 0, len(req.ExcludedPlayerIds))

		for _, id := range req.ExcludedPlayerIds {
			pId, err := uuid.Parse(id)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", id))
			}
			excludedPlayerIDs = append(excludedPlayerIDs, pId)
		}
	}

	players, pageData, err := s.repo.SearchPlayersByUsername(ctx, req.SearchUsername, req.Pageable, filter, excludedPlayerIDs)
	if err != nil {
		return nil, fmt.Errorf("error searching for players: %w", err)
	}

	var protoPlayers = make([]*mcplayer.McPlayer, len(players))
	for i, p := range players {
		protoPlayers[i], err = s.createMcPlayerFromPlayer(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("error creating player proto: %w", err)
		}
	}

	return &pb.SearchPlayersByUsernameResponse{
		Players:  protoPlayers,
		PageData: pageData,
	}, nil
}

func (s *mcPlayerService) GetLoginSessions(ctx context.Context, req *pb.GetLoginSessionsRequest) (*pb.LoginSessionsResponse, error) {
	pId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}

	sessions, err := s.repo.GetLoginSessions(ctx, pId, req.Pageable)
	if err != nil {
		return nil, fmt.Errorf("error getting login sessions: %w", err)
	}

	var protoSessions = make([]*mcplayer.LoginSession, len(sessions))
	for i, s := range sessions {
		protoSessions[i] = s.ToProto()
	}

	return &pb.LoginSessionsResponse{
		Sessions: protoSessions,
	}, nil
}

func (s *mcPlayerService) GetStatTotalUniquePlayers(ctx context.Context, _ *pb.GetStatTotalUniquePlayersRequest) (*pb.GetStatTotalUniquePlayersResponse, error) {
	count, err := s.repo.GetTotalUniquePlayers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting total unique players: %w", err)
	}

	return &pb.GetStatTotalUniquePlayersResponse{Count: count}, nil
}

func (s *mcPlayerService) GetStatTotalPlaytime(ctx context.Context, _ *pb.GetStatTotalPlaytimeRequest) (*pb.GetStatTotalPlaytimeResponse, error) {
	count, err := s.repo.GetTotalPlaytimeHours(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting total playtime: %w", err)
	}

	return &pb.GetStatTotalPlaytimeResponse{PlaytimeHours: count}, nil
}

func (s *mcPlayerService) AddExperienceToPlayers(ctx context.Context, req *pb.AddExperienceToPlayersRequest) (*pb.AddExperienceToPlayersResponse, error) {
	ids := make([]uuid.UUID, len(req.PlayerIds))
	for i, id := range req.PlayerIds {
		pId, err := uuid.Parse(id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", id))
		}

		ids[i] = pId
	}

	newXPs := make(map[string]uint64, len(ids))

	for _, id := range ids {
		newXP, err := s.svc.AddExperienceByID(ctx, id, req.Reason, int(req.Experience))

		if err != nil {
			return nil, fmt.Errorf("error adding experience to player %s: %w", id.String(), err)
		}

		newXPs[id.String()] = uint64(newXP)
	}

	return &pb.AddExperienceToPlayersResponse{
		Experience: newXPs,
	}, nil
}

func (s *mcPlayerService) GetPlayerExperience(ctx context.Context, req *pb.GetPlayerExperienceRequest) (*pb.GetPlayerExperienceResponse, error) {
	panic("implement me")
}

func (s *mcPlayerService) getOrCreateMcPlayer(ctx context.Context, pId uuid.UUID) (*mcplayer.McPlayer, error) {
	p, err := s.repo.GetPlayer(ctx, pId)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("player with id %s not found", pId.String()))
		}
		return nil, err
	}

	return s.createMcPlayerFromPlayer(ctx, p)
}

func (s *mcPlayerService) createMcPlayerFromPlayer(ctx context.Context, p model.Player) (*mcplayer.McPlayer, error) {
	var session model.LoginSession
	if p.CurrentServer != nil {
		var err error
		session, err = s.repo.GetCurrentLoginSession(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("error getting current login session (id: %s): %w", p.ID, err)
		}
	}

	return p.ToProto(session), nil
}
