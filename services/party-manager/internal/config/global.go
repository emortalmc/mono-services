package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Kafka   KafkaConfig
	MongoDB MongoDBConfig

	Port        uint16
	Development bool
}

type MongoDBConfig struct {
	URI string
}

type KafkaConfig struct {
	Host string
	Port int
}

func LoadGlobalConfig() (*Config, error) {
	cfg := &Config{}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.UnmarshalExact(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
