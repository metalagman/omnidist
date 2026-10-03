package uv

import (
	"fmt"
	"strings"

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
	publishCmd.Flags().StringVar(&publishURL, "publish-url", "", "Override upload endpoint (or set PYPI_PUBLISH_URL; UV_PUBLISH_URL is also supported)")
	publishCmd.Flags().StringVar(&publishLegacyURL, "repository-url", "", "Deprecated alias for --publish-url")
	_ = publishCmd.Flags().MarkDeprecated("repository-url", "use --publish-url instead")
	publishCmd.Flags().StringVar(&publishToken, "token", "", "PyPI publish token (or set PYPI_PUBLISH_TOKEN; UV_PUBLISH_TOKEN is also supported)")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish PyPI wheel artifacts to a PyPI-compatible index",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		opts := uvworkflow.PublishOptions{
			DryRun:     publishDryRun,
			PublishURL: strings.TrimSpace(publishURL),
			Token:      publishToken,
			Stdout:     cmd.OutOrStdout(),
			Stderr:     cmd.ErrOrStderr(),
		}
		if opts.PublishURL == "" {
			opts.PublishURL = publishLegacyURL
		}
		if err := uvworkflow.PreflightPublish(cfg, opts); err != nil {
			return fmt.Errorf("PyPI publish preflight failed: %w", err)
		}

		if err := uvworkflow.Publish(cfg, opts); err != nil {
			return fmt.Errorf("publish PyPI artifacts: %w", err)
		}

		fmt.Println("PyPI publish completed successfully")
		return nil
	},
}
