package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Print a quickstart command sequence for this project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				printQuickstart(cmd, nil)
				return nil
			}
			return fmt.Errorf("load config: %w", err)
		}

		printQuickstart(cmd, cfg)
		return nil
	},
}

func printQuickstart(cmd *cobra.Command, cfg *config.Config) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Quickstart")
	fmt.Fprintln(out)

	if cfg == nil {
		fmt.Fprintf(out, "1. Initialize workspace:\n   omnidist init\n\n")
		fmt.Fprintf(out, "2. Build/stage/verify/publish:\n")
		fmt.Fprintf(out, "   omnidist build\n")
		fmt.Fprintf(out, "   omnidist stage\n")
		fmt.Fprintf(out, "   omnidist verify\n")
		fmt.Fprintf(out, "   omnidist publish\n\n")
		fmt.Fprintf(out, "3. Bootstrap CI workflow:\n   omnidist ci\n")
		return
	}

	fmt.Fprintf(out, "Config: %s\n\n", getConfigPath())
	fmt.Fprintf(out, "1. Build artifacts:\n   omnidist build\n\n")
	fmt.Fprintf(out, "2. Stage and verify:\n")
	fmt.Fprintf(out, "   omnidist stage\n")
	fmt.Fprintf(out, "   omnidist verify\n\n")
	fmt.Fprintf(out, "3. Publish:\n   omnidist publish\n\n")
	fmt.Fprintf(out, "4. Bootstrap CI workflow:\n   omnidist ci\n")

	if strings.TrimSpace(cfg.Version.Source) == "env" {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Version source is env. Export %s before build/stage/publish.\n", shared.EnvVersionName)
	}
}

func init() {
	AddCommand(quickstartCmd)
}
