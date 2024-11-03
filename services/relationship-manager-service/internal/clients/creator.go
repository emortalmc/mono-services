package clients

import (
	"fmt"
	"github.com/emortalmc/mono-services/services/relationship-manager-service/internal/config"
	"github.com/emortalmc/proto-specs/gen/go/grpc/mcplayer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewPlayerTrackerClient(cfg *config.PlayerTrackerServiceConfig) (mcplayer.PlayerTrackerClient, error) {
	lis, err := createConnection(cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}

	return mcplayer.NewPlayerTrackerClient(lis), nil
}

func createConnection(addr string, port uint16) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", addr, port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
