package grpc

import (
	"context"
	"errors"
	"fmt"
	partySvc "github.com/emortalmc/mono-services/services/party-manager/internal/app/party"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/party"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/party"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type partyService struct {
	pb.UnimplementedPartyServiceServer

	r   partySvc.Reader
	svc *partySvc.Service
}

func newPartyService(r partySvc.Reader, svc *partySvc.Service) pb.PartyServiceServer {
	return &partyService{
		r:   r,
		svc: svc,
	}
}

var (
	partyNotFoundErr    = status.New(codes.NotFound, "party not found").Err()
	playerNotInPartyErr = status.New(codes.NotFound, "player not in party").Err()
)

func invalidFieldErr(field string) error {
	return status.New(codes.InvalidArgument, fmt.Sprintf("invalid %s", field)).Err()
}

var (
	emptyNotLeaderErr = panicIfErr(status.New(codes.PermissionDenied, "player is not the party leader").
		WithDetails(&pb.EmptyPartyErrorResponse{ErrorType: pb.EmptyPartyErrorResponse_NOT_LEADER})).Err()
)

func (s *partyService) EmptyParty(ctx context.Context, request *pb.EmptyPartyRequest) (resp *pb.EmptyPartyResponse, err error) {
	playerId := uuid.Nil
	if request.GetPlayerId() != "" {
		playerId, err = uuid.Parse(request.GetPlayerId())
		if err != nil {
			return nil, invalidFieldErr("player id")
		}
	}

	var party *model.Party
	if request.GetPartyId() != "" {
		partyId, err := primitive.ObjectIDFromHex(request.GetPartyId())
		if err != nil {
			return nil, invalidFieldErr("party id")
		}
		party, err = s.r.GetPartyByID(ctx, partyId)
		if err != nil {
			return nil, err
		}
	} else {
		party, err = s.r.GetPartyByMemberID(ctx, playerId)
		if err != nil {
			return nil, err
		}
	}

	// Perform permission checks as request is made on behalf of a player.
	if playerId != uuid.Nil {
		if party.LeaderId != playerId {
			return nil, emptyNotLeaderErr
		}
	}

	if err := s.svc.EmptyParty(ctx, party, false); err != nil {
		return nil, fmt.Errorf("failed to empty party: %w", err)
	}

	return &pb.EmptyPartyResponse{}, nil
}

var (
	setOpenPartyNotLeaderErr = panicIfErr(status.New(codes.PermissionDenied, "player is not the party leader").
		WithDetails(&pb.SetOpenPartyErrorResponse{ErrorType: pb.SetOpenPartyErrorResponse_NOT_LEADER})).Err()
)

func (s *partyService) SetOpenParty(ctx context.Context, request *pb.SetOpenPartyRequest) (*pb.SetOpenPartyResponse, error) {
	playerId, err := uuid.Parse(request.GetPlayerId())
	if err != nil {
		return nil, invalidFieldErr("player_id")
	}

	party, err := s.r.GetPartyByMemberID(ctx, playerId)
	if err != nil {
		return nil, err
	}

	if party.LeaderId != playerId {
		return nil, setOpenPartyNotLeaderErr
	}

	if err := s.svc.SetOpenParty(ctx, party.ID, request.GetOpen()); err != nil {
		return nil, fmt.Errorf("failed to set party open: %w", err)
	}

	return &pb.SetOpenPartyResponse{}, nil
}

func (s *partyService) GetParty(ctx context.Context, request *pb.GetPartyRequest) (*pb.GetPartyResponse, error) {
	if request.GetPartyId() != "" {
		partyId, err := primitive.ObjectIDFromHex(request.GetPartyId())
		if err != nil {
			return nil, invalidFieldErr("party_id")
		}
		party, err := s.r.GetPartyByID(ctx, partyId)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, partyNotFoundErr
			}
			return nil, err
		}
		return &pb.GetPartyResponse{
			Party: party.ToProto(),
		}, nil
	} else {
		playerId, err := uuid.Parse(request.GetPlayerId())
		if err != nil {
			return nil, invalidFieldErr("player_id")
		}

		party, err := s.r.GetPartyByMemberID(ctx, playerId)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, playerNotInPartyErr
			}
			return nil, err
		}

		return &pb.GetPartyResponse{
			Party: party.ToProto(),
		}, nil
	}
}

