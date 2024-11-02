package service

import (
	"context"
	"github.com/emortalmc/mono-services/services/message-handler/internal/kafka"
	"github.com/emortalmc/proto-specs/gen/go/grpc/mcplayer"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/messagehandler"
	"github.com/emortalmc/proto-specs/gen/go/grpc/relationship"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type privateMessageService struct {
	pb.MessageHandlerServer

	logger *zap.SugaredLogger

	notif         kafka.Notifier
	rs            relationship.RelationshipClient
	playerTracker mcplayer.PlayerTrackerClient
}

func newMessageHandlerService(logger *zap.SugaredLogger, notif kafka.Notifier, rs relationship.RelationshipClient,
	playerTracker mcplayer.PlayerTrackerClient) pb.MessageHandlerServer {

	return &privateMessageService{
		logger: logger,

		notif:         notif,
		rs:            rs,
		playerTracker: playerTracker,
	}
}

var (
	sendPrivateMessageNotOnlineErr = panicIfErr(status.New(codes.Unavailable, "the player is not online").
					WithDetails(&pb.PrivateMessageErrorResponse{Reason: pb.PrivateMessageErrorResponse_PLAYER_NOT_ONLINE})).Err()

	sendPrivateMessageYouBlockedErr = panicIfErr(status.New(codes.PermissionDenied, "you have blocked this player").
					WithDetails(&pb.PrivateMessageErrorResponse{Reason: pb.PrivateMessageErrorResponse_YOU_BLOCKED})).Err()

	sendPrivateMessagePrivacyBlockedErr = panicIfErr(status.New(codes.PermissionDenied, "you are blocked by this player").
						WithDetails(&pb.PrivateMessageErrorResponse{Reason: pb.PrivateMessageErrorResponse_PRIVACY_BLOCKED})).Err()
)

func (s *privateMessageService) SendPrivateMessage(ctx context.Context, req *pb.PrivateMessageRequest) (*pb.PrivateMessageResponse, error) {
	trackerResp, err := s.playerTracker.GetPlayerServers(ctx, &mcplayer.GetPlayerServersRequest{
		PlayerIds: []string{req.Message.RecipientId},
	})
	if err != nil {
		s.logger.Errorw("failed to get player server", "error", err)
		// don't return an error here. We'd rather send a message to an offline player than fail because the player-tracker is down.
	} else if len(trackerResp.PlayerServers) != 1 {
		return nil, sendPrivateMessageNotOnlineErr
	}

	resp, err := s.rs.IsBlocked(ctx, &relationship.IsBlockedRequest{
		IssuerId: req.Message.SenderId,
		TargetId: req.Message.RecipientId,
	})
	if err != nil {
		return nil, err
	}

	block := resp.GetBlock()

	if block != nil {
		if block.BlockerId == req.Message.SenderId {
			return nil, sendPrivateMessageYouBlockedErr
		} else {
			return nil, sendPrivateMessagePrivacyBlockedErr
		}
	}

	err = s.notif.PrivateMessageCreated(ctx, req.Message)
	if err != nil {
		return nil, err
	}

	return &pb.PrivateMessageResponse{
		Message: req.Message,
	}, nil
}

func panicIfErr[T any](thing T, err error) T {
	if err != nil {
		panic(err)
	}
	return thing
}
