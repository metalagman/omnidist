package gem

import (
	"fmt"

	gemworkflow "github.com/metalagman/omnidist/internal/workflow/gem"
	"github.com/spf13/cobra"
)

var (
	publishDryRun bool
	publishHost   string
	publishAPIKey string
	publishOTP    string
)

func init() {
	Cmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&publishDryRun, "dry-run", false, "Run publish without uploading artifacts")
	publishCmd.Flags().StringVar(&publishHost, "host", "", "Override RubyGems host")
	publishCmd.Flags().StringVar(&publishAPIKey, "api-key", "", "RubyGems API key (or set GEM_HOST_API_KEY / RUBYGEMS_API_KEY)")
	publishCmd.Flags().StringVar(&publishOTP, "otp", "", "RubyGems MFA OTP code (or set GEM_HOST_OTP_CODE)")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish staged gem artifacts to a RubyGems-compatible host",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		opts := gemworkflow.PublishOptions{
			DryRun:   publishDryRun,
			Host:     publishHost,
			APIKey:   publishAPIKey,
			OTP:      publishOTP,
			Stdout:   cmd.OutOrStdout(),
			Stderr:   cmd.ErrOrStderr(),
			Progress: cmd.OutOrStdout(),
		}
		if err := gemworkflow.PreflightPublish(cfg, opts); err != nil {
			return fmt.Errorf("gem publish preflight failed: %w", err)
		}
		if err := gemworkflow.Publish(cfg, opts); err != nil {
			return fmt.Errorf("publish gem artifacts: %w", err)
		}

		fmt.Println("Gem publish completed successfully")
		return nil
	},
}
