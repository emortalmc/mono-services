package model

import (
	"fmt"
	"github.com/emortalmc/proto-specs/gen/go/model/gametracker"
	"github.com/google/uuid"
)

type LiveBlockSumoData struct {
	Scoreboard *BlockSumoScoreboard `bson:"scoreboard"`
}

type HistoricBlockSumoData struct {
	Scoreboard *BlockSumoScoreboard `bson:"scoreboard"`
}

func CreateLiveBlockSumoDataFromUpdate(data *gametracker.BlockSumoUpdateData) (*LiveBlockSumoData, error) {
	scoreboard, err := CreateBlockSumoScoreboard(data.Scoreboard)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scoreboard: %w", err)
	}

	return &LiveBlockSumoData{
		Scoreboard: scoreboard,
	}, nil
}

func (d *LiveBlockSumoData) Update(data *gametracker.BlockSumoUpdateData) error {
	scoreboard, err := CreateBlockSumoScoreboard(data.Scoreboard)
	if err != nil {
		return fmt.Errorf("failed to parse scoreboard: %w", err)
	}

	d.Scoreboard = scoreboard

	return nil
}

func CreateBlockSumoScoreboard(data *gametracker.BlockSumoScoreboard) (*BlockSumoScoreboard, error) {
	entries := make(map[uuid.UUID]*BlockSumoScoreboardEntry)

	for id, e := range data.Entries {
		parsedId, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("failed to parse player id: %w", err)
		}

		entries[parsedId] = CreateBlockSumoScoreboardEntry(e)
	}

	return &BlockSumoScoreboard{Entries: entries}, nil
}

type BlockSumoScoreboard struct {
	Entries map[uuid.UUID]*BlockSumoScoreboardEntry `bson:"entries"`
}

type BlockSumoScoreboardEntry struct {
	RemainingLives int32 `bson:"remainingLives"`
	Kills          int32 `bson:"kills"`
	FinalKills     int32 `bson:"finalKills"`
}

func CreateBlockSumoScoreboardEntry(e *gametracker.BlockSumoScoreboard_Entry) *BlockSumoScoreboardEntry {
	return &BlockSumoScoreboardEntry{
		RemainingLives: e.RemainingLives,
		Kills:          e.Kills,
		FinalKills:     e.FinalKills,
	}
}

func CreateHistoricBlockSumoDataFromFinish(data *gametracker.BlockSumoFinishData) (*HistoricBlockSumoData, error) {
	scoreboard, err := CreateBlockSumoScoreboard(data.Scoreboard)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scoreboard: %w", err)
	}

	return &HistoricBlockSumoData{
		Scoreboard: scoreboard,
	}, nil
}
