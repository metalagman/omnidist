package gem

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
	cfg, err := config.LoadWithProfile(getConfigPath(), getSelectedProfile())
	if err != nil {
		return nil, err
	}
	if _, err := cfg.RequireGem(); err != nil {
		return nil, err
	}
	return cfg, nil
}
