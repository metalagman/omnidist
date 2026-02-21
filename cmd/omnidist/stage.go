package main

import (
	"fmt"
	"os"

	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var (
	stageDevFlag  bool
	stageOnlyFlag string
)

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Stage artifacts for configured distributions",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		distributions, err := resolveDistributions(stageOnlyFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error resolving distributions:", err)
			os.Exit(1)
		}

		if err := runDistributionSteps(distributions, func(dist distribution) error {
			switch dist {
			case distributionNPM:
				fmt.Println("==> npm stage")
				if err := npmworkflow.Stage(cfg, npmworkflow.StageOptions{Dev: stageDevFlag}); err != nil {
					return fmt.Errorf("npm stage failed: %w", err)
				}
				fmt.Println("npm stage completed")
			case distributionUV:
				fmt.Println("==> uv stage")
				if err := uvworkflow.CheckDependency(); err != nil {
					return err
				}
				if err := uvworkflow.Stage(cfg, uvworkflow.StageOptions{Dev: stageDevFlag}); err != nil {
					return fmt.Errorf("uv stage failed: %w", err)
				}
				fmt.Println("uv stage completed")
			}
			return nil
		}); err != nil {
			fmt.Fprintln(os.Stderr, "Error staging:", err)
			os.Exit(1)
		}

		fmt.Printf("Staging completed successfully for: %s\n", distributionList(distributions))
	},
}

func init() {
	stageCmd.Flags().BoolVar(&stageDevFlag, "dev", false, "Generate dev versions during staging")
	stageCmd.Flags().StringVar(&stageOnlyFlag, "only", "", "Run only selected distributions (comma-separated: npm,uv)")
	AddCommand(stageCmd)
}
