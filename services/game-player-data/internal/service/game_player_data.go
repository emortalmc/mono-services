package service

import (
	"context"
	"errors"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository"
	"github.com/emortalmc/mono-services/services/game-player-data/internal/repository/model"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/gameplayerdata"
	"github.com/emortalmc/proto-specs/gen/go/model/gameplayerdata"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

type gamePlayerDataService struct {
	pb.UnimplementedGamePlayerDataServiceServer

	repos *repository.GameDataRepoColl
	log   *zap.SugaredLogger
}

func newGamePlayerDataService(repos *repository.GameDataRepoColl, log *zap.SugaredLogger) pb.GamePlayerDataServiceServer {
	return &gamePlayerDataService{
		repos: repos,
		log:   log,
	}
}

func (s *gamePlayerDataService) GetGamePlayerData(ctx context.Context, req *pb.GetGamePlayerDataRequest) (*pb.GetGamePlayerDataResponse, error) {
	pId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player id")
	}

	var data model.GameData
	switch req.GameMode {
	case gameplayerdata.GameDataGameMode_BLOCK_SUMO:
		data, err = s.repos.BlockSumo.Get(ctx, pId)
	case gameplayerdata.GameDataGameMode_MARATHON:
		data, err = s.repos.Marathon.Get(ctx, pId)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported game mode")
	}

	if err != nil {
		return nil, s.createDbErr(err)
	}

	anyData, err := data.ToAnyProto()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert data to proto")
	}

	return &pb.GetGamePlayerDataResponse{
		Data: anyData,
	}, nil
}

func (s *gamePlayerDataService) GetMultipleGamePlayerData(ctx context.Context, req *pb.GetMultipleGamePlayerDataRequest) (*pb.GetMultipleGamePlayerDataResponse, error) {
	pIds := make([]uuid.UUID, len(req.PlayerIds))
	for i, pIdStr := range req.PlayerIds {
		pId, err := uuid.Parse(pIdStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid player id")
		}
		pIds[i] = pId
	}

	var genericData = make([]model.GameData, 0)

	switch req.GameMode {
	case gameplayerdata.GameDataGameMode_BLOCK_SUMO:
		uncastData, err := s.repos.BlockSumo.GetMultiple(ctx, pIds)
		if err != nil {
			return nil, s.createDbErr(err)
		}

		for _, d := range uncastData {
			genericData = append(genericData, d)
		}
	case gameplayerdata.GameDataGameMode_MARATHON:
		uncastData, err := s.repos.Marathon.GetMultiple(ctx, pIds)
		if err != nil {
			return nil, s.createDbErr(err)
		}

		for _, d := range uncastData {
			genericData = append(genericData, d)
		}
	}

	data := make(map[string]*anypb.Any, len(genericData))

	for _, d := range genericData {
		anypb, err := d.ToAnyProto()
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert data to proto")
		}

		data[d.PlayerID().String()] = anypb
	}

	return &pb.GetMultipleGamePlayerDataResponse{
		Data: data,
	}, nil

}

func (s *gamePlayerDataService) createDbErr(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return status.Error(codes.NotFound, "player not found")
	} else {
		s.log.Errorw("failed to get player data", "error", err)
		return status.Error(codes.Internal, "failed to get player data")
	}
}
