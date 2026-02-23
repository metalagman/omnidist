package main

import (
	"fmt"

	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var (
	publishDryRunFlag bool
	publishOnlyFlag   string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish staged artifacts for configured distributions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		distributions, err := resolveDistributions(publishOnlyFlag)
		if err != nil {
			return fmt.Errorf("resolve distributions: %w", err)
		}

		if err := runDistributionSteps(distributions, func(dist distribution) error {
			switch dist {
			case distributionNPM:
				fmt.Println("==> npm publish")
				if err := npmworkflow.CheckAuth(cfg, "", publishDryRunFlag); err != nil {
					return fmt.Errorf("npm authentication failed: %w", err)
				}
				if err := npmworkflow.Publish(cfg, npmworkflow.PublishOptions{DryRun: publishDryRunFlag}); err != nil {
					return fmt.Errorf("npm publish failed: %w", err)
				}
				fmt.Println("npm publish completed")
			case distributionUV:
				fmt.Println("==> uv publish")
				if err := uvworkflow.CheckDependency(); err != nil {
					return err
				}
				if err := uvworkflow.Publish(cfg, uvworkflow.PublishOptions{DryRun: publishDryRunFlag}); err != nil {
					return fmt.Errorf("uv publish failed: %w", err)
				}
				fmt.Println("uv publish completed")
			}
			return nil
		}); err != nil {
			return fmt.Errorf("publish: %w", err)
		}

		fmt.Printf("Publish completed successfully for: %s\n", distributionList(distributions))
		return nil
	},
}

func init() {
	publishCmd.Flags().BoolVar(&publishDryRunFlag, "dry-run", false, "Run publish without uploading artifacts")
	publishCmd.Flags().StringVar(&publishOnlyFlag, "only", "", "Run only selected distributions (comma-separated: npm,uv)")
	AddCommand(publishCmd)
}