func (s *partyService) GetPartyInvites(ctx context.Context, request *pb.GetPartyInvitesRequest) (*pb.GetPartyInvitesResponse, error) {
	var partyId primitive.ObjectID
	var err error

	if request.GetPartyId() != "" {
		partyId, err = primitive.ObjectIDFromHex(request.GetPartyId())
		if err != nil {
			return nil, invalidFieldErr("party_id")
		}
	} else {
		playerId, err := uuid.Parse(request.GetPlayerId())
		if err != nil {
			return nil, invalidFieldErr("player_id")
		}

		partyId, err = s.r.GetPartyIdByMemberId(ctx, playerId)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, playerNotInPartyErr
			}
			return nil, err
		}
	}

	invites, err := s.r.GetPartyInvitesByPartyId(ctx, partyId)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, partyNotFoundErr
		}
		return nil, err
	}

	inviteProtos := make([]*pbmodel.PartyInvite, 0, len(invites))
	for _, invite := range invites {
		inviteProtos = append(inviteProtos, invite.ToProto())
	}

	return &pb.GetPartyInvitesResponse{
		Invites: inviteProtos,
	}, nil
}

var (
	inviteMustBeLeaderErr = panicIfErr(status.New(codes.PermissionDenied, "player must be leader").
				WithDetails(&pb.InvitePlayerErrorResponse{ErrorType: pb.InvitePlayerErrorResponse_NO_PERMISSION})).Err()

	inviteAlreadyInvitedErr = panicIfErr(status.New(codes.AlreadyExists, "player is already invited").
				WithDetails(&pb.InvitePlayerErrorResponse{ErrorType: pb.InvitePlayerErrorResponse_TARGET_ALREADY_INVITED})).Err()

	inviteTargetInSelfPartyErr = panicIfErr(status.New(codes.AlreadyExists, "target is already in the party").
					WithDetails(&pb.InvitePlayerErrorResponse{ErrorType: pb.InvitePlayerErrorResponse_TARGET_ALREADY_IN_SELF_PARTY})).Err()
)

// InvitePlayer invites a player to the party of the inviter
// If the inviter is not in a party, one is implicitly created
func (s *partyService) InvitePlayer(ctx context.Context, request *pb.InvitePlayerRequest) (*pb.InvitePlayerResponse, error) {
	issuerId, err := uuid.Parse(request.IssuerId)
	if err != nil {
		return nil, invalidFieldErr("issuer_id")
	}

	targetId, err := uuid.Parse(request.TargetId)
	if err != nil {
		return nil, invalidFieldErr("target_id")
	}

	invite, err := s.svc.InvitePlayer(ctx, issuerId, request.IssuerUsername, targetId, request.TargetUsername)
	if err != nil {
		if errors.Is(err, partySvc.ErrExecutorNotLeader) {
			return nil, inviteMustBeLeaderErr
		}
		if errors.Is(err, partySvc.ErrTargetAndSelfSameParty) {
			return nil, inviteTargetInSelfPartyErr
		}
		if errors.Is(err, partySvc.ErrTargetAlreadyInvited) {
			return nil, inviteAlreadyInvitedErr
		}
	}

	return &pb.InvitePlayerResponse{Invite: invite.ToProto()}, nil
}

var (
	joinPartyAlreadyInSamePartyErr = panicIfErr(status.New(codes.AlreadyExists, "player is already in a party").
					WithDetails(&pb.JoinPartyErrorResponse{ErrorType: pb.JoinPartyErrorResponse_ALREADY_IN_SAME_PARTY})).Err()

	joinPartyNotInvitedErr = panicIfErr(status.New(codes.PermissionDenied, "player is not invited to the party").
				WithDetails(&pb.JoinPartyErrorResponse{ErrorType: pb.JoinPartyErrorResponse_NOT_INVITED})).Err()
)

func (s *partyService) JoinParty(ctx context.Context, request *pb.JoinPartyRequest) (*pb.JoinPartyResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, invalidFieldErr("player_id")
	}

	targetPlayerId, err := uuid.Parse(request.TargetPlayerId)
	if err != nil {
		return nil, invalidFieldErr("target_player_id")
	}

	party, err := s.svc.JoinPartyByMemberID(ctx, playerId, request.PlayerUsername, targetPlayerId)
	if err != nil {
		if errors.Is(err, partySvc.ErrNotInvited) {
			return nil, joinPartyNotInvitedErr
		}
		if errors.Is(err, partySvc.ErrAlreadyInSameParty) {
			return nil, joinPartyAlreadyInSamePartyErr
		}

		return nil, fmt.Errorf("failed to join party: %w", err)
	}

	return &pb.JoinPartyResponse{
		Party: party.ToProto(),
	}, nil
}

