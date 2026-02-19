package npm

import (
	"fmt"
	"os"

	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	"github.com/spf13/cobra"
)

var (
	flagDryRun   bool
	flagTag      string
	flagRegistry string
	flagOTP      string
)

func init() {
	Cmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Run without actually publishing")
	publishCmd.Flags().StringVar(&flagTag, "tag", "", "NPM tag to use")
	publishCmd.Flags().StringVar(&flagRegistry, "registry", "", "NPM registry URL")
	publishCmd.Flags().StringVar(&flagOTP, "otp", "", "One-time password for 2FA")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to npm registry",
	Run: func(cmd *cobra.Command, args []string) {
		if err := npmworkflow.CheckAuth(); err != nil {
			fmt.Fprintln(os.Stderr, "NPM authentication failed:", err)
			fmt.Fprintln(os.Stderr, "Run 'npm login' to authenticate")
			os.Exit(1)
		}

		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		opts := npmworkflow.PublishOptions{
			DryRun:   flagDryRun,
			Tag:      flagTag,
			Registry: flagRegistry,
			OTP:      flagOTP,
		}

		if err := npmworkflow.Publish(cfg, opts); err != nil {
			fmt.Fprintln(os.Stderr, "Error publishing:", err)
			os.Exit(1)
		}

		fmt.Println("Publish completed successfully")
	},
}
