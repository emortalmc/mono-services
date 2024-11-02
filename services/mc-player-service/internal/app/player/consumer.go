package player

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/utils"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func (s *serviceImpl) HandlePlayerConnect(ctx context.Context, time time.Time, playerID uuid.UUID, playerUsername string,
	proxyID string, playerSkin model.PlayerSkin, player model.Player) {

	session := model.LoginSession{
		ID:       primitive.NewObjectIDFromTimestamp(time),
		PlayerID: playerID,
	}

	if err := s.repo.CreateLoginSession(ctx, session); err != nil {
		s.log.Errorw("error creating login session", "error", err)
	}

	server := &model.CurrentServer{ProxyID: proxyID}
	updatedUsername := false

	if player.IsEmpty() {
		player = model.Player{
			ID:              playerID,
			CurrentUsername: playerUsername,
			CurrentSkin:     playerSkin,
			FirstLogin:      time,
			LastOnline:      time,
			TotalPlaytime:   0,
			CurrentServer:   server,
		}
		updatedUsername = true
	} else {
		if player.CurrentUsername != playerUsername {
			player.CurrentUsername = playerUsername
			updatedUsername = true
		}
		player.CurrentSkin = playerSkin
		player.CurrentServer = server
	}

	if err := s.repo.SavePlayer(ctx, player, true); err != nil {
		s.log.Errorw("error saving player", "error", err)
		return
	}
	count, err := s.repo.GetPlayerCount(ctx, nil, nil)
	if err != nil {
		s.log.Errorw("error getting player count", "error", err)
	}

	s.webhook.SendPlayerJoinWebhook(playerUsername, player.ID.String(), count)

	if updatedUsername {
		dbUsername := model.PlayerUsername{
			ID:       primitive.NewObjectIDFromTimestamp(time),
			PlayerID: player.ID,
			Username: playerUsername,
		}

		err = s.repo.CreatePlayerUsername(ctx, dbUsername)
		if err != nil {
			s.log.Errorw("error creating player username", "error", err)
		}
	}
}

func (s *serviceImpl) HandlePlayerDisconnect(ctx context.Context, time time.Time, playerID uuid.UUID, playerUsername string) {
	session, err := s.repo.GetCurrentLoginSession(ctx, playerID)
	if err != nil {
		s.log.Errorw("error getting current login session", "playerId", playerID, "error", err)
		return
	}

	err = s.repo.SetLoginSessionLogoutTime(ctx, playerID, time)
	if err != nil {
		s.log.Errorw("error setting logout time", "error", err)
		return
	}

	if err := s.repo.PlayerLogout(ctx, playerID, time, session.GetDuration()); err != nil {
		s.log.Errorw("error logging out player", "error", err)
		return
	}

	count, err := s.repo.GetPlayerCount(ctx, nil, nil)
	if err != nil {
		s.log.Errorw("error getting player count", "error", err)
	}

	s.webhook.SendPlayerLeaveWebhook(playerUsername, playerID.String(), count)
}

func (s *serviceImpl) HandlePlayerServerSwitch(ctx context.Context, pID uuid.UUID, newServerID string) {
	if err := s.repo.SetPlayerServerAndFleet(ctx, pID, newServerID, utils.ParseFleetFromPodName(newServerID)); err != nil {
		s.log.Errorw("error setting player server", "error", err)
		return
	}
}
