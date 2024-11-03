package model

import (
	"github.com/emortalmc/mono-services/services/permission-service/internal/utils"
	protoModel "github.com/emortalmc/proto-specs/gen/go/model/permission"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"testing"
)

type roleTest struct {
	input    *Role
	expected *protoModel.Role
}

var roleTests = []roleTest{
	{
		input: &Role{
			Id:          "validTestOne",
			Priority:    1,
			DisplayName: utils.PointerOf("testName"),
			Permissions: []PermissionNode{
				{
					Node:  "testNode",
					State: protoModel.PermissionNode_ALLOW,
				},
			},
		},
		expected: &protoModel.Role{
			Id:          "validTestOne",
			Priority:    1,
			DisplayName: utils.PointerOf("testName"),
			Permissions: []*protoModel.PermissionNode{
				{
					Node:  "testNode",
					State: protoModel.PermissionNode_ALLOW,
				},
			},
		},
	},
	{
		input: &Role{
			Id:          "validTestTwo",
			Priority:    100,
			DisplayName: nil,
			Permissions: []PermissionNode{
				{
					Node:  "testNode",
					State: protoModel.PermissionNode_DENY,
				},
				{
					Node:  "testNode2",
					State: protoModel.PermissionNode_ALLOW,
				},
			},
		},
		expected: &protoModel.Role{
			Id:          "validTestTwo",
			Priority:    100,
			DisplayName: nil,
			Permissions: []*protoModel.PermissionNode{
				{
					Node:  "testNode",
					State: protoModel.PermissionNode_DENY,
				},
				{
					Node:  "testNode2",
					State: protoModel.PermissionNode_ALLOW,
				},
			},
		},
	},
}

func TestRole_ToProto(t *testing.T) {
	for _, test := range roleTests {
		t.Run(test.input.Id, func(t *testing.T) {
			converted := test.input.ToProto()
			if !proto.Equal(converted, test.expected) {
				t.Errorf("expected %s, got %s", test.expected, converted)
			}
		})
	}
}

type permissionNodeTest struct {
	input    *PermissionNode
	expected *protoModel.PermissionNode
}

var permissionNodeTests = []permissionNodeTest{
	{
		input: &PermissionNode{
			Node:  "testNodeAllow",
			State: protoModel.PermissionNode_ALLOW,
		},
		expected: &protoModel.PermissionNode{
			Node:  "testNodeAllow",
			State: protoModel.PermissionNode_ALLOW,
		},
	},
	{
		input: &PermissionNode{
			Node:  "testNodeDeny",
			State: protoModel.PermissionNode_DENY,
		},
		expected: &protoModel.PermissionNode{
			Node:  "testNodeDeny",
			State: protoModel.PermissionNode_DENY,
		},
	},
}

func TestPermissionNode_ToProto(t *testing.T) {
	for _, test := range permissionNodeTests {
		t.Run(test.input.Node, func(t *testing.T) {
			converted := test.input.ToProto()
			assert.True(t, proto.Equal(converted, test.expected), "expected %s, got %s", test.expected, converted)
		})
	}
}
