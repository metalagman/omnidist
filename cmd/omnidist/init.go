package main

import (
	"fmt"

	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap omnidist workspace in existing Go repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := workflow.Init(paths.ConfigPath); err != nil {
			return fmt.Errorf("initialize project: %w", err)
		}

		fmt.Printf("Created %s\n", paths.ConfigPath)
		fmt.Printf("Created %s workspace and %s/.gitignore\n", paths.WorkspaceDir, paths.WorkspaceDir)
		return nil
	},
}

func init() {
	AddCommand(initCmd)
}
