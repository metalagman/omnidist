package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	appversion "github.com/metalagman/appkit/version"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/spf13/cobra"
)

var ciForceFlag bool

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Generate GitHub Actions release workflow for omnidist",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("config %s not found; run `omnidist init` first", paths.ConfigPath)
			}
			return fmt.Errorf("load config: %w", err)
		}

		npxVersion, fallbackToLatest := resolveCINPXVersion()
		content, err := workflow.GenerateGitHubReleaseWorkflow(cfg, workflow.CIWorkflowOptions{
			NPXVersion: npxVersion,
		})
		if err != nil {
			return fmt.Errorf("generate workflow: %w", err)
		}

		if err := workflow.WriteGitHubReleaseWorkflow(workflow.DefaultCIWorkflowPath, content, ciForceFlag); err != nil {
			return fmt.Errorf("write workflow: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", workflow.DefaultCIWorkflowPath)
		if fallbackToLatest {
			fmt.Fprintln(cmd.OutOrStdout(), "Info: omnidist runtime version is non-release, workflow uses @latest for npx.")
		}
		return nil
	},
}

func resolveCINPXVersion() (string, bool) {
	version := strings.TrimSpace(appversion.GetVersion())
	version = strings.TrimPrefix(version, "v")
	if version == "" || version == "dev" || strings.HasPrefix(version, "dev+") {
		return "latest", true
	}
	return version, false
}

func init() {
	ciCmd.Flags().BoolVar(&ciForceFlag, "force", false, "Overwrite existing workflow file if it already exists")
	AddCommand(ciCmd)
}
