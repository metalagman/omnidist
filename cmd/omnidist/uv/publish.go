package uv

import (
	"fmt"

	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var (
	publishDryRun    bool
	publishURL       string
	publishLegacyURL string
	publishToken     string
)

func init() {
	Cmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&publishDryRun, "dry-run", false, "Run publish without uploading artifacts")
	publishCmd.Flags().StringVar(&publishURL, "publish-url", "", "Override uv publish URL (upload endpoint)")
	publishCmd.Flags().StringVar(&publishLegacyURL, "repository-url", "", "Deprecated alias for --publish-url")
	_ = publishCmd.Flags().MarkDeprecated("repository-url", "use --publish-url instead")
	publishCmd.Flags().StringVar(&publishToken, "token", "", "PyPI token for uv publish (or set UV_PUBLISH_TOKEN)")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish uv wheel artifacts to a PyPI-compatible index",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := uvworkflow.CheckDependency(); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		opts := uvworkflow.PublishOptions{
			DryRun:     publishDryRun,
			PublishURL: publishURL,
			Token:      publishToken,
			Stdout:     cmd.OutOrStdout(),
			Stderr:     cmd.ErrOrStderr(),
		}
		if opts.PublishURL == "" {
			opts.PublishURL = publishLegacyURL
		}

		if err := uvworkflow.Publish(cfg, opts); err != nil {
			return fmt.Errorf("publish uv artifacts: %w", err)
		}

		fmt.Println("UV publish completed successfully")
		return nil
	},
}
