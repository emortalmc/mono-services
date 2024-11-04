package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/party-manager/internal/app/event"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"github.com/emortalmc/mono-services/services/party-manager/internal/utils"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/party"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/party"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type eventService struct {
	pb.UnimplementedEventServiceServer

	svc *event.Service
	r   event.Reader
}

func newEventService(svc *event.Service, reader event.Reader) pb.EventServiceServer {
	return &eventService{
		svc: svc,
		r:   reader,
	}
}

func (e *eventService) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*pb.CreateEventResponse, error) {
	reqEvent, err := eventFromRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create reqEvent: %w", err)
	}

	if err := e.svc.CreateEvent(ctx, reqEvent); err != nil {
		if errors.Is(err, event.ErrEventAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "event already exists")
		}

		return nil, fmt.Errorf("failed to create event: %w", err)

	}

	return &pb.CreateEventResponse{
		Event: reqEvent.ToProto(),
	}, nil
}

func eventFromRequest(req *pb.CreateEventRequest) (*model.Event, error) {
	ownerId, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	var displayTime, startTime *time.Time
	if req.DisplayTime != nil {
		displayTime = utils.PointerOf(req.DisplayTime.AsTime())
	}
	if req.StartTime != nil {
		startTime = utils.PointerOf(req.StartTime.AsTime())
	}

	return &model.Event{
		ID:            req.EventId,
		OwnerID:       ownerId,
		OwnerUsername: req.OwnerUsername,
		Skin: model.PlayerSkin{
			Texture:   req.OwnerSkin.Texture,
			Signature: req.OwnerSkin.Signature,
		},
		DisplayTime: displayTime,
		StartTime:   startTime,
	}, nil
}

func (e *eventService) UpdateEvent(ctx context.Context, req *pb.UpdateEventRequest) (*pb.UpdateEventResponse, error) {
	var displayTime, startTime *time.Time
	if req.DisplayTime != nil {
		displayTime = utils.PointerOf(req.DisplayTime.AsTime())
	}
	if req.StartTime != nil {
		startTime = utils.PointerOf(req.StartTime.AsTime())
	}

	res, err := e.svc.UpdateEvent(ctx, req.EventId, displayTime, startTime)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "event not found")
		}

		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return &pb.UpdateEventResponse{
		Event: res.ToProto(),
	}, nil
}

var errNoCurrentEvent = panicIfErr(status.New(codes.NotFound, "no current event").
	WithDetails(&pb.DeleteEventErrorResponse{ErrorType: pb.DeleteEventErrorResponse_NO_CURRENT_EVENT})).Err()

var errEventNotFound = panicIfErr(status.New(codes.NotFound, "event not found").
	WithDetails(&pb.DeleteEventErrorResponse{ErrorType: pb.DeleteEventErrorResponse_NOT_FOUND})).Err()

func (e *eventService) DeleteEvent(ctx context.Context, in *pb.DeleteEventRequest) (*pb.DeleteEventResponse, error) {
	if in.EventId == nil {
		if err := e.svc.DeleteCurrentEvent(ctx); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, errNoCurrentEvent
			}

			return nil, fmt.Errorf("failed to delete current event: %w", err)
		}

		return &pb.DeleteEventResponse{}, nil
	}

	if err := e.svc.DeleteEventByID(ctx, in.GetEventId()); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errEventNotFound
		}

		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return &pb.DeleteEventResponse{}, nil
}

func (e *eventService) ListEvents(ctx context.Context, in *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	events, err := e.r.ListEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	protoEvents := make([]*pbmodel.EventData, len(events))
	for i, event := range events {
		protoEvents[i] = event.ToProto()
	}

	return &pb.ListEventsResponse{
		Events: protoEvents,
	}, nil
}
