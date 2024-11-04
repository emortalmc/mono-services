package parsers

import (
	"github.com/emortalmc/mono-services/services/game-tracker/internal/repository/model"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/gametracker"
	"google.golang.org/protobuf/proto"
)

func parseGameTeamData(m proto.Message, g *model.Game) error {
	cast := m.(*pbmodel.CommonGameTeamData)

	teams := make([]*model.Team, len(cast.Teams))
	for i, t := range cast.Teams {
		parsed, err := model.TeamFromProto(t)
		if err != nil {
			return err
		}

		teams[i] = parsed
	}

	g.TeamData = &teams

	return nil
}

func parseGameFinishWinnerData(m proto.Message, g *model.HistoricGame) error {
	cast := m.(*pbmodel.CommonGameFinishWinnerData)

	d, err := model.HistoricWinnerDataFromProto(cast)
	if err != nil {
		return err
	}

	g.WinnerData = d

	return nil
}
