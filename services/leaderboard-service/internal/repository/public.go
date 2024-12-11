package repository

type Repository interface {
	GetLeaderboardEntries(id string, period)
}
