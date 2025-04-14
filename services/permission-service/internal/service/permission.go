package service

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/permission-service/internal/kafka/notifier"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository"
	"github.com/emortalmc/mono-services/services/permission-service/internal/repository/model"
	"github.com/emortalmc/proto-specs/gen/go/grpc/permission"
	permission2 "github.com/emortalmc/proto-specs/gen/go/message/permission"
	protoModel "github.com/emortalmc/proto-specs/gen/go/model/permission"
	"github.com/google/uuid"
	mongoDb "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sort"
)

type permissionService struct {
	permission.UnimplementedPermissionServiceServer

	logger *zap.SugaredLogger

	repo  repository.Repository
	notif notifier.Notifier
}

func newPermissionService(logger *zap.SugaredLogger, repo repository.Repository, notif notifier.Notifier) permission.PermissionServiceServer {
	return &permissionService{
		logger: logger,

		repo:  repo,
		notif: notif,
	}
}

func (s *permissionService) GetAllRoles(ctx context.Context, _ *permission.GetAllRolesRequest) (*permission.GetAllRolesResponse, error) {
	roles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		return nil, err
	}

	if roles == nil {
		return &permission.GetAllRolesResponse{}, nil
	}

	var protoRoles = make([]*protoModel.Role, len(roles))
	for i, role := range roles {
		protoRoles[i] = role.ToProto()
	}

	return &permission.GetAllRolesResponse{
		Roles: protoRoles,
	}, nil
}

func (s *permissionService) GetPlayerRoles(ctx context.Context, req *permission.GetPlayerRolesRequest) (*permission.PlayerRolesResponse, error) {
	pId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}
	roles, err := s.repo.GetPlayerRoleIds(ctx, pId)
	if err != nil {
		return nil, err
	}

	activeRole, err := s.computeActiveDisplayNameRole(ctx, roles)
	if err != nil {
		return nil, err
	}

	var activeRoleId *string
	if activeRole != nil {
		activeRoleId = &activeRole.Id
	}

	return &permission.PlayerRolesResponse{
		RoleIds:                 roles,
		ActiveDisplayNameRoleId: activeRoleId,
	}, nil
}

func (s *permissionService) CreateRole(ctx context.Context, req *permission.RoleCreateRequest) (*permission.CreateRoleResponse, error) {
	role := &model.Role{
		Id:          req.Id,
		Priority:    req.Priority,
		DisplayName: req.DisplayName,
		Permissions: make([]model.PermissionNode, 0),
	}

	err := s.repo.CreateRole(ctx, role)

	if err != nil {
		if mongoDb.IsDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "role already exists")
		}
		return nil, fmt.Errorf("error creating role: %w", err)
	}

	if err := s.notif.RoleUpdate(ctx, role, permission2.RoleUpdateMessage_CREATE); err != nil {
		s.logger.Errorw("error sending role update notification", "error", err)
	}

	return &permission.CreateRoleResponse{
		Role: role.ToProto(),
	}, err
}

func (s *permissionService) UpdateRole(ctx context.Context, req *permission.RoleUpdateRequest) (*permission.UpdateRoleResponse, error) {
	role, err := s.repo.GetRole(ctx, req.Id)

	if err != nil {
		if err == mongoDb.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "Role not found")
		}
		return nil, fmt.Errorf("error getting role: %w", err)
	}

	if req.Priority != nil {
		role.Priority = *req.Priority
	}
	if req.DisplayName != nil {
		role.DisplayName = req.DisplayName
	}

	for _, perm := range req.UnsetPermissions {
		for i, node := range role.Permissions {
			if node.Node == perm {
				role.Permissions = append(role.Permissions[:i], role.Permissions[i+1:]...)
			}
		}
	}

	// Update the permission state if it already exists, otherwise add it
	for _, perm := range req.SetPermissions {
		existed := false
		for i, node := range role.Permissions {
			if node.Node == perm.Node {
				role.Permissions[i].State = perm.State
				existed = true
				continue
			}
		}
		if !existed {
			role.Permissions = append(role.Permissions, model.PermissionNode{Node: perm.Node, State: perm.State})
		}
	}

	err = s.repo.UpdateRole(ctx, role)

	if err != nil {
		return nil, fmt.Errorf("error updating role: %w", err)
	}

	if err := s.notif.RoleUpdate(ctx, role, permission2.RoleUpdateMessage_MODIFY); err != nil {
		s.logger.Errorw("error sending role update notification", "error", err)
	}

	return &permission.UpdateRoleResponse{
		Role: role.ToProto(),
	}, nil
}

