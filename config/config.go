package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

var (
	// ELASTIC_HOST ...
	ElasticHost string
)

func init() {
	// Get the config
	InitConfig()
}

// New gets the service configuration
func InitConfig() {
	viper.SetDefault("ELASTIC_HOST", "localhost:9200")

	if os.Getenv("ENVIRONMENT") == "DEV" {
		_, dirname, _, _ := runtime.Caller(0)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(filepath.Dir(dirname))
		viper.ReadInConfig()
	} else {
		viper.AutomaticEnv()
	}

	// Assign env variables value to global variables
	ElasticHost = viper.GetString("ELASTIC_HOST")

}
