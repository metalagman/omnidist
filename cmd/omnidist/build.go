package main

import (
	"fmt"
	"os"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile Go binaries for configured targets",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		if err := workflow.Build(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error building:", err)
			os.Exit(1)
		}

		fmt.Println("Build completed successfully")
	},
}

func loadConfig() (*config.Config, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = "omnidist.yaml"
	}
	return config.Load(configFile)
}

func init() {
	AddCommand(buildCmd)
}