func (s *permissionService) AddRoleToPlayer(ctx context.Context, req *permission.AddRoleToPlayerRequest) (*permission.AddRoleToPlayerResponse, error) {
	pId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}

	ok, err := s.repo.DoesRoleExist(ctx, req.RoleId)
	if err != nil {
		return nil, err
	}

	if !ok {
		st := status.New(codes.NotFound, "role not found")
		st, _ = st.WithDetails(&permission.AddRoleToPlayerError{ErrorType: permission.AddRoleToPlayerError_ROLE_NOT_FOUND})
		return nil, st.Err()
	}

	err = s.repo.AddRoleToPlayer(ctx, pId, req.RoleId)

	// NOTE: err no documents should never be thrown here because if so, we create a new player with role + default role
	if err != nil {
		if err == repository.ErrAlreadyHasRole {
			st := status.New(codes.AlreadyExists, "player already has role")
			st, _ = st.WithDetails(&permission.AddRoleToPlayerError{ErrorType: permission.AddRoleToPlayerError_ALREADY_HAS_ROLE})
			return nil, st.Err()
		}
		return nil, err
	}

	if err := s.notif.PlayerRolesUpdate(ctx, pId.String(), req.RoleId, permission2.PlayerRolesUpdateMessage_ADD); err != nil {
		s.logger.Errorw("error sending player roles update", "error", err)
	}

	return &permission.AddRoleToPlayerResponse{}, nil
}

var (
	removeRoleFromPlayerPlayerNotFound = panicIfErr(status.New(codes.NotFound, "player not found").
		WithDetails(&permission.RemoveRoleFromPlayerError{ErrorType: permission.RemoveRoleFromPlayerError_PLAYER_NOT_FOUND})).Err()

	removeRoleFromPlayerDoesntHaveRole = panicIfErr(status.New(codes.NotFound, "player does not have role").
		WithDetails(&permission.RemoveRoleFromPlayerError{ErrorType: permission.RemoveRoleFromPlayerError_DOES_NOT_HAVE_ROLE})).Err()
)

func (s *permissionService) RemoveRoleFromPlayer(ctx context.Context, req *permission.RemoveRoleFromPlayerRequest) (*permission.RemoveRoleFromPlayerResponse, error) {
	pId, err := uuid.Parse(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid player id %s", req.PlayerId))
	}

	err = s.repo.RemoveRoleFromPlayer(ctx, pId, req.RoleId)
	if err != nil {
		switch err {
		case mongoDb.ErrNoDocuments:
			return nil, removeRoleFromPlayerPlayerNotFound
		case repository.ErrDoesNotHaveRole:
			return nil, removeRoleFromPlayerDoesntHaveRole
		}
		return nil, err
	}

	if err := s.notif.PlayerRolesUpdate(ctx, pId.String(), req.RoleId, permission2.PlayerRolesUpdateMessage_REMOVE); err != nil {
		s.logger.Errorw("error sending player roles update", "error", err)
	}

	return &permission.RemoveRoleFromPlayerResponse{}, nil
}

func (s *permissionService) computeActiveDisplayNameRole(ctx context.Context, roleIds []string) (*model.Role, error) {
	allRoles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting all roles: %w", err)
	}

	playerRoles := make([]*model.Role, 0)
	for _, roleId := range roleIds {
		for _, role := range allRoles {
			if role.Id == roleId {
				playerRoles = append(playerRoles, role)
			}
		}
	}

	// Sort roles by priority
	sort.SliceStable(playerRoles, func(i, j int) bool {
		return playerRoles[i].Priority > playerRoles[j].Priority
	})

	// Get the highest priority role with a display name
	for _, role := range playerRoles {
		if role.DisplayName != nil {
			return role, nil
		}
	}

	return nil, nil
}

func panicIfErr[T any](thing T, err error) T {
	if err != nil {
		panic(err)
	}
	return thing
}