var (
	leaveIsOnlyMember = panicIfErr(status.New(codes.FailedPrecondition, "player is the leader of the party").
		WithDetails(&pb.LeavePartyErrorResponse{ErrorType: pb.LeavePartyErrorResponse_CANNOT_LEAVE_ONLY_MEMBER})).Err()
)

func (s *partyService) LeaveParty(ctx context.Context, request *pb.LeavePartyRequest) (*pb.LeavePartyResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, invalidFieldErr("player_id")
	}

	if _, err := s.svc.RemovePlayerFromParty(ctx, playerId, true); err != nil {
		if errors.Is(err, partySvc.ErrPlayerIsOnlyMember) {
			return nil, leaveIsOnlyMember
		}
		return nil, fmt.Errorf("failed to leave party: %w", err)
	}

	return &pb.LeavePartyResponse{}, nil
}

var (
	kickNotLeaderErr = panicIfErr(status.New(codes.FailedPrecondition, "issuer is not the leader of the party").
				WithDetails(&pb.KickPlayerErrorResponse{ErrorType: pb.KickPlayerErrorResponse_SELF_NOT_LEADER})).Err()

	// todo this error is no longer used. was stupid it existed in the first place oopsies
	//kickTargetIsLeaderErr = panicIfErr(status.New(codes.FailedPrecondition, "target is the leader of the party").
	//			WithDetails(&pb.KickPlayerErrorResponse{ErrorType: pb.KickPlayerErrorResponse_TARGET_IS_LEADER})).Err()

	kickTargetNotInPartyErr = panicIfErr(status.New(codes.FailedPrecondition, "target is not in the party").
				WithDetails(&pb.KickPlayerErrorResponse{ErrorType: pb.KickPlayerErrorResponse_TARGET_NOT_IN_PARTY})).Err()
)

func (s *partyService) KickPlayer(ctx context.Context, request *pb.KickPlayerRequest) (*pb.KickPlayerResponse, error) {
	issuerId, err := uuid.Parse(request.IssuerId)
	if err != nil {
		return nil, invalidFieldErr("issuer_id")
	}

	targetId, err := uuid.Parse(request.TargetId)
	if err != nil {
		return nil, invalidFieldErr("target_id")
	}

	if err := s.svc.KickPlayerFromParty(ctx, issuerId, targetId); err != nil {
		if errors.Is(err, partySvc.ErrExecutorNotLeader) {
			return nil, kickNotLeaderErr
		}
		if errors.Is(err, partySvc.ErrTargetNotInParty) {
			return nil, kickTargetNotInPartyErr
		}
	}

	return &pb.KickPlayerResponse{}, nil
}

var (
	setLeaderSelfNotLeaderErr = panicIfErr(status.New(codes.FailedPrecondition, "issuer is not the leader of the party").
					WithDetails(&pb.SetPartyLeaderErrorResponse{ErrorType: pb.SetPartyLeaderErrorResponse_SELF_NOT_LEADER})).Err()

	// setLeaderTargetNotInPartyErr this only means they are not in the same party, they may be in another party
	setLeaderTargetNotInPartyErr = panicIfErr(status.New(codes.FailedPrecondition, "target is not in the party").
					WithDetails(&pb.SetPartyLeaderErrorResponse{ErrorType: pb.SetPartyLeaderErrorResponse_TARGET_NOT_IN_PARTY})).Err()
)

func (s *partyService) SetPartyLeader(ctx context.Context, request *pb.SetPartyLeaderRequest) (*pb.SetPartyLeaderResponse, error) {
	issuerId, err := uuid.Parse(request.IssuerId)
	if err != nil {
		return nil, invalidFieldErr("issuer_id")
	}

	targetId, err := uuid.Parse(request.TargetId)
	if err != nil {
		return nil, invalidFieldErr("target_id")
	}

	party, err := s.r.GetPartyByMemberID(ctx, issuerId)
	if err != nil {
		return nil, err
	}

	// Check if the issuer is the party leader
	if party.LeaderId != issuerId {
		return nil, setLeaderSelfNotLeaderErr
	}

	if err := s.svc.ChangePartyLeader(ctx, party, targetId); err != nil {
		if errors.Is(err, partySvc.ErrTargetNotInParty) {
			return nil, setLeaderTargetNotInPartyErr
		}
		return nil, fmt.Errorf("failed to set party leader: %w", err)
	}

	return &pb.SetPartyLeaderResponse{}, nil
}

func panicIfErr[T any](thing T, err error) T {
	if err != nil {
		panic(err)
	}
	return thing
}
