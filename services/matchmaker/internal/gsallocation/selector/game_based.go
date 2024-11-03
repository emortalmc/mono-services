package selector

import (
	"agones.dev/agones/pkg/apis"
	v2 "agones.dev/agones/pkg/apis/agones/v1"
	allocatorv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"github.com/emortalmc/live-config-parser/golang/pkg/liveconfig"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/utils"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateAvailableSelector(cfg *liveconfig.GameModeConfig, match *pb.Match) *allocatorv1.GameServerAllocation {
	fleetName := cfg.FleetName

	return &allocatorv1.GameServerAllocation{
		Spec: allocatorv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Priorities: []v2.Priority{
				{
					Type:  "Counter",
					Key:   "games",
					Order: "Ascending",
				},
			},
			Selectors: []allocatorv1.GameServerSelector{
				{
					LabelSelector: v1.LabelSelector{
						MatchLabels: map[string]string{
							"agones.dev/fleet": fleetName,
						},
						MatchExpressions: []v1.LabelSelectorRequirement{notOutdatedExpression},
					},
					Counters: map[string]allocatorv1.CounterSelector{
						"games": {
							MinAvailable: 1,
						},
					},
					GameServerState: &AllocatedState,
				},
				createReadySelector(fleetName),
			},
			Counters: map[string]allocatorv1.CounterAction{
				"games": {
					Action: utils.PointerOf("Increment"),
					Amount: utils.PointerOf(int64(1)),
				},
			},
			Lists: map[string]allocatorv1.ListAction{
				"games": {
					AddValues: []string{match.Id},
				},
			},
		},
	}
}
