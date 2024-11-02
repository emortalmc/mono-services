package repository

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/mc-player-service/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (m *mongoRepository) GetPlayerServers(ctx context.Context, playerID []uuid.UUID) (map[uuid.UUID]model.CurrentServer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type result struct {
		ID            uuid.UUID           `bson:"_id"`
		CurrentServer model.CurrentServer `bson:"currentServer,omitempty"`
	}
	var mongoResults []result

	cursor, err := m.playerCollection.Find(ctx, bson.M{"_id": bson.M{"$in": playerID}},
		options.Find().SetProjection(bson.M{"currentServer": 1}))
	if err != nil {
		return nil, err
	}

	if err := cursor.All(ctx, &mongoResults); err != nil {
		return nil, err
	}

	resultMap := make(map[uuid.UUID]model.CurrentServer)
	for _, r := range mongoResults {
		resultMap[r.ID] = r.CurrentServer
	}

	return resultMap, nil
}

func (m *mongoRepository) GetServerPlayers(ctx context.Context, serverId string) ([]model.OnlinePlayer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := m.playerCollection.Find(ctx, bson.M{"currentServer.serverId": serverId},
		options.Find().SetProjection(model.OnlinePlayerProjection))
	if err != nil {
		return nil, err
	}

	var mongoResults []model.OnlinePlayer
	err = cursor.All(ctx, &mongoResults)
	if err != nil {
		return nil, err
	}

	return mongoResults, nil
}

// GetPlayerCount TODO implement caching
func (m *mongoRepository) GetPlayerCount(ctx context.Context, serverId *string, fleetNames []string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := createAdapterQuery(serverId, fleetNames)
	return m.playerCollection.CountDocuments(ctx, query)
}

func (m *mongoRepository) GetOnlinePlayers(ctx context.Context, serverId *string, fleetNames []string) ([]model.OnlinePlayer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := createAdapterQuery(serverId, fleetNames)
	cursor, err := m.playerCollection.Find(ctx, query, options.Find().SetProjection(model.OnlinePlayerProjection))
	if err != nil {
		return nil, fmt.Errorf("failed to find online players: %w", err)
	}

	var mongoResults []model.OnlinePlayer
	if err := cursor.All(ctx, &mongoResults); err != nil {
		return nil, fmt.Errorf("failed to decode online players: %w", err)
	}

	return mongoResults, nil
}

func createAdapterQuery(serverId *string, fleetNames []string) bson.M {
	query := bson.M{}
	if serverId != nil {
		query["currentServer.serverId"] = serverId
	} else if fleetNames != nil {
		query["currentServer.fleetName"] = bson.M{"$in": fleetNames}
	} else {
		// global
		query["currentServer"] = bson.M{"$exists": true}
	}
	return query
}

func (m *mongoRepository) GetFleetPlayerCounts(ctx context.Context, fleetNames []string) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.playerCollection.Aggregate(ctx, []bson.M{
		{"$match": bson.M{"currentServer.fleetName": bson.M{"$in": fleetNames}}},
		{"$group": bson.M{"_id": "$currentServer.fleetName", "count": bson.M{"$sum": 1}}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate player count: %w", err)
	}

	resultMap := make(map[string]int64)
	for res.Next(ctx) {
		var r struct {
			FleetName string `bson:"_id"`
			Count     int64  `bson:"count"`
		}
		err := res.Decode(&r)
		if err != nil {
			return nil, err
		}
		resultMap[r.FleetName] = r.Count
	}

	for _, fleetName := range fleetNames {
		if _, ok := resultMap[fleetName]; !ok {
			resultMap[fleetName] = 0
		}
	}

	return resultMap, nil
}

func (m *mongoRepository) GetAllFleetPlayerCounts(ctx context.Context) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.playerCollection.Aggregate(ctx, []bson.M{
		{"$match": bson.M{"currentServer": bson.M{"$exists": true}}},
		{"$group": bson.M{"_id": "$currentServer.fleetName", "count": bson.M{"$sum": 1}}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate player count: %w", err)
	}

	resultMap := make(map[string]int64)
	for res.Next(ctx) {
		var r struct {
			FleetName string `bson:"_id"`
			Count     int64  `bson:"count"`
		}
		err := res.Decode(&r)
		if err != nil {
			return nil, err
		}
		resultMap[r.FleetName] = r.Count
	}

	return resultMap, nil
}

func (m *mongoRepository) SetPlayerServerAndFleet(ctx context.Context, playerId uuid.UUID, serverId string, fleet string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.playerCollection.UpdateOne(ctx, bson.M{"_id": playerId}, bson.M{
		"$set": bson.M{
			"currentServer.serverId":  serverId,
			"currentServer.fleetName": fleet,
		},
	})
	return err
}
