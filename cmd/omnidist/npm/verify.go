package npm

import (
	"fmt"
	"os"

	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Enforce correctness before publishing",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		result := npmworkflow.Verify(cfg)

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
		} else {
			os.Exit(1)
		}
	},
}
