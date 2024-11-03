package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Kafka   *KafkaConfig
	MongoDB *MongoDBConfig

	Development bool

	Port uint16
}

type KafkaConfig struct {
	Host string
	Port int
}

type MongoDBConfig struct {
	URI string
}

func LoadGlobalConfig() (config *Config, err error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	return
}
