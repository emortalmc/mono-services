package service

import (
	"context"
	"fmt"
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/kafka"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository/model"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/simplecontroller"
	"github.com/emortalmc/proto-specs/gen/go/grpc/matchmaker"
	pbparty "github.com/emortalmc/proto-specs/gen/go/grpc/party"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
)

type matchmakerService struct {
	matchmaker.UnimplementedMatchmakerServer

	logger        *zap.SugaredLogger
	repo          repository.Repository
	notifier      kafka.Notifier
	cfgController liveconfig.GameModeConfigController

	lobbyController    simplecontroller.SimpleController
	velocityController simplecontroller.SimpleController

	partyService         pbparty.PartyServiceClient
	partySettingsService pbparty.PartySettingsServiceClient
}

func newMatchmakerService(logger *zap.SugaredLogger, repository repository.Repository, notifier kafka.Notifier,
	cfgController liveconfig.GameModeConfigController, lobbyController simplecontroller.SimpleController,
	velocityController simplecontroller.SimpleController,
	partyService pbparty.PartyServiceClient, partySettingsService pbparty.PartySettingsServiceClient) matchmaker.MatchmakerServer {

	return &matchmakerService{
		logger:        logger,
		repo:          repository,
		notifier:      notifier,
		cfgController: cfgController,

		lobbyController:    lobbyController,
		velocityController: velocityController,

		partyService:         partyService,
		partySettingsService: partySettingsService,
	}
}

var (
	queueAlreadyInQueueErr = panicIfErr(status.New(codes.AlreadyExists, "party is already in queue").
				WithDetails(&matchmaker.QueueByPlayerErrorResponse{Reason: matchmaker.QueueByPlayerErrorResponse_ALREADY_IN_QUEUE})).Err()

	queueInvalidGameModeErr = panicIfErr(status.New(codes.InvalidArgument, "invalid game_mode_id").
				WithDetails(&matchmaker.QueueByPlayerErrorResponse{Reason: matchmaker.QueueByPlayerErrorResponse_INVALID_GAME_MODE})).Err()

	queueGameModeDisabledErr = panicIfErr(status.New(codes.InvalidArgument, "game_mode_id is disabled").
					WithDetails(&matchmaker.QueueByPlayerErrorResponse{Reason: matchmaker.QueueByPlayerErrorResponse_GAME_MODE_DISABLED})).Err()

	queueInvalidMapErr = panicIfErr(status.New(codes.InvalidArgument, "invalid map_id").
				WithDetails(&matchmaker.QueueByPlayerErrorResponse{Reason: matchmaker.QueueByPlayerErrorResponse_INVALID_MAP})).Err()

	queuePartyTooLargeErr = panicIfErr(status.New(codes.InvalidArgument, "party is too large").
				WithDetails(&matchmaker.QueueByPlayerErrorResponse{Reason: matchmaker.QueueByPlayerErrorResponse_PARTY_TOO_LARGE})).Err()

	queueNoPermissionErr = panicIfErr(status.New(codes.PermissionDenied, "player does not have permission to queue").
				WithDetails(&matchmaker.QueueByPlayerErrorResponse{Reason: matchmaker.QueueByPlayerErrorResponse_NO_PERMISSION})).Err()
)

