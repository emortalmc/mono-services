package director

import (
	allocatorv1 "agones.dev/agones/pkg/apis/allocation/v1"
	v1 "agones.dev/agones/pkg/client/clientset/versioned/typed/allocation/v1"
	"context"
	"fmt"
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/gsallocation"
	selector2 "github.com/emortalmc/mono-services/services/kurushimi/internal/gsallocation/selector"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/kafka"
	matchfunction2 "github.com/emortalmc/mono-services/services/kurushimi/internal/matchfunction"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository/model"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/utils"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/utils/protoutils"
	msg "github.com/emortalmc/proto-specs/gen/go/message/matchmaker"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"golang.org/x/exp/rand"
	"reflect"
	"time"
)

type Director interface {
	Start(ctx context.Context)
}

type directorImpl struct {
	logger *zap.SugaredLogger

	repo     repository.Repository
	notifier kafka.Notifier

	allocationClient v1.GameServerAllocationInterface

	configs map[string]*liveconfig.GameModeConfig
}

func New(logger *zap.SugaredLogger, repo repository.Repository, notifier kafka.Notifier,
	allocationClient v1.GameServerAllocationInterface, cfgController liveconfig.GameModeConfigController) Director {

	// Filter for only enabled configs
	configs := cfgController.GetConfigs()
	for i, liveConfig := range configs {
		if !liveConfig.Enabled {
			delete(configs, i)
		}
	}

	d := &directorImpl{
		logger: logger,

		repo:     repo,
		notifier: notifier,

		allocationClient: allocationClient,

		configs: cfgController.GetConfigs(),
	}

	cfgController.AddGlobalUpdateListener(d.onGameModeConfigUpdate)
	return d
}

func (d *directorImpl) Start(ctx context.Context) {
	for _, c := range d.configs {
		go func(config *liveconfig.GameModeConfig) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			for {
				lastRunTime := time.Now()
				d.run(ctx, config)

				// Wait for the next run
				timeSinceLastRun := time.Since(lastRunTime)
				if timeSinceLastRun < config.MatchmakerInfo.Rate {
					time.Sleep(config.MatchmakerInfo.Rate - timeSinceLastRun)
				}
			}
		}(c)
	}
}

func (d *directorImpl) run(ctx context.Context, originalConfig *liveconfig.GameModeConfig) {
	temp := *originalConfig
	config := &temp

	// account for deletions
	err := d.processDequeues(ctx, config)
	if err != nil {
		d.logger.Errorw("failed to process dequeues", "error", err)
		return
	}

	// run match function
	matches, err := d.runMatchFunction(ctx, config)
	if err != nil {
		d.logger.Errorw("failed to run match function", "error", err)
		return
	}

	if len(matches) == 0 {
		return
	}

	d.logger.Debugw("match function returned matches", "count", len(matches))
	d.logger.Debugw("matches", "matches", matches)

	// todo allocate gameservers
}

func (d *directorImpl) processDequeues(ctx context.Context, config *liveconfig.GameModeConfig) error {
	configId := config.Id

	tickets, err := d.repo.GetTicketsWithDequeueRequest(ctx, configId)
	if err != nil {
		return err
	}

	// ticketIdsToUpdate contains the ids of every ticket that isn't in the ticketIdsToDelete slice
	ticketIdsToUpdate := make([]primitive.ObjectID, 0)
	playerIdsToDelete := make([]uuid.UUID, 0)
	ticketIdsToDelete := make([]primitive.ObjectID, 0)

	for _, ticket := range tickets {
		removals := ticket.Removals

		if removals.MarkedForRemoval {
			playerIdsToDelete = append(playerIdsToDelete, ticket.PlayerIds...)
			ticketIdsToDelete = append(ticketIdsToDelete, ticket.Id)
		} else {
			playerIdsToDelete = append(playerIdsToDelete, removals.PlayersForRemoval...)
			ticketIdsToUpdate = append(ticketIdsToUpdate, ticket.Id)
		}
	}

	// Delete tickets
	if len(ticketIdsToDelete) > 0 {
		modified, err := d.repo.DeleteAllTicketsById(ctx, ticketIdsToDelete)
		if err != nil {
			return fmt.Errorf("failed to delete tickets: %w", err)
		}

		if int(modified) != len(ticketIdsToDelete) {
			d.logger.Warnw("deleted tickets count does not match expected count", "deleted", modified, "expected", len(ticketIdsToDelete))
		}

		// Delete tickets from any PendingMatches they are in.
		// NOTE: The modified count is irrelevant as we don't know if they were in any PendingMatches
		_, err = d.repo.RemoveTicketsFromPendingMatchesById(ctx, ticketIdsToDelete)
		if err != nil {
			return fmt.Errorf("failed to remove tickets from pending matches: %w", err)
		}

		// Send Kafka notifications
		for _, ticket := range tickets {
			if ticket.Removals.MarkedForRemoval {
				if err := d.notifier.TicketDeleted(ctx, ticket.ToProto(), msg.TicketDeletedMessage_MANUAL_DEQUEUE); err != nil {
					d.logger.Errorw("failed to send ticket deleted notification", "error", err)
				}
			}
		}
	}

	if len(ticketIdsToUpdate) > 0 {
		// Update tickets removal requests.
		modified, err := d.repo.ResetAllDequeueRequestsById(ctx, ticketIdsToUpdate)
		if err != nil {
			return fmt.Errorf("failed to reset dequeue requests: %w", err)
		}

		if int(modified) != len(ticketIdsToUpdate) {
			d.logger.Warnw("updated tickets count does not match expected count", "updated", modified, "expected", len(ticketIdsToUpdate))
		}
	}

	if len(playerIdsToDelete) > 0 {
		// Delete players
		modified, err := d.repo.DeleteAllQueuedPlayersById(ctx, playerIdsToDelete)
		if err != nil {
			return fmt.Errorf("failed to delete players: %w", err)
		}

		if int(modified) != len(playerIdsToDelete) {
			d.logger.Warnw("deleted players count does not match expected count", "deleted", modified, "expected", len(playerIdsToDelete))
		}
		return nil
	}

	return err
}

