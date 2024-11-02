package badge

import (
	"context"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/config"
	permmsg "github.com/emortalmc/proto-specs/gen/go/message/permission"
	"github.com/google/uuid"
)

func (s *serviceImpl) HandlePlayerRolesUpdate(ctx context.Context, playerID uuid.UUID, roleID string,
	changeType permmsg.PlayerRolesUpdateMessage_ChangeType) {

	var badge *config.Badge
	for id, b := range s.badgeCfg.Badges {
		if b.AutomaticGrants == nil || b.AutomaticGrants.PermissionRole == nil {
			continue
		}

		if *b.AutomaticGrants.PermissionRole == roleID {
			badge = s.badgeCfg.Badges[id]
			break
		}
	}

	var err error
	switch changeType {
	case permmsg.PlayerRolesUpdateMessage_ADD:
		err = s.AddBadgeToPlayer(ctx, playerID, badge.Id)
	case permmsg.PlayerRolesUpdateMessage_REMOVE:
		err = s.RemoveBadgeFromPlayer(ctx, playerID, badge.Id)
	}
	if err != nil {
		s.log.Errorw("error updating player badge", "error", err)
	}
}
