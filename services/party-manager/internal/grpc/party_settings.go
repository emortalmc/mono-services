package grpc

import (
	"context"
	"errors"
	"github.com/emortalmc/mono-services/services/party-manager/internal/app/party"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/party"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type partySettingsService struct {
	pb.PartySettingsServiceServer

	r party.ReadWriter
}

func newPartySettingsService(r party.ReadWriter) pb.PartySettingsServiceServer {
	return &partySettingsService{
		r: r,
	}
}

func (p *partySettingsService) GetPartySettings(ctx context.Context, request *pb.GetPartySettingsRequest) (*pb.GetPartySettingsResponse, error) {
	var playerId uuid.UUID

	if request.GetPartyId() != "" {
		partyId, err := primitive.ObjectIDFromHex(request.GetPartyId())
		if err != nil {
			return nil, status.New(codes.InvalidArgument, "partyId is invalid").Err()
		}

		playerId, err = p.r.GetPartyLeaderByPartyId(ctx, partyId)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, status.New(codes.NotFound, "party does not exist").Err()
			}
			return nil, status.New(codes.Internal, "error getting party").Err()
		}
	} else if request.GetPlayerId() != "" {
		var err error
		playerId, err = uuid.Parse(request.GetPlayerId())
		if err != nil {
			return nil, status.New(codes.InvalidArgument, "playerId is invalid").Err()
		}
	} else {
		return nil, status.New(codes.InvalidArgument, "partyId or playerId must be set").Err()
	}

	settings, err := p.r.GetPartySettings(ctx, playerId)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return &pb.GetPartySettingsResponse{Settings: model.NewPartySettings(playerId).ToProto()}, nil
		}
		return nil, err
	}

	return &pb.GetPartySettingsResponse{
		Settings: settings.ToProto(),
	}, nil
}

func (p *partySettingsService) UpdatePartySettings(ctx context.Context, request *pb.UpdatePartySettingsRequest) (*pb.UpdatePartySettingsResponse, error) {
	playerId, err := uuid.Parse(request.IssuerId)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, "issuerId is invalid").Err()
	}

	settings, err := p.r.GetPartySettings(ctx, playerId)
	if err != nil {
		return nil, err
	}

	if request.DequeueOnDisconnect != nil {
		settings.DequeueOnDisconnect = *request.DequeueOnDisconnect
	}

	if request.AllowMemberDequeue != nil {
		settings.AllowMemberDequeue = *request.AllowMemberDequeue
	}

	if request.AllowMemberInvite != nil {
		settings.AllowMemberInvite = *request.AllowMemberInvite
	}

	err = p.r.UpdatePartySettings(ctx, settings)
	if err != nil {
		return nil, status.New(codes.Internal, "error updating party settings").Err()
	}

	// TODO: Notify

	return &pb.UpdatePartySettingsResponse{}, nil
}
