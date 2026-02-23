package main

import (
	"fmt"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		distributions, err := resolveDistributions(stageOnlyFlag)
		if err != nil {
			return fmt.Errorf("resolve distributions: %w", err)
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
			return fmt.Errorf("stage: %w", err)
		}

		fmt.Printf("Staging completed successfully for: %s\n", distributionList(distributions))
		return nil
	},
}

func init() {
	stageCmd.Flags().BoolVar(&stageDevFlag, "dev", false, "Generate dev versions during staging")
	stageCmd.Flags().StringVar(&stageOnlyFlag, "only", "", "Run only selected distributions (comma-separated: npm,uv)")
	AddCommand(stageCmd)
}
