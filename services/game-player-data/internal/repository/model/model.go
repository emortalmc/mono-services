package model

import (
	"github.com/emortalmc/mono-services/services/game-player-data/internal/utils"
	"github.com/emortalmc/proto-specs/gen/go/model/gameplayerdata"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"
)

type GameData interface {
	ToAnyProto() (*anypb.Any, error)
	PlayerID() uuid.UUID
}

type BaseGameData struct {
	PlayerId uuid.UUID `bson:"_id"`
}

func (d *BaseGameData) PlayerID() uuid.UUID {
	return d.PlayerId
}

type BlockSumoData struct {
	BaseGameData `bson:",inline"`

	BlockSlot  uint32 `bson:"blockSlot"`
	ShearsSlot uint32 `bson:"shearsSlot"`
}

func (d *BlockSumoData) ToAnyProto() (*anypb.Any, error) {
	return anypb.New(&gameplayerdata.V1BlockSumoPlayerData{
		BlockSlot:  d.BlockSlot,
		ShearsSlot: d.ShearsSlot,
	})
}

type MarathonData struct {
	BaseGameData `bson:",inline"`

	Time         string
	BlockPalette string
	Animation    string `bson:",omitempty"`
}

func (d *MarathonData) ToAnyProto() (*anypb.Any, error) {
	pb := &gameplayerdata.V1MarathonData{
		Time:         d.Time,
		BlockPalette: d.BlockPalette,
	}

	if d.Animation != "" {
		pb.Animation = utils.PointerOf(d.Animation)
	}

	return anypb.New(pb)
}

// MinesweeperData TODO
type MinesweeperData struct {
	BaseGameData `bson:",inline"`
}

// TODO
func (d *MinesweeperData) ToProto() *gameplayerdata.V1MinesweeperPlayerData {
	return &gameplayerdata.V1MinesweeperPlayerData{}
}

// TowerDefenceData TODO
type TowerDefenceData struct {
	PlayerId uuid.UUID `bson:"_id"`
}

// TODO
func (d *TowerDefenceData) ToProto() *gameplayerdata.V1TowerDefencePlayerData {
	return &gameplayerdata.V1TowerDefencePlayerData{}
}