func (d *directorImpl) runMatchFunction(ctx context.Context, cfg *liveconfig.GameModeConfig) ([]*pb.Match, error) {
	// NOTE: these tickets are ALL the tickets for this gamemode, even ones already in a PendingMatch
	tickets, err := d.repo.GetTicketsByGameMode(ctx, cfg.Id)
	if err != nil {
		return nil, err
	}

	if len(tickets) != 0 {
		d.logger.Debugw("matchmaker running", "gamemode", cfg.Id, "tickets", len(tickets), "method", cfg.MatchmakerInfo.MatchMethod)
	}

	// make matches
	var matches []*pb.Match
	switch cfg.MatchmakerInfo.MatchMethod {
	case liveconfig.MatchMethodCountdown:
		pendingMatches, err := d.repo.GetPendingMatchesByGameMode(ctx, cfg.Id)
		if err != nil {
			return nil, err
		}

		// Create a pending matches map
		pendingMatchesMap := make(map[primitive.ObjectID]*model.PendingMatch)
		for _, pendingMatch := range pendingMatches {
			pendingMatchesMap[pendingMatch.Id] = pendingMatch
		}

		// Create a ticket map
		ticketMap := make(map[primitive.ObjectID]*model.Ticket)
		for _, ticket := range tickets {
			ticketMap[ticket.Id] = ticket
		}

		// Clean up existing pending matches
		deletedPending := matchfunction2.CountdownRemoveInvalidPendingMatches(pendingMatchesMap, ticketMap, cfg)

		// Delete matches from db and notify with reason cancelled
		if len(deletedPending) > 0 {
			//ticketUpdates := make(map[primitive.ObjectID]bool)

			for _, match := range deletedPending {
				if err := d.notifier.PendingMatchDeleted(ctx, match, msg.PendingMatchDeletedMessage_CANCELLED); err != nil {
					d.logger.Errorw("failed to send pending match deleted notification", "error", err)
				}

				for _, ticketId := range match.TicketIds {
					ticket := ticketMap[ticketId]
					if err := d.notifier.TicketUpdated(ctx, ticket); err != nil {
						d.logger.Errorw("failed to send ticket updated notification", "error", err)
					}
				}

				//for _, ticketId := range match.TicketIds {
				//	ticketUpdates[ticketId] = false
				//}
			}

			// Update the tickets in the DB
			//if _, err := d.repo.MassUpdateTicketInPendingMatch(ctx, ticketUpdates); err != nil {
			//	return nil, fmt.Errorf("failed to update tickets in pending match: %w", err)
			//}

			// Delete the pending matches from the db
			deletedIds := make([]primitive.ObjectID, 0)
			for _, match := range deletedPending {
				deletedIds = append(deletedIds, match.Id)
			}
			if err := d.repo.DeletePendingMatches(ctx, deletedIds); err != nil {
				return nil, err
			}
		}

		createdPending, updatedPending, deletedPending, createdMatches, err := matchfunction2.RunCountdown(d.logger, ticketMap, pendingMatchesMap, cfg)
		if err != nil {
			return nil, err
		}

		if len(createdPending) != 0 || len(updatedPending) != 0 || len(deletedPending) != 0 || len(createdMatches) != 0 {
			d.logger.Debugw("countdown matchmaker results",
				"gamemode", cfg.Id,
				"pending (exists)", len(updatedPending),
				"pending (created)", len(createdPending),
				"pending (deleted)", len(deletedPending),
				"matches", len(createdMatches),
			)
		}

		matches = createdMatches // assign the created matches to the return value

		// handle the pending match stuff
		if len(updatedPending) > 0 {
			err = d.repo.UpdatePendingMatches(ctx, updatedPending)
			if err != nil {
				return nil, err
			}

			for _, match := range updatedPending {
				if err := d.notifier.PendingMatchUpdated(ctx, match); err != nil {
					d.logger.Errorw("failed to notify pending match updated", "error", err)
				}
			}
		}

		// handle created pending matches (just notify rn)
		if len(createdPending) > 0 {
			err = d.repo.CreatePendingMatches(ctx, createdPending)
			if err != nil {
				return nil, err
			}

			for _, match := range createdPending {

				if err := d.notifier.PendingMatchCreated(ctx, match); err != nil {
					d.logger.Errorw("failed to notify pending match created", "error", err)
				}
			}
		}

		// Delete pending matches from DB and notify with reason match created
		if len(deletedPending) > 0 {
			deletedIds := make([]primitive.ObjectID, 0)
			for _, pending := range deletedPending {
				deletedIds = append(deletedIds, pending.Id)
			}

			err = d.repo.DeletePendingMatches(ctx, deletedIds)
			if err != nil {
				return nil, err
			}

			for _, match := range deletedPending {
				if err := d.notifier.PendingMatchDeleted(ctx, match, msg.PendingMatchDeletedMessage_MATCH_CREATED); err != nil {
					d.logger.Errorw("failed to notify pending match deleted", "error", err)
				}
			}
		}

		inPendingMatchUpdates := make(map[primitive.ObjectID]bool)
		for _, ticket := range tickets {
			if ticket.InternalUpdates != nil && ticket.InternalUpdates.InPendingMatchUpdated {
				inPendingMatchUpdates[ticket.Id] = ticket.InPendingMatch
			}
		}
		if len(inPendingMatchUpdates) > 0 {
			_, err = d.repo.MassUpdateTicketInPendingMatch(ctx, inPendingMatchUpdates)
			if err != nil {
				return nil, err
			}
		}

		// (Kafka) send notifications for updated tickets
		for _, ticket := range tickets {
			if ticket.InternalUpdates != nil {
				if err := d.notifier.TicketUpdated(ctx, ticket); err != nil {
					d.logger.Errorw("failed to send ticket updated notification", "error", err)
				}
			}
		}
	case liveconfig.MatchMethodInstant:
		matches, err = matchfunction2.RunInstant(tickets, cfg)
	}
	if err != nil {
		return nil, err
	}

	if len(matches) != 0 {
		d.logger.Debugw("matchmaker finished", "gamemode", cfg.Id, "matches", len(matches))
	}

	if len(matches) == 0 {
		return nil, nil
	}

	if len(cfg.Maps) > 0 {
		err = d.calculateMaps(ctx, cfg, matches)
		if err != nil {
			return nil, err
		}
	}

	// Assign a server for each match
	errorMap := d.allocateServers(ctx, cfg, matches)
	if len(errorMap) > 0 {
		d.logger.Error("failed to allocate servers", zap.String("gamemode", cfg.Id), loggableErrorMap(errorMap))
	}
	// TODO let's use the errorMap to do some retry logic and not delete the Tickets and QueuedPlayers

	// Notify of match creation
	for _, match := range matches {
		d.logger.Infow("match created", "match", match.Id, "assignment", match.Assignment)
		err = d.notifier.MatchCreated(ctx, match)
		if err != nil {
			d.logger.Errorw("error notifying of match creation", "match", match.Id, "error", err)
		}
	}

	// delete all Tickets and QueuedPlayers in Matches (not PendingMatches)
	for _, match := range matches {
		ticketIds := make([]primitive.ObjectID, 0)
		playerIds := make([]uuid.UUID, 0)
		for _, pbTicket := range match.Tickets {
			var ticket *model.Ticket
			for _, t := range tickets {
				if t.Id.Hex() == pbTicket.Id {
					ticket = t
					break
				}
			}

			// (Kafka) Notify of ticket deletion
			// TODO this is actually wrong. This match may have been deleted because there were no longer enough tickets to maintain the PendingMatch
			if err := d.notifier.TicketDeleted(ctx, pbTicket, msg.TicketDeletedMessage_MATCH_CREATED); err != nil {
				d.logger.Errorw("failed to notify ticket deleted", "error", err)
			}

			ticketIds = append(ticketIds, ticket.Id)

			playerIds = append(playerIds, ticket.PlayerIds...)
		}

		// Delete Tickets
		deletedCount, err := d.repo.DeleteAllTicketsById(ctx, ticketIds)
		if err != nil {
			return nil, err
		}

		if int(deletedCount) != len(ticketIds) {
			d.logger.Warnw("deleted tickets count does not match expected count", "deleted", deletedCount, "expected", len(ticketIds))
		}

		// Delete QueuedPlayers
		deletedCount, err = d.repo.DeleteAllQueuedPlayersById(ctx, playerIds)
		if err != nil {
			return nil, err
		}

		if int(deletedCount) != len(playerIds) {
			d.logger.Warnw("deleted players count does not match expected count", "deleted", deletedCount, "expected", len(playerIds))
		}
	}

	return matches, nil
}

