package model

import (
	protoModel "github.com/emortalmc/proto-specs/gen/go/model/permission"
	"github.com/google/uuid"
)

const DefaultRoleId = "default"

type Role struct {
	Id            string           `bson:"_id" ,json:"id"`
	Priority      uint32           `bson:"priority" ,json:"priority"`
	DisplayName   *string          `bson:"displayName" ,json:"displayName"`
	Permissions   []PermissionNode `bson:"permissions" ,json:"permissions"`
}

func (r *Role) ToProto() *protoModel.Role {
	protoPermissions := make([]*protoModel.PermissionNode, 0)
	for _, p := range r.Permissions {
		protoPermissions = append(protoPermissions, p.ToProto())
	}

	return &protoModel.Role{
		Id:            r.Id,
		Priority:      r.Priority,
		DisplayName:   r.DisplayName,
		Permissions:   protoPermissions,
	}
}

type PermissionNode struct {
	Node  string                                    `bson:"node" ,json:"node"`
	State protoModel.PermissionNode_PermissionState `bson:"permissionState" ,json:"permissionState"`
}

func (p *PermissionNode) ToProto() *protoModel.PermissionNode {
	return &protoModel.PermissionNode{
		Node:  p.Node,
		State: p.State,
	}
}

// Player does not have an equivalent proto
// as roles are retrieved separately from the player
type Player struct {
	Id    uuid.UUID `bson:"_id" ,json:"id"`
	Roles []string  `bson:"roles" ,json:"roles"`
}
