package main

import (
	"fmt"
	"os"

	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var verifyOnlyFlag string

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify staged artifacts for configured distributions",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		distributions, err := resolveDistributions(verifyOnlyFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error resolving distributions:", err)
			os.Exit(1)
		}

		if err := runDistributionSteps(distributions, func(dist distribution) error {
			switch dist {
			case distributionNPM:
				fmt.Println("==> npm verify")
				result := npmworkflow.Verify(cfg)
				if err := verifyResult("npm", result.Errors, result.Warnings, result.Valid); err != nil {
					return err
				}
				fmt.Println("npm verify passed")
			case distributionUV:
				fmt.Println("==> uv verify")
				if err := uvworkflow.CheckDependency(); err != nil {
					return err
				}
				result := uvworkflow.Verify(cfg)
				if err := verifyResult("uv", result.Errors, result.Warnings, result.Valid); err != nil {
					return err
				}
				fmt.Println("uv verify passed")
			}
			return nil
		}); err != nil {
			fmt.Fprintln(os.Stderr, "Error verifying:", err)
			os.Exit(1)
		}

		fmt.Printf("Verification completed successfully for: %s\n", distributionList(distributions))
	},
}

func verifyResult(name string, errors []string, warnings []string, valid bool) error {
	for _, warning := range warnings {
		fmt.Printf("%s warning: %s\n", name, warning)
	}
	if valid {
		return nil
	}
	for _, errText := range errors {
		fmt.Printf("%s error: %s\n", name, errText)
	}
	if len(errors) == 0 {
		return fmt.Errorf("%s verify failed", name)
	}
	return fmt.Errorf("%s verify failed with %d error(s)", name, len(errors))
}

func init() {
	verifyCmd.Flags().StringVar(&verifyOnlyFlag, "only", "", "Run only selected distributions (comma-separated: npm,uv)")
	AddCommand(verifyCmd)
}