// QueueByPlayer requests a player is queued for a game.
// NOTE: A player is always in a party and we clean up when a player changes party.
// Therefore, we only need to check if the player's party is in a queue, not the player themselves.
func (m *matchmakerService) QueueByPlayer(ctx context.Context, request *matchmaker.QueueByPlayerRequest) (*matchmaker.QueueByPlayerResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	// check if game_mode is valid
	modeConfig := m.cfgController.GetCurrentConfig(request.GameModeId)
	if modeConfig == nil {
		return nil, queueInvalidGameModeErr
	}

	if !modeConfig.Enabled {
		return nil, queueGameModeDisabledErr
	}

	// check if map is present
	if request.MapId != nil {
		_, ok := modeConfig.Maps[*request.MapId]
		if !ok {
			return nil, queueInvalidMapErr
		}
		// todo check if player has enough credits
	}

	res, err := m.partyService.GetParty(ctx, &pbparty.GetPartyRequest{Id: &pbparty.GetPartyRequest_PlayerId{PlayerId: playerId.String()}})
	if err != nil {
		return nil, err
	}

	party := res.Party
	partyId, err := primitive.ObjectIDFromHex(party.Id)
	if err != nil {
		return nil, err
	}

	// check if the party is already in a queue
	// if yes, return error
	isInQueue, err := m.repo.IsPartyQueued(ctx, partyId)
	if err != nil {
		return nil, err
	}

	if isInQueue {
		return nil, queueAlreadyInQueueErr
	}

	// check if player is leader of party
	if playerId.String() != party.GetLeaderId() {
		return nil, queueNoPermissionErr
	}

	// party checks
	partyRestrictions := modeConfig.PartyRestrictions
	if partyRestrictions.MaxSize != nil && len(party.Members) > *modeConfig.PartyRestrictions.MaxSize {
		return nil, queuePartyTooLargeErr
	}

	// private game checks
	var privateGame bool
	if request.PrivateGame != nil {
		privateGame = *request.PrivateGame
	} else {
		privateGame = false
	}

	if privateGame {
		// check if there are enough players
		if len(party.Members) < modeConfig.MinPlayers {
			return nil, status.Error(codes.InvalidArgument, "not enough players to create private game")
		}

		// TODO check if player has enough credits
	}

	// Get the party settings
	settingsRes, err := m.partySettingsService.GetPartySettings(ctx,
		&pbparty.GetPartySettingsRequest{Id: &pbparty.GetPartySettingsRequest_PlayerId{PlayerId: party.LeaderId}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get party settings (leaderId: %s): %w", party.LeaderId, err)
	}

	settings := settingsRes.GetSettings()

	autoTeleport := true
	if request.AutoTeleport != nil {
		autoTeleport = *request.AutoTeleport
	}

	memberIds := make([]uuid.UUID, 0)
	for _, member := range party.Members {
		id, err := uuid.Parse(member.Id)
		if err != nil {
			return nil, err
		}

		memberIds = append(memberIds, id)
	}

	partyLeaderId, err := uuid.Parse(party.LeaderId)
	if err != nil {
		return nil, err
	}

	// create ticket
	ticket := model.NewTicket(&partyId, &model.ReducedPartySettings{
		LeaderId:            partyLeaderId,
		DequeueOnDisconnect: settings.DequeueOnDisconnect,
		AllowMemberDequeue:  settings.AllowMemberDequeue,
	}, memberIds, request.GameModeId, autoTeleport)

	err = m.repo.ExecuteTransaction(ctx, func(ctx mongo.SessionContext) error {
		err = m.repo.CreateTicket(ctx, ticket)
		if err != nil {
			return err
		}

		// save player map choice
		queuedPlayers := make([]*model.QueuedPlayer, 0)
		for _, playerId := range memberIds {
			var mapId *string
			if request.MapId != nil && playerId.String() == party.LeaderId {
				mapId = request.MapId
			}

			queuedPlayers = append(queuedPlayers, &model.QueuedPlayer{
				PlayerId: playerId,
				TicketId: ticket.Id,
				MapId:    mapId,
			})
		}

		err = m.repo.CreateQueuedPlayers(ctx, queuedPlayers)
		if err != nil {
			return err
		}

		err = m.notifier.TicketCreated(ctx, ticket)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &matchmaker.QueueByPlayerResponse{}, nil
}

func (m *matchmakerService) SendPlayersToLobby(ctx context.Context, request *matchmaker.SendPlayerToLobbyRequest) (*matchmaker.SendPlayerToLobbyResponse, error) {
	playerIds := make([]uuid.UUID, 0)
	for _, playerId := range request.PlayerIds {
		id, err := uuid.Parse(playerId)
		if err != nil {
			return nil, err
		}

		playerIds = append(playerIds, id)
	}

	// If send parties, retrieve all the necessary player IDs
	if request == nil || *request.SendParties {
		playerIdsMutex := sync.Mutex{}
		partyReqWaitGroup := sync.WaitGroup{}
		partyReqWaitGroup.Add(len(playerIds))

		for _, loopPlayerId := range request.PlayerIds {
			go func(playerId string) {
				defer partyReqWaitGroup.Done()

				resp, err := m.partyService.GetParty(ctx, &pbparty.GetPartyRequest{
					Id: &pbparty.GetPartyRequest_PlayerId{PlayerId: playerId},
				})
				if err != nil {
					m.logger.Errorw("failed to get party", "error", err)
					return
				}

				party := resp.Party
				for _, member := range party.Members {
					id, err := uuid.Parse(member.Id)
					if err != nil {
						m.logger.Errorw("failed to parse party member id", "error", err)
						return
					}

					playerIdsMutex.Lock()
					if !slices.Contains(playerIds, id) {
						playerIds = append(playerIds, id)
					}
					playerIdsMutex.Unlock()
				}
			}(loopPlayerId)
		}

		partyReqWaitGroup.Wait()
	}

	for _, playerId := range playerIds {
		m.lobbyController.QueuePlayer(playerId, true)
	}

	return &matchmaker.SendPlayerToLobbyResponse{}, nil
}

func (m *matchmakerService) LoginQueueByPlayer(_ context.Context, request *matchmaker.LoginQueueByPlayerRequest) (*matchmaker.LoginQueueByPlayerResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	if !request.IsProxy {
		m.lobbyController.QueuePlayer(playerId, false)
	} else {
		m.velocityController.QueuePlayer(playerId, false)
	}

	return &matchmaker.LoginQueueByPlayerResponse{}, nil
}

var (
	dequeueNotInQueueErr = panicIfErr(status.New(codes.NotFound, "player is not in queue").
				WithDetails(&matchmaker.DequeueByPlayerErrorResponse{Reason: matchmaker.DequeueByPlayerErrorResponse_NOT_IN_QUEUE})).Err()

	dequeueNoPermissionErr = panicIfErr(status.New(codes.PermissionDenied, "player does not have permission to dequeue").
				WithDetails(&matchmaker.DequeueByPlayerErrorResponse{Reason: matchmaker.DequeueByPlayerErrorResponse_NO_PERMISSION})).Err()

	dequeueAlreadyDequeuedErr = panicIfErr(status.New(codes.AlreadyExists, "party is already requested for dequeue").
					WithDetails(&matchmaker.DequeueByPlayerErrorResponse{Reason: matchmaker.DequeueByPlayerErrorResponse_ALREADY_MARKED_FOR_DEQUEUE})).Err()
)

// DequeueByPlayer requests a player is dequeued from a game. Note that this player acts on behalf of the party, not themselves.
// NOTE: This DOES NOT actually dequeue the player, it just requests it.
// Not every error will be fed back as it happens lazily.
func (m *matchmakerService) DequeueByPlayer(ctx context.Context, request *matchmaker.DequeueByPlayerRequest) (*matchmaker.DequeueByPlayerResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	ticket, err := m.repo.GetTicketByPlayerId(ctx, playerId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, dequeueNotInQueueErr
		}

		return nil, err
	}

	// check if the player has permission to dequeue
	if playerId != ticket.PartySettings.LeaderId && !ticket.PartySettings.AllowMemberDequeue {
		return nil, dequeueNoPermissionErr
	}

	// TODO we could do a preemptive check to predict if a PendingMatch is being processed and will become a Match
	// if so, we could return an error here to prevent the player from being dequeued

	modCount, err := m.repo.AddTicketDequeueRequest(ctx, ticket.Id)
	if err != nil {
		return nil, err
	}

	if modCount == 0 {
		return nil, dequeueAlreadyDequeuedErr
	}

	return &matchmaker.DequeueByPlayerResponse{}, nil
}

var (
	changeMapInvalidMapErr = panicIfErr(status.New(codes.InvalidArgument, "invalid map_id").
				WithDetails(&matchmaker.ChangePlayerMapVoteErrorResponse{Reason: matchmaker.ChangePlayerMapVoteErrorResponse_INVALID_MAP})).Err()

	changeMapNotInQueueErr = panicIfErr(status.New(codes.NotFound, "player is not in queue").
				WithDetails(&matchmaker.ChangePlayerMapVoteErrorResponse{Reason: matchmaker.ChangePlayerMapVoteErrorResponse_NOT_IN_QUEUE})).Err()
)

func (m *matchmakerService) ChangePlayerMapVote(ctx context.Context, request *matchmaker.ChangePlayerMapVoteRequest) (*matchmaker.ChangePlayerMapVoteResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	ticket, err := m.repo.GetTicketByPlayerId(ctx, playerId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, changeMapNotInQueueErr
		}

		return nil, err
	}

	ok := m.isMapIdValid(ticket.GameModeId, request.MapId)
	if !ok {
		return nil, changeMapInvalidMapErr
	}

	err = m.repo.SetMapIdOfQueuedPlayer(ctx, playerId, request.MapId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, changeMapNotInQueueErr
		}

		return nil, err
	}

	return &matchmaker.ChangePlayerMapVoteResponse{}, nil
}

