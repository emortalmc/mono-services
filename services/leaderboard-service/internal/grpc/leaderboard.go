package grpc

import (
	"context"
	"github.com/emortalmc/mono-services/services/leaderboard-service/internal/repository"
	lbpb "github.com/emortalmc/proto-specs/gen/go/grpc/leaderboard"
)

type leaderboardService struct {
	lbpb.LeaderboardServer
	
	repo repository.Repository
}

func newLeaderboardService(repo repository.Repository) lbpb.LeaderboardServer {
	return &leaderboardService{
		repo: repo,
	}
}

func (s *leaderboardService) GetEntries(ctx context.Context, req *lbpb.GetEntriesRequest) (*lbpb.GetEntriesResponse, error) {
	req.Period
	req.Id
	req.Limit
	req.Offset
	req.PeriodRolling
	req.SortOverride
	req.UuidFilter
}
