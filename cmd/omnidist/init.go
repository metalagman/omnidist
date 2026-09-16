package main

import (
	"fmt"

	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	initForce    bool
	initToolName string
	initToolMain string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap omnidist workspace in existing Go repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := getConfigPath()
		opts := workflow.InitOptions{
			Force:    initForce,
			ToolName: initToolName,
			ToolMain: initToolMain,
		}
		if err := workflow.Init(cfgPath, opts); err != nil {
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
		fmt.Fprintln(out, "5. omnidist verify")
		fmt.Fprintln(out, "6. omnidist publish (publishing to registries cannot be rolled back atomically)")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Replace an existing config file")
	initCmd.Flags().StringVar(&initToolName, "name", "", "Tool and package name (inferred from cmd/* when omitted)")
	initCmd.Flags().StringVar(&initToolMain, "main", "", "Go main package path (inferred from cmd/* when omitted)")
	AddCommand(initCmd)
}