func (m *matchmakerService) GetPlayerQueueInfo(ctx context.Context, request *matchmaker.GetPlayerQueueInfoRequest) (*matchmaker.GetPlayerQueueInfoResponse, error) {
	playerId, err := uuid.Parse(request.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid player_id")
	}

	ticket, err := m.repo.GetTicketByPlayerId(ctx, playerId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "player is not in queue")
		}

		return nil, err
	}

	queuedPlayer, err := m.repo.GetQueuedPlayerById(ctx, playerId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "player is not in queue")
		}

		return nil, err
	}

	var pbPendingMatch *pb.PendingMatch
	if ticket.InPendingMatch {
		match, err := m.repo.GetPendingMatchByTicketId(ctx, ticket.Id)
		if err == nil {
			pbPendingMatch = match.ToProto()
		} else if err != mongo.ErrNoDocuments {
			return nil, err
		}
	}

	return &matchmaker.GetPlayerQueueInfoResponse{
		Ticket:       ticket.ToProto(),
		QueuedPlayer: queuedPlayer.ToProto(),
		PendingMatch: pbPendingMatch,
	}, nil
}

func (m *matchmakerService) isMapIdValid(modeId string, mapId string) bool {
	cfg := m.cfgController.GetCurrentConfig(modeId)
	if cfg == nil {
		return false
	}

	for _, mapCfg := range cfg.Maps {
		if mapCfg.Id == mapId {
			return true
		}
	}

	return false
}

func panicIfErr[T any](thing T, err error) T {
	if err != nil {
		panic(err)
	}
	return thing
}
