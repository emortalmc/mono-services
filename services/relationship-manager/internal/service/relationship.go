package service

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/kafka"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/repository"
	"github.com/emortalmc/mono-services/services/relationship-manager/internal/repository/model"
	"github.com/emortalmc/proto-specs/gen/go/grpc/relationship"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type relationshipService struct {
	relationship.RelationshipServer

	ctx context.Context

	logger *zap.SugaredLogger

	repo  repository.Repository
	notif kafka.Notifier
}

// newPermissionService creates a new permission service
// - Takes context as a request's context cannot be used (it will cancel before the kafka message is sent)
func newPermissionService(ctx context.Context, repo repository.Repository, logger *zap.SugaredLogger,
	notif kafka.Notifier) relationship.RelationshipServer {
	return &relationshipService{
		ctx: ctx,

		logger: logger,

		repo:  repo,
		notif: notif,
	}
}

func (s *relationshipService) AddFriend(ctx context.Context, req *relationship.AddFriendRequest) (*relationship.AddFriendResponse, error) {
	senderId, err := uuid.Parse(req.Request.SenderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.Request.SenderId))
	}
	targetId, err := uuid.Parse(req.Request.TargetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.Request.TargetId))
	}

	alrFriends, err := s.repo.AreFriends(ctx, senderId, targetId)
	if err != nil {
		return nil, err
	}
	if alrFriends {
		return &relationship.AddFriendResponse{
			Result: relationship.AddFriendResponse_ALREADY_FRIENDS,
		}, nil
	}

	blocks, err := s.repo.GetMutualBlocks(ctx, senderId, targetId)
	if err != nil {
		return nil, err
	}
	if len(blocks) > 0 {
		for _, block := range blocks {
			if block.ContainsPlayer(senderId) {
				return &relationship.AddFriendResponse{
					Result: relationship.AddFriendResponse_YOU_BLOCKED,
				}, nil
			}
		}

		return &relationship.AddFriendResponse{
			Result: relationship.AddFriendResponse_PRIVACY_BLOCKED,
		}, nil
	}

	alrRequested, err := s.repo.DoesPendingFriendConnectionExist(ctx, senderId, targetId)
	if err != nil {
		return nil, err
	}
	if alrRequested {
		return &relationship.AddFriendResponse{
			Result: relationship.AddFriendResponse_ALREADY_REQUESTED,
		}, nil
	}

	// Whether they have sent a friend request to you already
	oppositeRequest, err := s.repo.DoesPendingFriendConnectionExist(ctx, targetId, senderId)
	if err != nil {
		return nil, err
	}

	if oppositeRequest {
		// create friendship
		err = s.repo.CreateFriendConnection(ctx, &model.FriendConnection{
			Id:          primitive.NewObjectID(),
			PlayerOneId: senderId,
			PlayerTwoId: targetId,
		})
		if err != nil {
			return nil, err
		}

		// delete pending friend connection
		err = s.repo.DeletePendingFriendConnection(ctx, senderId, targetId)
		if err != nil {
			return nil, err
		}

		go func() {
			err := s.notif.FriendAdded(s.ctx, senderId, targetId, req.Request.SenderUsername)
			if err != nil {
				s.logger.Errorf("failed to send friend added notification to %s: %v", targetId, err)
			}
		}()

		return &relationship.AddFriendResponse{
			Result:       relationship.AddFriendResponse_FRIEND_ADDED,
			FriendsSince: timestamppb.Now(),
		}, nil
	} else {
		friendConn := model.PendingFriendConnection{
			Id:          primitive.NewObjectID(),
			RequesterId: senderId,
			TargetId:    targetId,
		}
		err = s.repo.CreatePendingFriendConnection(ctx, &friendConn)
		if err != nil {
			return nil, err
		}

		go func() {
			err := s.notif.FriendRequest(s.ctx, req.Request)
			if err != nil {
				s.logger.Errorf("failed to send friend request notification to %s: %v", targetId, err)
			}
		}()

		return &relationship.AddFriendResponse{
			Result: relationship.AddFriendResponse_REQUEST_SENT,
		}, nil
	}
}