// calculate map retrieves the map votes for those present in a Match
// and assigns the MapId field of a msg.Match
func (d *directorImpl) calculateMaps(ctx context.Context, cfg *liveconfig.GameModeConfig, matches []*pb.Match) error {
	for _, match := range matches {
		playerIds := make([]uuid.UUID, 0)

		for _, ticket := range match.Tickets {
			for _, playerId := range ticket.PlayerIds {
				parsedId, err := uuid.Parse(playerId)
				if err != nil {
					// Note: We're not returning the error as map selection isn't critical
					d.logger.Errorw("failed to parse player id", "playerId", playerId)
					continue
				}

				playerIds = append(playerIds, parsedId)
			}
		}

		players, err := d.repo.GetAllQueuedPlayersByIds(ctx, playerIds)
		if err != nil {
			return err
		}

		// map of map id to number of votes
		mapVotes := make(map[string]int)
		for _, player := range players {
			if player.MapId != nil {
				mapVotes[*player.MapId]++
			}
		}

		// find the map with the most votes
		mostVotedMapIds := make([]*string, 0)
		mostVotes := 0

		for mapId, votes := range mapVotes {
			if votes > mostVotes {
				mostVotedMapIds = []*string{&mapId}
				mostVotes = votes
			}
			if votes == mostVotes {
				mostVotedMapIds = append(mostVotedMapIds, &mapId)
			}
		}

		// If no maps are voted, add all to the pool
		if len(mostVotedMapIds) == 0 {
			for _, lMap := range cfg.Maps {
				mostVotedMapIds = append(mostVotedMapIds, &lMap.Id)
			}
		}

		mapVoteLen := len(mostVotedMapIds)
		if mapVoteLen > 1 {
			match.MapId = mostVotedMapIds[rand.Intn(len(mostVotedMapIds))]
		} else if mapVoteLen == 1 {
			match.MapId = mostVotedMapIds[0]
		} else {
			random := rand.Intn(len(cfg.Maps))
			match.MapId = utils.PointerOf(reflect.ValueOf(cfg.Maps).MapKeys()[random].String())
		}
	}

	return nil
}

