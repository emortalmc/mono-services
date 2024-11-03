package notifier

import (
	"context"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/model"
	"github.com/emortalmc/proto-specs/gen/go/message/permission"
)

type Notifier interface {
	RoleUpdate(ctx context.Context, role *model.Role, changeType permission.RoleUpdateMessage_ChangeType) error
	PlayerRolesUpdate(ctx context.Context, playerId string, roleId string, changeType permission.PlayerRolesUpdateMessage_ChangeType) error
}
