package matchfunction

import (
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository/model"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RunInstant(tickets []*model.Ticket, config *liveconfig.GameModeConfig) (createdMatches []*pb.Match, err error) {
	createdMatches = make([]*pb.Match, 0)

	currentPlayerCount := 0
	currentMatch := &pb.Match{
		Id:         primitive.NewObjectID().Hex(),
		GameModeId: config.Id,
		Tickets:    make([]*pb.Ticket, 0),
		MapId:      nil, // Done by the director
		Assignment: nil, // Done by the director
	}

	for _, ticket := range tickets {
		currentPlayerCount += len(ticket.PlayerIds)

		// The match becomes too full with this ticket, finalize this match
		if currentPlayerCount > config.MaxPlayers {
			createdMatches = append(createdMatches, currentMatch)

			// Note: we set the player count to the current ticket's player count as we continue creating matches here
			currentPlayerCount = len(ticket.PlayerIds)
			currentMatch = &pb.Match{
				Id:         primitive.NewObjectID().Hex(),
				GameModeId: config.Id,
				Tickets:    make([]*pb.Ticket, 0),
				MapId:      nil, // Done by the director
				Assignment: nil, // Done by the director
			}
		}

		currentMatch.Tickets = append(currentMatch.Tickets, ticket.ToProto())
	}

	if currentPlayerCount >= config.MinPlayers {
		createdMatches = append(createdMatches, currentMatch)
	}

	return createdMatches, nil
}
