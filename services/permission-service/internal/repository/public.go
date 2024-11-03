package repository

import (
	"context"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/model"
	"github.com/google/uuid"
)

type Repository interface {
	GetAllRoles(ctx context.Context) ([]*model.Role, error)
	GetRole(ctx context.Context, roleId string) (*model.Role, error)
	DoesRoleExist(ctx context.Context, roleId string) (bool, error)
	CreateRole(ctx context.Context, role *model.Role) error
	UpdateRole(ctx context.Context, newRole *model.Role) error

	GetPlayerRoleIds(ctx context.Context, playerId uuid.UUID) ([]string, error)
	AddRoleToPlayer(ctx context.Context, playerId uuid.UUID, roleId string) error
	RemoveRoleFromPlayer(ctx context.Context, playerId uuid.UUID, roleId string) error
}
