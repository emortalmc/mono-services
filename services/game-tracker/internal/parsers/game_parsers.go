package parsers

import (
	"fmt"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository/model"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/gametracker"
	"google.golang.org/protobuf/proto"
)

func handleTowerDefenceStartData(m proto.Message, g *model.LiveGame) error {
	cast := m.(*pbmodel.TowerDefenceStartData)
	g.SetGameData(model.CreateLiveTowerDefenceDataFromStart(cast))

	return nil
}

func handleTowerDefenceUpdateData(m proto.Message, g *model.LiveGame) error {
	cast := m.(*pbmodel.TowerDefenceUpdateData)

	(g.GameData).(*model.LiveTowerDefenceData).Update(cast)

	return nil
}

func handleTowerDefenceFinishData(m proto.Message, g *model.HistoricGame) error {
	cast := m.(*pbmodel.TowerDefenceFinishData)

	g.GameData = model.CreateHistoricTowerDefenceDataFromFinish(cast)

	return nil
}

// Block Sumo

func handleBlockSumoUpdateData(m proto.Message, g *model.LiveGame) error {
	cast := m.(*pbmodel.BlockSumoUpdateData)

	if g.GameData == nil {
		newData, err := model.CreateLiveBlockSumoDataFromUpdate(cast)

		if err != nil {
			return fmt.Errorf("failed to create new live block sumo data: %w", err)
		}

		g.SetGameData(newData)
		return nil
	}

	if err := g.GameData.(*model.LiveBlockSumoData).Update(cast); err != nil {
		return err
	}

	return nil
}

func handleBlockSumoFinishData(m proto.Message, g *model.HistoricGame) error {
	cast := m.(*pbmodel.BlockSumoFinishData)

	data, err := model.CreateHistoricBlockSumoDataFromFinish(cast)
	if err != nil {
		return fmt.Errorf("failed to create historic block sumo data: %w", err)
	}

	g.GameData = data

	return nil
}
