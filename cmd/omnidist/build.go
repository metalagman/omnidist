package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	envBuildCommitName = "OMNIDIST_GIT_COMMIT"
	envBuildDateName   = "OMNIDIST_BUILD_DATE"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile Go binaries for configured targets",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		var buildVersion string
		if version, err := shared.ResolveVersion(cfg, false); err == nil {
			buildVersion = version
			fmt.Println("Version:", version)
		} else {
			fmt.Fprintln(cmd.ErrOrStderr(), "Warning: unable to resolve version:", err)
		}
		if buildVersion != "" {
			prevVersion, hadPrevVersion := os.LookupEnv(shared.EnvVersionName)
			if err := os.Setenv(shared.EnvVersionName, buildVersion); err != nil {
				return fmt.Errorf("export build version: %w", err)
			}
			defer func() {
				if hadPrevVersion {
					_ = os.Setenv(shared.EnvVersionName, prevVersion)
					return
				}
				_ = os.Unsetenv(shared.EnvVersionName)
			}()
		}
		restoreBuildMetadataEnv, err := setBuildMetadataEnv()
		if err != nil {
			return fmt.Errorf("export build metadata: %w", err)
		}
		defer restoreBuildMetadataEnv()

		if err := workflow.Build(cfg); err != nil {
			return fmt.Errorf("build: %w", err)
		}
		if buildVersion != "" {
			if err := shared.WriteBuildVersion(buildVersion); err != nil {
				return fmt.Errorf("write build version: %w", err)
			}
			fmt.Println("Build version saved:", buildVersion)
		}

		fmt.Println("Build completed successfully")
		return nil
	},
}

func loadConfig() (*config.Config, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = paths.ConfigPath
	}
	return config.Load(configFile)
}

func init() {
	AddCommand(buildCmd)
}

func setBuildMetadataEnv() (func(), error) {
	restoreFns := []func(){}

	commit := resolveBuildGitCommit()
	if commit != "" {
		prev, hadPrev := os.LookupEnv(envBuildCommitName)
		if err := os.Setenv(envBuildCommitName, commit); err != nil {
			return nil, err
		}
		restoreFns = append(restoreFns, func() {
			if hadPrev {
				_ = os.Setenv(envBuildCommitName, prev)
				return
			}
			_ = os.Unsetenv(envBuildCommitName)
		})
	}

	buildDate := time.Now().UTC().Format(time.RFC3339)
	prevDate, hadPrevDate := os.LookupEnv(envBuildDateName)
	if err := os.Setenv(envBuildDateName, buildDate); err != nil {
		return nil, err
	}
	restoreFns = append(restoreFns, func() {
		if hadPrevDate {
			_ = os.Setenv(envBuildDateName, prevDate)
			return
		}
		_ = os.Unsetenv(envBuildDateName)
	})

	return func() {
		for i := len(restoreFns) - 1; i >= 0; i-- {
			restoreFns[i]()
		}
	}, nil
}

func resolveBuildGitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
