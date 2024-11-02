package grpc

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/mcplayer"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/mcplayer"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type playerTrackerService struct {
	pb.UnimplementedPlayerTrackerServer

	repo repository.PlayerReader
}

func newPlayerTrackerService(repo repository.PlayerReader) pb.PlayerTrackerServer {
	return &playerTrackerService{
		repo: repo,
	}
}

func (s *playerTrackerService) GetPlayerServers(ctx context.Context, req *pb.GetPlayerServersRequest) (*pb.GetPlayerServersResponse, error) {
	playerIds := make([]uuid.UUID, len(req.PlayerIds))
	for i, id := range req.PlayerIds {
		playerId, err := uuid.Parse(id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid player id")
		}

		playerIds[i] = playerId
	}

	servers, err := s.repo.GetPlayerServers(ctx, playerIds)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get player servers")
	}

	var protoServers = make(map[string]*pbmodel.CurrentServer, len(servers))
	for playerId, server := range servers {
		protoServers[playerId.String()] = server.ToProto()
	}

	return &pb.GetPlayerServersResponse{
		PlayerServers: protoServers,
	}, nil
}
func (s *playerTrackerService) GetServerPlayers(ctx context.Context, req *pb.GetServerPlayersRequest) (*pb.GetServerPlayersResponse, error) {
	players, err := s.repo.GetServerPlayers(ctx, req.ServerId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get server players")
	}

	var protoPlayers = make([]*pbmodel.OnlinePlayer, len(players))
	for i, p := range players {
		protoPlayers[i] = p.ToProto()
	}

	return &pb.GetServerPlayersResponse{
		OnlinePlayers: protoPlayers,
	}, nil
}

// TODO implement caching
func (s *playerTrackerService) GetPlayerCount(ctx context.Context, req *pb.GetPlayerCountRequest) (*pb.GetPlayerCountResponse, error) {
	count, err := s.repo.GetPlayerCount(ctx, req.ServerId, req.FleetNames)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get player count")
	}

	return &pb.GetPlayerCountResponse{Count: count}, nil
}

// TODO implement caching
func (s *playerTrackerService) GetFleetPlayerCounts(ctx context.Context, req *pb.GetFleetsPlayerCountRequest) (*pb.GetFleetsPlayerCountResponse, error) {
	counts, err := s.repo.GetFleetPlayerCounts(ctx, req.FleetNames)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get fleet player counts")
	}

	return &pb.GetFleetsPlayerCountResponse{FleetPlayerCounts: counts}, nil
}

// GetGlobalPlayersSummary note that this won't scale - not the code, the concept in general.
// If we have like 300 players the command (/list ...) will be unusable kekw. Chat output will be too long
func (s *playerTrackerService) GetGlobalPlayersSummary(ctx context.Context, req *pb.GetGlobalPlayersSummaryRequest) (*pb.GetGlobalPlayersSummaryResponse, error) {
	players, err := s.repo.GetOnlinePlayers(ctx, req.ServerId, req.FleetNames)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get scoped online players")
	}

	var protoPlayers = make([]*pbmodel.OnlinePlayer, len(players))
	for i, p := range players {
		protoPlayers[i] = p.ToProto()
	}

	return &pb.GetGlobalPlayersSummaryResponse{Players: protoPlayers}, nil
}
