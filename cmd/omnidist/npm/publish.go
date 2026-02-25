package npm

import (
	"fmt"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		opts := npmworkflow.PublishOptions{
			DryRun:   flagDryRun,
			Tag:      flagTag,
			Registry: flagRegistry,
			OTP:      flagOTP,
			Stdout:   cmd.OutOrStdout(),
			Stderr:   cmd.ErrOrStderr(),
			Progress: cmd.OutOrStdout(),
		}

		if err := npmworkflow.CheckAuth(cfg, opts.Registry, opts.DryRun); err != nil {
			return fmt.Errorf("npm authentication failed: %w\nSet NPM_PUBLISH_TOKEN in environment for .npmrc substitution, or run 'npm login'", err)
		}

		if err := npmworkflow.Publish(cfg, opts); err != nil {
			return fmt.Errorf("publish: %w", err)
		}

		fmt.Println("Publish completed successfully")
		return nil
	},
}
