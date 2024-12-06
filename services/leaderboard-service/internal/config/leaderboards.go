package config

import lbproto "github.com/emortalmc/proto-specs/gen/go/model/leaderboard"

type LeaderboardConfig struct {
	ID string

	Order lbproto.SortOrder
	// EvaluatedPeriods used for certain features like notifying users of position changes in weekly, all time, etc.. leaderboards
	EvaluatedPeriods []lbproto.Period
	Personal         PersonalLeaderboardConfig
}

type PersonalLeaderboardConfig struct {
	enabled    bool
	storeLimit int
}

var Leaderboards = map[string]LeaderboardConfig{
	"minesweeperTime": {
		ID: "minesweeperTime",

		Order:            lbproto.SortOrder_ASC,
		EvaluatedPeriods: []lbproto.Period{lbproto.Period_ALL_TIME, lbproto.Period_WEEK, lbproto.Period_MONTH},
		Personal: PersonalLeaderboardConfig{
			enabled:    true,
			storeLimit: 10,
		},
	},
	"marathonScore": {
		ID: "marathonScore",

		Order:            lbproto.SortOrder_DESC,
		EvaluatedPeriods: []lbproto.Period{lbproto.Period_ALL_TIME, lbproto.Period_WEEK, lbproto.Period_MONTH},
		Personal: PersonalLeaderboardConfig{
			enabled:    true,
			storeLimit: 10,
		},
	},
}