func (s *relationshipService) RemoveFriend(ctx context.Context, req *relationship.RemoveFriendRequest) (*relationship.RemoveFriendResponse, error) {
	senderId, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.SenderId))
	}
	targetId, err := uuid.Parse(req.TargetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.TargetId))
	}

	err = s.repo.DeleteFriendConnection(ctx, senderId, targetId)
	if err != nil {
		if err == repository.NotFriendsError {
			return &relationship.RemoveFriendResponse{
				Result: relationship.RemoveFriendResponse_NOT_FRIENDS,
			}, nil
		}
		return nil, err
	}

	err = s.notif.FriendRemoved(s.ctx, senderId, targetId)
	if err != nil {
		return nil, err
	}

	return &relationship.RemoveFriendResponse{
		Result: relationship.RemoveFriendResponse_REMOVED,
	}, nil
}

func (s *relationshipService) DenyFriendRequest(ctx context.Context, req *relationship.DenyFriendRequestRequest) (*relationship.DenyFriendRequestResponse, error) {
	senderId, err := uuid.Parse(req.IssuerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.IssuerId))
	}
	targetId, err := uuid.Parse(req.TargetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.TargetId))
	}

	err = s.repo.DeletePendingFriendConnection(ctx, senderId, targetId)
	if err != nil {
		if err == repository.NoFriendRequestError {
			return &relationship.DenyFriendRequestResponse{
				Result: relationship.DenyFriendRequestResponse_NO_REQUEST,
			}, nil
		}
		return nil, err
	}

	return &relationship.DenyFriendRequestResponse{
		Result: relationship.DenyFriendRequestResponse_DENIED,
	}, nil
}

func (s *relationshipService) MassDenyFriendRequest(ctx context.Context, req *relationship.MassDenyFriendRequestRequest) (*relationship.MassDenyFriendRequestResponse, error) {
	senderId, err := uuid.Parse(req.IssuerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.IssuerId))
	}

	var opts repository.DirectionOpts
	if req.Incoming {
		opts = repository.DirectionOpts{
			Incoming: true,
		}
	} else {
		opts = repository.DirectionOpts{
			Outgoing: true,
		}
	}

	count, err := s.repo.DeletePendingFriendConnections(ctx, senderId, &opts)
	if err != nil {
		return nil, err
	}

	return &relationship.MassDenyFriendRequestResponse{RequestsDenied: uint32(count)}, nil
}

func (s *relationshipService) GetFriendList(ctx context.Context, req *relationship.GetFriendListRequest) (*relationship.FriendListResponse, error) {
	playerId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}

	conns, err := s.repo.GetFriendConnections(ctx, playerId)
	if err != nil {
		return nil, err
	}

	var connProtos = make([]*relationship.FriendListResponse_FriendListPlayer, len(conns))

	for i, conn := range conns {
		var friendId uuid.UUID
		if conn.PlayerOneId == playerId {
			friendId = conn.PlayerTwoId
		} else {
			friendId = conn.PlayerOneId
		}

		connProtos[i] = &relationship.FriendListResponse_FriendListPlayer{
			Id:           friendId.String(),
			FriendsSince: timestamppb.New(conn.Id.Timestamp()),
		}
	}

	return &relationship.FriendListResponse{
		Friends: connProtos,
	}, nil
}

