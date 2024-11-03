package selector

import (
	"agones.dev/agones/pkg/apis"
	allocatorv1 "agones.dev/agones/pkg/apis/allocation/v1"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePlayerBasedSelector selects a GameServer where there is no 'match'.
// This could be a singleplayer game (e.g. marathon) or a stateless drop-in drop-out game (e.g. the lobby)
func CreatePlayerBasedSelector(fleetName string, match *pb.Match, playerCount int64) *allocatorv1.GameServerAllocation {
	return &allocatorv1.GameServerAllocation{
		Spec: allocatorv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocatorv1.GameServerSelector{
				{
					LabelSelector: v1.LabelSelector{
						MatchLabels: map[string]string{
							"agones.dev/fleet": fleetName,
						},
						MatchExpressions: []v1.LabelSelectorRequirement{notOutdatedExpression},
					},
					Counters: map[string]allocatorv1.CounterSelector{
						"players": {
							MinAvailable: playerCount,
						},
					},
					GameServerState: &AllocatedState,
				},
				createReadySelector(fleetName),
			},
		},
	}
}
