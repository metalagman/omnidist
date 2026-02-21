package uv

import (
	"github.com/metalagman/omnidist/internal/config"
	"github.com/spf13/viper"
)

func loadConfig() (*config.Config, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = "omnidist.yaml"
	}
	return config.Load(configFile)
}
