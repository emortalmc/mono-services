package config

import (
	"github.com/emortalmc/mono-services/services/matchmaker/internal/utils/runtime"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	kafkaHostFlag  = "kafka-host"
	kafkaPortFlag  = "kafka-port"
	mongoDBURIFlag = "mongodb-uri"

	namespaceFlag = "namespace"

	partyServiceHostFlag         = "party-service-host"
	partyServicePortFlag         = "party-service-port"
	partyServiceSettingsHostFlag = "party-settings-service-host"
	partyServiceSettingsPortFlag = "party-settings-service-port"

	lobbyFleetNameFlag = "lobby-fleet-name"
	lobbyMatchRateFlag = "lobby-match-rate"
	lobbyMatchSizeFlag = "lobby-match-size"

	proxyFleetNameFlag = "proxy-fleet-name"
	proxyMatchRateFlag = "proxy-match-rate"
	proxyMatchSizeFlag = "proxy-match-size"

	grpcPortFlag    = "port"
	developmentFlag = "development"
)

type Config struct {
	Kafka   KafkaConfig
	MongoDB MongoDBConfig

	PartyService PartyServiceConfig

	Lobby LobbyConfig
	Proxy ProxyConfig

	Namespace   string
	GrpcPort    int
	Development bool
}

type MongoDBConfig struct {
	URI string
}

type KafkaConfig struct {
	Host string
	Port int
}

type PartyServiceConfig struct {
	Host string
	Port uint16

	SettingsHost string
	SettingsPort int
}

type LobbyConfig struct {
	FleetName string
	MatchRate time.Duration
	MatchSize int
}

type ProxyConfig struct {
	FleetName string
	MatchRate time.Duration
	MatchSize int
}

func LoadGlobalConfig() Config {
	// Kafka
	viper.SetDefault(kafkaHostFlag, "localhost")
	viper.SetDefault(kafkaPortFlag, 9092)
	// MongoDBV
	viper.SetDefault(mongoDBURIFlag, "mongodb://localhost:27017")
	// PartyService
	viper.SetDefault(partyServiceHostFlag, "localhost")
	viper.SetDefault(partyServicePortFlag, 10006)
	viper.SetDefault(partyServiceSettingsHostFlag, "localhost")
	viper.SetDefault(partyServiceSettingsPortFlag, 10006)
	// Lobby
	viper.SetDefault(lobbyFleetNameFlag, "lobby")
	viper.SetDefault(lobbyMatchRateFlag, 175_000_000)
	viper.SetDefault(lobbyMatchSizeFlag, 50)
	// Proxy
	viper.SetDefault(proxyFleetNameFlag, "velocity")
	viper.SetDefault(proxyMatchRateFlag, 175_000_000)
	viper.SetDefault(proxyMatchSizeFlag, 50)
	// Global
	viper.SetDefault(namespaceFlag, "emortalmc")
	viper.SetDefault(grpcPortFlag, 1007)
	viper.SetDefault(developmentFlag, true)

	pflag.String(kafkaHostFlag, viper.GetString(kafkaHostFlag), "Kafka host")
	pflag.Int32(kafkaPortFlag, viper.GetInt32(kafkaPortFlag), "Kafka port")
	pflag.String(mongoDBURIFlag, viper.GetString(mongoDBURIFlag), "MongoDB URI")
	pflag.String(partyServiceHostFlag, viper.GetString(partyServiceHostFlag), "PartyService host")
	pflag.Int32(partyServicePortFlag, viper.GetInt32(partyServicePortFlag), "PartyService port")
	pflag.String(partyServiceSettingsHostFlag, viper.GetString(partyServiceSettingsHostFlag), "PartyService settings host")
	pflag.Int32(partyServiceSettingsPortFlag, viper.GetInt32(partyServiceSettingsPortFlag), "PartyService settings port")
	pflag.String(lobbyFleetNameFlag, viper.GetString(lobbyFleetNameFlag), "Lobby fleet name")
	pflag.Duration(lobbyMatchRateFlag, viper.GetDuration(lobbyMatchRateFlag), "Delay between creating lobby matches")
	pflag.Int32(lobbyMatchSizeFlag, viper.GetInt32(lobbyMatchSizeFlag), "Maximum size of a lobby (accounts for players already in the lobby)")
	pflag.String(proxyFleetNameFlag, viper.GetString(proxyFleetNameFlag), "Proxy fleet name (default velocity)")
	pflag.Duration(proxyMatchRateFlag, viper.GetDuration(proxyMatchRateFlag), "Delay between creating proxy matches")
	pflag.Int32(proxyMatchSizeFlag, viper.GetInt32(proxyMatchSizeFlag), "Maximum size of a proxy (accounts for players already in the proxy)")
	pflag.String(namespaceFlag, viper.GetString(namespaceFlag), "Namespace that the resource is in")
	pflag.Int32(grpcPortFlag, viper.GetInt32(grpcPortFlag), "gRPC port of THIS service")
	pflag.Bool(developmentFlag, viper.GetBool(developmentFlag), "Development mode")
	pflag.Parse()

	// Bind the viper flags to environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(kafkaHostFlag))
	runtime.Must(viper.BindEnv(kafkaPortFlag))
	runtime.Must(viper.BindEnv(mongoDBURIFlag))
	runtime.Must(viper.BindEnv(partyServiceHostFlag))
	runtime.Must(viper.BindEnv(partyServicePortFlag))
	runtime.Must(viper.BindEnv(partyServiceSettingsHostFlag))
	runtime.Must(viper.BindEnv(partyServiceSettingsPortFlag))
	runtime.Must(viper.BindEnv(lobbyFleetNameFlag))
	runtime.Must(viper.BindEnv(lobbyMatchRateFlag))
	runtime.Must(viper.BindEnv(lobbyMatchSizeFlag))
	runtime.Must(viper.BindEnv(proxyFleetNameFlag))
	runtime.Must(viper.BindEnv(proxyMatchRateFlag))
	runtime.Must(viper.BindEnv(proxyMatchSizeFlag))
	runtime.Must(viper.BindEnv(namespaceFlag))
	runtime.Must(viper.BindEnv(grpcPortFlag))
	runtime.Must(viper.BindEnv(developmentFlag))

	return Config{
		Kafka: KafkaConfig{
			Host: viper.GetString(kafkaHostFlag),
			Port: int(viper.GetInt32(kafkaPortFlag)),
		},
		MongoDB: MongoDBConfig{
			URI: viper.GetString(mongoDBURIFlag),
		},
		PartyService: PartyServiceConfig{
			Host:         viper.GetString(partyServiceHostFlag),
			Port:         uint16(viper.GetInt32(partyServicePortFlag)),
			SettingsHost: viper.GetString(partyServiceSettingsHostFlag),
			SettingsPort: int(viper.GetInt32(partyServiceSettingsPortFlag)),
		},
		Lobby: LobbyConfig{
			FleetName: viper.GetString(lobbyFleetNameFlag),
			MatchRate: viper.GetDuration(lobbyMatchRateFlag),
			MatchSize: int(viper.GetInt32(lobbyMatchSizeFlag)),
		},
		Proxy: ProxyConfig{
			FleetName: viper.GetString(proxyFleetNameFlag),
			MatchRate: viper.GetDuration(proxyMatchRateFlag),
			MatchSize: int(viper.GetInt32(proxyMatchSizeFlag)),
		},
		Namespace:   viper.GetString(namespaceFlag),
		GrpcPort:    int(viper.GetInt32(grpcPortFlag)),
		Development: viper.GetBool(developmentFlag),
	}
}