func (s *relationshipService) GetPendingFriendRequestList(ctx context.Context, req *relationship.GetPendingFriendRequestListRequest) (*relationship.PendingFriendListResponse, error) {
	playerId, err := uuid.Parse(req.IssuerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.IssuerId))
	}

	conns, err := s.repo.GetPendingFriendConnections(ctx, playerId, repository.DirectionOpts{
		Incoming: req.Incoming,
		Outgoing: !req.Incoming,
	})
	if err != nil {
		return nil, err
	}

	var connProtos = make([]*relationship.PendingFriendListResponse_RequestedFriendPlayer, len(conns))
	for i, conn := range conns {
		connProtos[i] = &relationship.PendingFriendListResponse_RequestedFriendPlayer{
			RequesterId: conn.RequesterId.String(),
			TargetId:    conn.TargetId.String(),
			RequestTime: timestamppb.New(conn.Id.Timestamp()),
		}
	}

	return &relationship.PendingFriendListResponse{
		Requests: connProtos,
	}, nil
}

func (s *relationshipService) CreateBlock(ctx context.Context, req *relationship.CreateBlockRequest) (*relationship.CreateBlockResponse, error) {
	blockProto := req.GetBlock()
	blockerId, err := uuid.Parse(blockProto.BlockerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid blocker id %s", blockProto.BlockerId))
	}
	blockedId, err := uuid.Parse(blockProto.BlockedId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid blocked player id %s", blockProto.BlockedId))
	}

	areFriends, err := s.repo.AreFriends(ctx, blockerId, blockedId)
	if err != nil {
		return nil, err
	}
	if areFriends {
		return &relationship.CreateBlockResponse{
			Result: relationship.CreateBlockResponse_FAILED_FRIENDS,
		}, nil
	}

	err = s.repo.CreatePlayerBlock(ctx, &model.PlayerBlock{Id: primitive.NewObjectID(), BlockedId: blockedId, BlockerId: blockerId})

	if err != nil {
		if err == repository.AlreadyBlockedError {
			return &relationship.CreateBlockResponse{
				Result: relationship.CreateBlockResponse_ALREADY_BLOCKED,
			}, nil
		}
		return nil, err
	}

	return &relationship.CreateBlockResponse{
		Result: relationship.CreateBlockResponse_SUCCESS,
	}, nil
}

func (s *relationshipService) DeleteBlock(ctx context.Context, req *relationship.DeleteBlockRequest) (*relationship.DeleteBlockResponse, error) {
	issuerId, err := uuid.Parse(req.IssuerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid issuer id %s", req.IssuerId))
	}
	targetId, err := uuid.Parse(req.TargetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid target id %s", req.TargetId))
	}

	err = s.repo.DeletePlayerBlock(ctx, issuerId, targetId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("block between %s and %s not found", issuerId, targetId))
		}
		return nil, err
	}

	return &relationship.DeleteBlockResponse{}, nil
}

func (s *relationshipService) IsBlocked(ctx context.Context, req *relationship.IsBlockedRequest) (*relationship.IsBlockedResponse, error) {
	issuerId, err := uuid.Parse(req.IssuerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid issuer id %s", req.IssuerId))
	}
	targetId, err := uuid.Parse(req.TargetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid target id %s", req.TargetId))
	}

	blocks, err := s.repo.GetMutualBlocks(ctx, issuerId, targetId)
	if err != nil {
		return nil, err
	}

	// select the issuer as the block if there is more than 1
	var block *model.PlayerBlock
	if len(blocks) > 1 {
		for _, b := range blocks {
			if b.BlockedId == issuerId {
				block = b
				break
			}
		}
	} else if len(blocks) == 1 {
		block = blocks[0]
	} else {
		return &relationship.IsBlockedResponse{
			Block: nil,
		}, nil
	}

	return &relationship.IsBlockedResponse{
		Block: block.ToProto(),
	}, nil
}

func (s *relationshipService) GetBlockedList(ctx context.Context, req *relationship.GetBlockedListRequest) (*relationship.BlockedListResponse, error) {
	playerId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}

	blocks, err := s.repo.GetPlayerBlocks(ctx, playerId)
	if err != nil {
		return nil, err
	}

	var blockedIds = make([]string, len(blocks))
	for i, block := range blocks {
		blockedIds[i] = block.BlockedId.String()
	}

	return &relationship.BlockedListResponse{
		BlockedPlayerIds: blockedIds,
	}, nil
}
