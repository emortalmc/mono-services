package protoutils

import pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"

func GetMatchPlayerCount(match *pb.Match) int64 {
	count := 0

	for _, ticket := range match.Tickets {
		count += len(ticket.PlayerIds)
	}

	return int64(count)
}
