package clients

import (
	"fmt"
	"github.com/emortalmc/mono-services/services/message-handler/internal/config"
	"github.com/emortalmc/proto-specs/gen/go/grpc/badge"
	"github.com/emortalmc/proto-specs/gen/go/grpc/mcplayer"
	"github.com/emortalmc/proto-specs/gen/go/grpc/permission"
	"github.com/emortalmc/proto-specs/gen/go/grpc/relationship"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewRelationshipClient(cfg *config.RelationshipServiceConfig) (relationship.RelationshipClient, error) {
	lis, err := createConnection(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	return relationship.NewRelationshipClient(lis), nil
}

func NewPlayerTrackerClient(cfg *config.PlayerTrackerServiceConfig) (mcplayer.PlayerTrackerClient, error) {
	lis, err := createConnection(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	return mcplayer.NewPlayerTrackerClient(lis), nil
}

func NewPermissionClient(cfg *config.PermissionServiceConfig) (permission.PermissionServiceClient, error) {
	lis, err := createConnection(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	return permission.NewPermissionServiceClient(lis), nil
}

func NewBadgeClient(cfg *config.BadgeServiceConfig) (badge.BadgeManagerClient, error) {
	lis, err := createConnection(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	return badge.NewBadgeManagerClient(lis), nil
}

func createConnection(addr string, port uint16) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", addr, port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
