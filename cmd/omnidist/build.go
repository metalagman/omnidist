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
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		var buildVersion string
		if version, err := shared.ResolveVersion(cfg, false); err == nil {
			buildVersion = version
			fmt.Println("Version:", version)
		} else {
			fmt.Fprintln(os.Stderr, "Warning: unable to resolve version:", err)
		}
		if buildVersion != "" {
			prevVersion, hadPrevVersion := os.LookupEnv(shared.EnvVersionName)
			if err := os.Setenv(shared.EnvVersionName, buildVersion); err != nil {
				fmt.Fprintln(os.Stderr, "Error exporting build version:", err)
				os.Exit(1)
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
			fmt.Fprintln(os.Stderr, "Error exporting build metadata:", err)
			os.Exit(1)
		}
		defer restoreBuildMetadataEnv()

		if err := workflow.Build(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error building:", err)
			os.Exit(1)
		}
		if buildVersion != "" {
			if err := shared.WriteBuildVersion(buildVersion); err != nil {
				fmt.Fprintln(os.Stderr, "Error writing build version:", err)
				os.Exit(1)
			}
			fmt.Println("Build version saved:", buildVersion)
		}

		fmt.Println("Build completed successfully")
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
