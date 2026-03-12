package npm

import (
	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/spf13/viper"
)

func getConfigPath() string {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = paths.ConfigPath
	}
	return configFile
}

func loadConfig() (*config.Config, error) {
	return config.Load(getConfigPath())
}
