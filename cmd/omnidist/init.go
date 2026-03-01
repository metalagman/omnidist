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

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Created %s\n", paths.ConfigPath)
		fmt.Fprintf(out, "Created %s workspace and %s/.gitignore\n\n", paths.WorkspaceDir, paths.WorkspaceDir)
		fmt.Fprintln(out, "Next steps:")
		fmt.Fprintf(out, "1. Edit %s\n", paths.ConfigPath)
		fmt.Fprintln(out, "2. Set environment variables in .env (loaded automatically by omnidist)")
		fmt.Fprintln(out, "3. omnidist build")
		fmt.Fprintln(out, "4. omnidist stage")
		fmt.Fprintln(out, "5. omnidist publish")
		return nil
	},
}

func init() {
	AddCommand(initCmd)
}
