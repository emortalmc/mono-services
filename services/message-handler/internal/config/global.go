package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Kafka *KafkaConfig

	RelationshipService  *RelationshipServiceConfig
	PlayerTrackerService *PlayerTrackerServiceConfig
	BadgeService         *BadgeServiceConfig
	PermissionService    *PermissionServiceConfig

	Development bool
	Port        uint16
}

type KafkaConfig struct {
	Host string
	Port int
}

type RelationshipServiceConfig struct {
	Host string
	Port uint16
}

type PlayerTrackerServiceConfig struct {
	Host string
	Port uint16
}

type BadgeServiceConfig struct {
	Host string
	Port uint16
}

type PermissionServiceConfig struct {
	Host string
	Port uint16
}

func LoadGlobalConfig() (cfg *Config, err error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return
	}

	return
}
