package main

import (
	"fmt"
	"os"

	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap omnidist workspace in existing Go repo",
	Run: func(cmd *cobra.Command, args []string) {
		if err := workflow.Init(paths.ConfigPath); err != nil {
			fmt.Fprintln(os.Stderr, "Error initializing project:", err)
			os.Exit(1)
		}

		fmt.Printf("Created %s\n", paths.ConfigPath)
		fmt.Printf("Created %s workspace and updated .gitignore\n", paths.WorkspaceDir)
	},
}

func init() {
	AddCommand(initCmd)
}
