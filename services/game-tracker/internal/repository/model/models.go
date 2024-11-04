package model

import (
	"fmt"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository/registrytypes"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/utils"
	"github.com/emortalmc/proto-specs/gen/go/model/gametracker"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type GameStage uint8

const (
	LiveTowerDefenceDataId int32 = 1
	LiveBlockSumoDataId    int32 = 2
)

var dataType = map[int32]interface{}{
	LiveTowerDefenceDataId: &LiveTowerDefenceData{},
	LiveBlockSumoDataId:    &LiveBlockSumoData{},
}

func getDataType(example interface{}) int32 {
	for k, v := range dataType {
		if utils.IsSameType(example, v) {
			return k
		}
	}

	return 0 // 0 is ignored by omitempty so it won't be put in the db
}

type IGame interface {
	GetGame() *Game
}

type Game struct {
	Id         primitive.ObjectID `bson:"_id"`
	GameModeId string             `bson:"gameModeId"`

	ServerId  string         `bson:"serverId"`
	StartTime *time.Time     `bson:"startTime,omitempty"`
	Players   []*BasicPlayer `bson:"players"`

	// The below data is all optional and varies by game mode

	TeamData *[]*Team `bson:"teams,omitempty"`

	// GameData is data specific to the game mode. It is only present if the game sends it
	GameData     interface{} `bson:"gameData,omitempty"`
	GameDataType int32       `bson:"gameDataType,omitempty"`
}

func (g *Game) GetGame() *Game {
	return g
}

func (g *Game) SetGameData(data interface{}) {
	g.GameData = data
	g.GameDataType = getDataType(data)
}

// ParseGameData converts and replaces the game data from bson.D to the correct type
func (g *Game) ParseGameData() error {
	if g.GameData == nil {
		return nil
	}

	if g.GameDataType == 0 {
		return fmt.Errorf("game data type not set but game data is present")
	}

	example, ok := dataType[g.GameDataType]
	if !ok {
		return fmt.Errorf("unknown game data type: %d", g.GameDataType)
	}

	if example == nil {
		return fmt.Errorf("unknown game data type: %d", g.GameDataType)
	}

	// Convert the interface{} to bytes
	bytes, err := bson.Marshal(g.GameData)
	if err != nil {
		return fmt.Errorf("failed to marshal game data: %w", err)
	}

	// Parse the bytes from the interface{} into the correct type
	dec, err := bson.NewDecoder(bsonrw.NewBSONDocumentReader(bytes))
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := dec.SetRegistry(registrytypes.CodecRegistry); err != nil {
		return fmt.Errorf("failed to set registry: %w", err)
	}

	if err := dec.Decode(example); err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	g.GameData = example

	return nil
}

type LiveGame struct {
	// Embed Game for common fields
	*Game `bson:",inline"`

	// GameData provided at game create (allocation messages)
	LastUpdated time.Time `bson:"lastUpdated"`
}

type HistoricGame struct {
	// Embed Game for common fields
	*Game `bson:",inline"`

	EndTime time.Time `bson:"endTime"`

	// The below data is all optional and varies by game mode
	WinnerData *HistoricWinnerData `bson:"winnerData,omitempty"`
}

type HistoricWinnerData struct {
	WinnerIds []uuid.UUID `bson:"winnerIds"`
	LoserIds  []uuid.UUID `bson:"loserIds"`
}

func HistoricWinnerDataFromProto(d *gametracker.CommonGameFinishWinnerData) (*HistoricWinnerData, error) {
	winnerIds, err := ParseUuids(d.WinnerIds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse winners: %w", err)
	}

	loserIds, err := ParseUuids(d.LoserIds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse losers: %w", err)
	}

	return &HistoricWinnerData{
		WinnerIds: winnerIds,
		LoserIds:  loserIds,
	}, nil
}

type BasicPlayer struct {
	Id       uuid.UUID `bson:"id"`
	Username string    `bson:"username"`
}

func BasicPlayerFromProto(p *gametracker.BasicGamePlayer) (*BasicPlayer, error) {
	id, err := uuid.Parse(p.Id)
	if err != nil {
		return nil, err
	}

	return &BasicPlayer{
		Id:       id,
		Username: p.Username,
	}, nil
}

func BasicPlayersFromProto(players []*gametracker.BasicGamePlayer) ([]*BasicPlayer, error) {
	basicPlayers := make([]*BasicPlayer, len(players))
	for i, p := range players {
		basicPlayer, err := BasicPlayerFromProto(p)
		if err != nil {
			return nil, err
		}

		basicPlayers[i] = basicPlayer
	}

	return basicPlayers, nil
}

type Team struct {
	Id           string      `bson:"id"`
	FriendlyName string      `bson:"friendlyName"`
	Color        int32       `bson:"color"`
	PlayerIds    []uuid.UUID `bson:"playerIds"`
}

func TeamFromProto(t *gametracker.Team) (*Team, error) {
	playerIds := make([]uuid.UUID, len(t.PlayerIds))
	for i, id := range t.PlayerIds {
		playerId, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}

		playerIds[i] = playerId
	}

	return &Team{
		Id:           t.Id,
		FriendlyName: t.FriendlyName,
		Color:        t.Color,
		PlayerIds:    playerIds,
	}, nil
}

func ParseUuids(uuidStrs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(uuidStrs))
	for i, id := range uuidStrs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}

		ids[i] = parsed
	}

	return ids, nil
}
