package config

type order uint8

const (
	Ascending order = iota
	Descending
)

type LeaderboardConfig struct {
	ID string

	Order    order
	Personal PersonalLeaderboardConfig
}

type PersonalLeaderboardConfig struct {
	enabled    bool
	storeLimit int
}

var Leaderboards = map[string]LeaderboardConfig{
	"minesweeperTime": {
		ID: "minesweeperTime",

		Order: Ascending,
		Personal: PersonalLeaderboardConfig{
			enabled:    true,
			storeLimit: 10,
		},
	},
	"marathonScore": {
		ID: "marathonScore",

		Order: Descending,
		Personal: PersonalLeaderboardConfig{
			enabled:    true,
			storeLimit: 10,
		},
	},
}
