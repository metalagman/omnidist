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
		cfgPath := getConfigPath()
		if err := workflow.Init(cfgPath); err != nil {
			return fmt.Errorf("initialize project: %w", err)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Created %s\n", cfgPath)
		fmt.Fprintf(out, "Created %s workspace\n\n", paths.WorkspaceDir)
		fmt.Fprintln(out, "Next steps:")
		fmt.Fprintf(out, "1. Edit %s\n", cfgPath)
		fmt.Fprintln(out, "2. Set environment variables in .env as needed (version.source: env, publish tokens)")
		fmt.Fprintln(out, "3. omnidist build")
		fmt.Fprintln(out, "4. omnidist stage")
		fmt.Fprintln(out, "5. omnidist publish")
		return nil
	},
}

func init() {
	AddCommand(initCmd)
}