// allocateServers allocates servers for the given matches
// returns: map of match id to error
// NOTE: this function blocks until all matches have been allocated
// TODO let's make a system to retry failed allocations
func (d *directorImpl) allocateServers(ctx context.Context, config *liveconfig.GameModeConfig, matches []*pb.Match) map[*pb.Match]error {
	allocationMap := make(map[*pb.Match]*allocatorv1.GameServerAllocation)
	for _, match := range matches {
		var selector *allocatorv1.GameServerAllocation
		switch config.MatchmakerInfo.SelectMethod {
		case liveconfig.SelectMethodAvailable:
			selector = selector2.CreateAvailableSelector(config, match)
		case liveconfig.SelectMethodPlayerCount:
			selector = selector2.CreatePlayerBasedSelector(config.FleetName, match, protoutils.GetMatchPlayerCount(match))
		}

		allocationMap[match] = selector
	}

	allocationErrs := gsallocation.AllocateServers(ctx, d.allocationClient, allocationMap)
	return allocationErrs
}

func (d *directorImpl) onGameModeConfigUpdate(update liveconfig.ConfigUpdate[liveconfig.GameModeConfig]) {
	// TODO
}

func loggableErrorMap(errorMap map[*pb.Match]error) []zap.Field {
	logs := make([]zap.Field, len(errorMap))

	i := 0
	for match, err := range errorMap {
		logs[i] = zap.NamedError(match.Id, err)
	}

	return logs
}
