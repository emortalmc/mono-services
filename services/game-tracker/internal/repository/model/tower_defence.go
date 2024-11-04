package model

import "github.com/emortalmc/proto-specs/gen/go/model/gametracker"

type LiveTowerDefenceData struct {
	MaxHealth  int32 `bson:"maxHealth"`
	RedHealth  int32 `bson:"redHealth"`
	BlueHealth int32 `bson:"blueHealth"`
}

func (d *LiveTowerDefenceData) Update(data *gametracker.TowerDefenceUpdateData) {
	healthData := data.HealthData

	d.RedHealth = healthData.RedHealth
	d.BlueHealth = healthData.BlueHealth
}

func CreateLiveTowerDefenceDataFromStart(data *gametracker.TowerDefenceStartData) *LiveTowerDefenceData {
	healthData := data.HealthData

	return &LiveTowerDefenceData{
		MaxHealth:  healthData.MaxHealth,
		RedHealth:  healthData.RedHealth,
		BlueHealth: healthData.BlueHealth,
	}
}

type HistoricTowerDefenceData struct {
	MaxHealth  int32 `bson:"maxHealth"`
	RedHealth  int32 `bson:"redHealth"`
	BlueHealth int32 `bson:"blueHealth"`
}

func CreateHistoricTowerDefenceDataFromFinish(data *gametracker.TowerDefenceFinishData) *HistoricTowerDefenceData {
	healthData := data.HealthData

	return &HistoricTowerDefenceData{
		MaxHealth:  healthData.MaxHealth,
		RedHealth:  healthData.RedHealth,
		BlueHealth: healthData.BlueHealth,
	}
}
