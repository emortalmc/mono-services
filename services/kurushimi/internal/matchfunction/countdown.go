package matchfunction

import (
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository/model"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"sort"
	"time"
)

// CountdownRemoveInvalidPendingMatches removes pending matches from the pendingMatches array that don't have enough players.
// returns the pending matches that have been removed.
// This method also updates the PlayerCount field of the pending matches that are still valid.
func CountdownRemoveInvalidPendingMatches(pendingMatches map[primitive.ObjectID]*model.PendingMatch,
	tickets map[primitive.ObjectID]*model.Ticket, config *liveconfig.GameModeConfig) (deletedPendingMatches []*model.PendingMatch) {

	for id, pendingMatch := range pendingMatches {
		playerCount := getPendingMatchPlayerCount(tickets, pendingMatch)
		if playerCount < config.MinPlayers {
			deletedPendingMatches = append(deletedPendingMatches, pendingMatch)

			// remove pending match from pendingMatches
			delete(pendingMatches, id)

			// put tickets back into the pool
			for _, ticketId := range pendingMatch.TicketIds {
				if ticket, ok := tickets[ticketId]; ok {
					ticket.UpdateInPendingMach(false) // TODO not working
				}
			}
		} else {
			pendingMatch.PlayerCount = playerCount
		}
	}

	return
}

// RunCountdown
// returns:
// - updatedPendingMatches: pending matches that have been created or updated. These should be saved with upsert.
// - deletedPendingMatches: pending matches that have been deleted (either because < min players or converted to a Match)
// - createdMatches: matches that have been created.
// TODO what about the tickets no longer used?
func RunCountdown(logger *zap.SugaredLogger, ticketMap map[primitive.ObjectID]*model.Ticket,
	pendingMatches map[primitive.ObjectID]*model.PendingMatch, config *liveconfig.GameModeConfig) (
	createdPendingMatches []*model.PendingMatch, updatedPendingMatches []*model.PendingMatch,
	deletedPendingMatches []*model.PendingMatch, createdMatches []*pb.Match, err error) {

	updatedPendingMatches = make([]*model.PendingMatch, 0)
	deletedPendingMatches = make([]*model.PendingMatch, 0)
	createdMatches = make([]*pb.Match, 0)

	remainingTicketMap := make(map[primitive.ObjectID]*model.Ticket)
	for ticketId, ticket := range ticketMap {
		if !ticket.InPendingMatch {
			remainingTicketMap[ticketId] = ticket
		}
	}

	// for remaining tickets, try to fill pending matches
	filledPendingMatches := fillPendingMatches(remainingTicketMap, pendingMatches, config)
	updatedPendingMatches = append(updatedPendingMatches, filledPendingMatches...)

	enoughPlayers := false
	playerCount := 0
	for _, ticket := range remainingTicketMap {
		playerCount += len(ticket.PlayerIds)
		if playerCount >= config.MinPlayers {
			enoughPlayers = true
			break
		}
	}

	// create new pending matches if there are still tickets left
	if enoughPlayers {
		logger.Debugw("RunCountdown creating new pending matches", "remainingTicketMap", len(remainingTicketMap))
		remainingTickets := make([]*model.Ticket, len(remainingTicketMap))
		i := 0
		for _, ticket := range remainingTicketMap {
			remainingTickets[i] = ticket
			i++
		}

		newPendingMatches := createPendingMatches(remainingTickets, config)
		createdPendingMatches = append(createdPendingMatches, newPendingMatches...)

		logger.Debugw("RunCountdown created new pending matches", "newPendingMatches", len(newPendingMatches))
	}

	// Go through PendingMatches and create Matches if TeleportTime has passed
	for id, pendingMatch := range pendingMatches {
		if pendingMatch.TeleportTime.Before(time.Now()) {
			match := finalisePendingMatch(ticketMap, config, pendingMatch)

			createdMatches = append(createdMatches, match)

			// remove from pendingMatches
			delete(pendingMatches, id)
			deletedPendingMatches = append(deletedPendingMatches, pendingMatch)

			// TODO notify
		}
	}

	return
}

