package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/metalagman/omnidist/internal/config"
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

		if err := runBuild(cfg); err != nil {
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

func runBuild(cfg *config.Config) error {
	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		if err := buildTarget(cfg, target); err != nil {
			return fmt.Errorf("failed to build %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func buildTarget(cfg *config.Config, target config.Target) error {
	outputDir := filepath.Join("dist", target.OS, config.MapArchToNPM(target.Arch))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputName := cfg.Tool.Name
	if target.OS == "win32" {
		outputName += ".exe"
	}
	outputPath := filepath.Join(outputDir, outputName)

	args := []string{"build"}
	if cfg.Build.Ldflags != "" {
		args = append(args, "-ldflags", cfg.Build.Ldflags)
	}
	for _, tag := range cfg.Build.Tags {
		args = append(args, "-tags", tag)
	}
	args = append(args, "-o", outputPath, cfg.Tool.Main)

	buildCmd := exec.Command("go", args...)
	buildCmd.Env = append(os.Environ(), "GOOS="+config.MapOSToGo(target.OS), "GOARCH="+config.MapArchFromNPM(target.Arch))
	if cfg.Build.CGO {
		buildCmd.Env = append(buildCmd.Env, "CGO_ENABLED=1")
	} else {
		buildCmd.Env = append(buildCmd.Env, "CGO_ENABLED=0")
	}
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return err
	}

	if target.OS != "win32" {
		if err := os.Chmod(outputPath, 0755); err != nil {
			return err
		}
	}

	fmt.Printf("Built: %s\n", outputPath)
	return nil
}

func init() {
	AddCommand(buildCmd)
}
