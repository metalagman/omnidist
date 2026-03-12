package npm

import (
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/spf13/viper"
)

func getConfigPath() string {
	configFile := strings.TrimSpace(viper.GetString("config"))
	if configFile != "" {
		return configFile
	}
	configFile = strings.TrimSpace(viper.ConfigFileUsed())
	if configFile != "" {
		return configFile
	}
	return paths.ConfigPath
}

func getSelectedProfile() string {
	profile := strings.TrimSpace(viper.GetString("profile"))
	if profile == "" {
		return config.DefaultProfileName
	}
	return profile
}

func loadConfig() (*config.Config, error) {
	return config.LoadWithProfile(getConfigPath(), getSelectedProfile())
}