func finalisePendingMatch(ticketMap map[primitive.ObjectID]*model.Ticket, config *liveconfig.GameModeConfig, pendingMatch *model.PendingMatch) *pb.Match {
	pbTickets := make([]*pb.Ticket, len(pendingMatch.TicketIds))
	for j, ticketId := range pendingMatch.TicketIds {
		ticket := ticketMap[ticketId]
		pbTickets[j] = ticket.ToProto()
	}

	// create match
	match := &pb.Match{
		Id:         primitive.NewObjectID().Hex(),
		GameModeId: config.Id,
		Tickets:    pbTickets,
		MapId:      nil, // Done by the director
		Assignment: nil, // Done by the director
	}

	return match
}

// fillPendingMatches
// NOTE: The tickets map passed in should be mutable and tolerate tickets being removed when they are assigned.
func fillPendingMatches(tickets map[primitive.ObjectID]*model.Ticket, pendingMatches map[primitive.ObjectID]*model.PendingMatch, config *liveconfig.GameModeConfig) []*model.PendingMatch {
	updatedPendingMatches := make([]*model.PendingMatch, 0)

	for _, pendingMatch := range pendingMatches {
		remainingSpace := config.MaxPlayers - getPendingMatchPlayerCount(tickets, pendingMatch)
		updated := false

		for ticketId, ticket := range tickets {
			if len(ticket.PlayerIds) <= remainingSpace {
				// add ticket to pending match
				pendingMatch.TicketIds = append(pendingMatch.TicketIds, ticketId)
				ticket.UpdateInPendingMach(true)

				// remove ticket from the tickets map
				delete(tickets, ticketId)

				remainingSpace -= len(ticket.PlayerIds)
				updated = true
			}
		}

		if updated {
			pendingMatch.PlayerCount = getPendingMatchPlayerCount(tickets, pendingMatch)
			updatedPendingMatches = append(updatedPendingMatches, pendingMatch)
		}
	}

	return updatedPendingMatches
}

func createPendingMatches(tickets []*model.Ticket, config *liveconfig.GameModeConfig) []*model.PendingMatch {
	createdPendingMatches := make([]*model.PendingMatch, 0)

	remainingPlayerCount := 0
	for _, ticket := range tickets {
		remainingPlayerCount += len(ticket.PlayerIds)
	}

	sort.Slice(tickets, func(i, j int) bool {
		return len(tickets[i].PlayerIds) > len(tickets[j].PlayerIds)
	})

	// Whilst sufficient players, make new PendingMatches
	// Each iteration will create a new PendingMatch and fill it with tickets
	// TODO still needs some rework to follow the above desc
	for remainingPlayerCount >= config.MinPlayers {
		teleportTime := time.Now().Add(10 * time.Second) // TODO make this predictive of when the batch is going to run to reduce perceived lag
		currentPendingMatch := &model.PendingMatch{
			Id:           primitive.NewObjectID(),
			GameModeId:   config.Id,
			TicketIds:    make([]primitive.ObjectID, 0),
			TeleportTime: &teleportTime,
		}

		pendingMatchSpace := config.MaxPlayers

		// fill the current pending match with tickets
		for _, ticket := range tickets {
			if len(ticket.PlayerIds) <= pendingMatchSpace {
				currentPendingMatch.TicketIds = append(currentPendingMatch.TicketIds, ticket.Id)
				currentPendingMatch.PlayerCount += len(ticket.PlayerIds)
				ticket.UpdateInPendingMach(true)

				remainingPlayerCount -= len(ticket.PlayerIds)
				pendingMatchSpace -= len(ticket.PlayerIds)
			}

			if pendingMatchSpace == 0 {
				break
			}
		}

		// Match is either full or we have no more tickets that fit into it
		createdPendingMatches = append(createdPendingMatches, currentPendingMatch)
	}

	return createdPendingMatches
}

func getPendingMatchPlayerCount(tickets map[primitive.ObjectID]*model.Ticket, pendingMatch *model.PendingMatch) int {
	count := 0
	for _, ticketId := range pendingMatch.TicketIds {
		if ticket, ok := tickets[ticketId]; ok {
			count += len(ticket.PlayerIds)
		}
	}
	return count
}
