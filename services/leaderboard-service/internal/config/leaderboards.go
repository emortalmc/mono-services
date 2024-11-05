package config

type order uint8

const (
	Ascending order = iota
	Descending
)

type period uint8

const (
	AllTime period = iota
	Daily
	Weekly
	Monthly
)

type LeaderboardConfig struct {
	ID string

	Order order
	// EvaluatedPeriods used for certain features like notifying users of position changes in weekly, all time, etc.. leaderboards
	EvaluatedPeriods []period
	Personal         PersonalLeaderboardConfig
}

type PersonalLeaderboardConfig struct {
	enabled    bool
	storeLimit int
}

var Leaderboards = map[string]LeaderboardConfig{
	"minesweeperTime": {
		ID: "minesweeperTime",

		Order:            Ascending,
		EvaluatedPeriods: []period{AllTime, Weekly, Monthly},
		Personal: PersonalLeaderboardConfig{
			enabled:    true,
			storeLimit: 10,
		},
	},
	"marathonScore": {
		ID: "marathonScore",

		Order:            Descending,
		EvaluatedPeriods: []period{AllTime, Weekly, Monthly},
		Personal: PersonalLeaderboardConfig{
			enabled:    true,
			storeLimit: 10,
		},
	},
}
