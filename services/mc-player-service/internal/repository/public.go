package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/emortalmc/proto-specs/gen/go/model/common"
	"github.com/google/uuid"
	"time"
)

type Repository interface {
	BadgeReadWriter
	PlayerReadWriter

	Ping(ctx context.Context) error
}

type BadgeReader interface {
	GetActivePlayerBadge(ctx context.Context, playerId uuid.UUID) (*string, error)
	GetBadgePlayer(ctx context.Context, playerId uuid.UUID) (model.BadgePlayer, error)
}

type BadgeWriter interface {
	AddPlayerBadge(ctx context.Context, playerId uuid.UUID, badgeId string) (int64, error)
	SetActivePlayerBadge(ctx context.Context, playerId uuid.UUID, badgeId *string) error
	RemovePlayerBadge(ctx context.Context, playerId uuid.UUID, badgeId string) (int64, error)
	UpdatePlayerBadgesAndActive(ctx context.Context, playerId uuid.UUID, badges []string, activeBadge *string) error
}

type BadgeReadWriter interface {
	BadgeReader
	BadgeWriter
}

type PlayerReader interface {
	GetPlayer(ctx context.Context, id uuid.UUID) (model.Player, error)
	GetPlayers(ctx context.Context, ids []uuid.UUID) ([]model.Player, error)
	GetPlayerByUsername(ctx context.Context, username string, ignoreCase bool) (model.Player, error)
	SearchPlayersByUsername(ctx context.Context, username string, pageable *common.Pageable, filter *UsernameSearchFilter, ignoredPlayerIds []uuid.UUID) ([]model.Player, *common.PageData, error)

	GetCurrentLoginSession(ctx context.Context, playerId uuid.UUID) (model.LoginSession, error)
	GetLoginSessions(ctx context.Context, playerId uuid.UUID, pageable *common.Pageable) ([]model.LoginSession, error)

	GetPlayerServers(ctx context.Context, playerId []uuid.UUID) (map[uuid.UUID]model.CurrentServer, error)
	GetServerPlayers(ctx context.Context, serverId string) ([]model.OnlinePlayer, error)

	// GetPlayerCount returns the number of players on:
	// 1. the given server if present
	// 2. the given fleets if present
	// 3. globally if neither are present
	GetPlayerCount(ctx context.Context, serverID *string, fleetNames []string) (int64, error)

	// GetOnlinePlayers functions the same as GetPlayerCount
	GetOnlinePlayers(ctx context.Context, serverID *string, fleetNames []string) ([]model.OnlinePlayer, error)

	GetFleetPlayerCounts(ctx context.Context, fleetNames []string) (map[string]int64, error)

	GetTotalUniquePlayers(ctx context.Context) (int64, error)
	GetTotalPlaytimeHours(ctx context.Context) (int64, error)
}

type PlayerWriter interface {
	SavePlayer(ctx context.Context, player model.Player, upsert bool) error
	PlayerLogout(ctx context.Context, playerID uuid.UUID, lastOnline time.Time, addedPlaytime time.Duration) error

	CreateLoginSession(ctx context.Context, session model.LoginSession) error
	SetLoginSessionLogoutTime(ctx context.Context, playerID uuid.UUID, logoutTime time.Time) error
	CreatePlayerUsername(ctx context.Context, username model.PlayerUsername) error

	AddExperienceToPlayer(ctx context.Context, playerID uuid.UUID, experience int) (int, error)
	CreateExperienceTransaction(ctx context.Context, transaction model.ExperienceTransaction) error

	SetPlayerServerAndFleet(ctx context.Context, playerId uuid.UUID, serverId string, fleet string) error
}

type PlayerReadWriter interface {
	PlayerReader
	PlayerWriter
}

type UsernameSearchFilter struct {
	OnlineOnly bool
	Friends    bool
}
