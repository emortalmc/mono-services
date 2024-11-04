package config

import (
	"github.com/emortalmc/mono-services/services/game-tracker/internal/utils/runtime"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
)

const (
	kafkaHostFlag   = "kafka-host"
	kafkaPortFlag   = "kafka-port"
	mongoDBURIFlag  = "mongodb-uri"
	developmentFlag = "development"
	grpcPortFlag    = "port"
)

func LoadGlobalConfig() Config {
	viper.SetDefault(kafkaHostFlag, "localhost")
	viper.SetDefault(kafkaPortFlag, 9092)
	viper.SetDefault(mongoDBURIFlag, "mongodb://localhost:27017")
	viper.SetDefault(developmentFlag, true)
	viper.SetDefault(grpcPortFlag, 10010)

	pflag.String(kafkaHostFlag, viper.GetString(kafkaHostFlag), "Kafka host")
	pflag.Int32(kafkaPortFlag, viper.GetInt32(kafkaPortFlag), "Kafka port")
	pflag.String(mongoDBURIFlag, viper.GetString(mongoDBURIFlag), "MongoDB URI")
	pflag.Bool(developmentFlag, viper.GetBool(developmentFlag), "Development mode")
	pflag.Int32(grpcPortFlag, viper.GetInt32(grpcPortFlag), "gRPC port")
	pflag.Parse()

	// Bind the viper flags to environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(kafkaHostFlag))
	runtime.Must(viper.BindEnv(kafkaPortFlag))
	runtime.Must(viper.BindEnv(mongoDBURIFlag))
	runtime.Must(viper.BindEnv(developmentFlag))
	runtime.Must(viper.BindEnv(grpcPortFlag))

	return Config{
		Kafka: KafkaConfig{
			Host: viper.GetString(kafkaHostFlag),
			Port: int(viper.GetInt32(kafkaPortFlag)),
		},
		MongoDB: MongoDBConfig{
			URI: viper.GetString(mongoDBURIFlag),
		},
		Development: viper.GetBool(developmentFlag),
		GRPCPort:    int(viper.GetInt32(grpcPortFlag)),
	}
}

type Config struct {
	Kafka   KafkaConfig
	MongoDB MongoDBConfig

	Development bool

	GRPCPort int
}

type KafkaConfig struct {
	Host string
	Port int
}

type MongoDBConfig struct {
	URI string
}
