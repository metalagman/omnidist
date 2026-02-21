package npm

import (
	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/spf13/viper"
)

func loadConfig() (*config.Config, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = paths.ConfigPath
	}
	return config.Load(configFile)
}
