package main

import (
	"errors"
	"fmt"

	gemworkflow "github.com/metalagman/omnidist/internal/workflow/gem"
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

		distributions, err := resolveDistributions(cfg, publishOnlyFlag)
		if err != nil {
			return fmt.Errorf("resolve distributions: %w", err)
		}

		preflightErrors := make([]error, 0)
		for _, dist := range distributions {
			switch dist {
			case distributionNPM:
				opts := npmworkflow.PublishOptions{DryRun: publishDryRunFlag}
				if err := npmworkflow.PreflightPublish(cfg, opts); err != nil {
					preflightErrors = append(preflightErrors, fmt.Errorf("npm: %w", err))
				}
			case distributionUV:
				opts := uvworkflow.PublishOptions{DryRun: publishDryRunFlag}
				if err := uvworkflow.PreflightPublish(cfg, opts); err != nil {
					preflightErrors = append(preflightErrors, fmt.Errorf("uv: %w", err))
				}
			case distributionGem:
				opts := gemworkflow.PublishOptions{DryRun: publishDryRunFlag}
				if err := gemworkflow.PreflightPublish(cfg, opts); err != nil {
					preflightErrors = append(preflightErrors, fmt.Errorf("gem: %w", err))
				}
			}
		}
		if len(preflightErrors) > 0 {
			return fmt.Errorf("publish preflight failed; no uploads attempted: %w", errors.Join(preflightErrors...))
		}

		completed := make([]distribution, 0, len(distributions))
		if err := runDistributionSteps(distributions, func(dist distribution) error {
			switch dist {
			case distributionNPM:
				fmt.Println("==> npm publish")
				if err := npmworkflow.Publish(cfg, npmworkflow.PublishOptions{
					DryRun:   publishDryRunFlag,
					Stdout:   cmd.OutOrStdout(),
					Stderr:   cmd.ErrOrStderr(),
					Progress: cmd.OutOrStdout(),
				}); err != nil {
					return fmt.Errorf("npm publish failed: %w", err)
				}
				fmt.Println("npm publish completed")
			case distributionUV:
				fmt.Println("==> uv publish")
				if err := uvworkflow.Publish(cfg, uvworkflow.PublishOptions{
					DryRun: publishDryRunFlag,
					Stdout: cmd.OutOrStdout(),
					Stderr: cmd.ErrOrStderr(),
				}); err != nil {
					return fmt.Errorf("uv publish failed: %w", err)
				}
				fmt.Println("uv publish completed")
			case distributionGem:
				fmt.Println("==> gem publish")
				if err := gemworkflow.Publish(cfg, gemworkflow.PublishOptions{
					DryRun:   publishDryRunFlag,
					Stdout:   cmd.OutOrStdout(),
					Stderr:   cmd.ErrOrStderr(),
					Progress: cmd.OutOrStdout(),
				}); err != nil {
					return fmt.Errorf("gem publish failed: %w", err)
				}
				fmt.Println("gem publish completed")
			}
			completed = append(completed, dist)
			return nil
		}); err != nil {
			if len(completed) > 0 {
				return fmt.Errorf("publish failed after completed backends %s; those registry uploads may already exist and cannot be rolled back atomically: %w", distributionList(completed), err)
			}
			return fmt.Errorf("publish failed before any backend completed; individual package uploads reported above may already exist: %w", err)
		}

		fmt.Printf("Publish completed successfully for: %s\n", distributionList(distributions))
		return nil
	},
}

func init() {
	publishCmd.Flags().BoolVar(&publishDryRunFlag, "dry-run", false, "Run publish without uploading artifacts")
	publishCmd.Flags().StringVar(&publishOnlyFlag, "only", "", "Run only selected distributions (comma-separated: npm,uv,gem)")
	AddCommand(publishCmd)
}
