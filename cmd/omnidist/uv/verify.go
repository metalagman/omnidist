package uv

import (
	"fmt"

	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify uv wheel artifacts before publishing",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := uvworkflow.CheckDependency(); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		result := uvworkflow.Verify(cfg)

		if len(result.Errors) > 0 {
			fmt.Println("Verification FAILED:")
			for _, e := range result.Errors {
				fmt.Println("  ERROR:", e)
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, w := range result.Warnings {
				fmt.Println("  WARN:", w)
			}
		}

		if result.Valid {
			fmt.Println("Verification PASSED")
			return nil
		}
		return fmt.Errorf("uv verification failed")
	},
}
