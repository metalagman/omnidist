package uv

import (
	"fmt"
	"os"

	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var (
	publishDryRun     bool
	publishRepository string
	publishToken      string
)

func init() {
	Cmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&publishDryRun, "dry-run", false, "Run publish without uploading artifacts")
	publishCmd.Flags().StringVar(&publishRepository, "repository-url", "", "Override uv publish repository URL")
	publishCmd.Flags().StringVar(&publishToken, "token", "", "PyPI token for uv publish (sets UV_PUBLISH_TOKEN)")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish uv wheel artifacts to a PyPI-compatible index",
	Run: func(cmd *cobra.Command, args []string) {
		if err := uvworkflow.CheckDependency(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		opts := uvworkflow.PublishOptions{
			DryRun:        publishDryRun,
			RepositoryURL: publishRepository,
			Token:         publishToken,
		}

		if err := uvworkflow.Publish(cfg, opts); err != nil {
			fmt.Fprintln(os.Stderr, "Error publishing uv artifacts:", err)
			os.Exit(1)
		}

		fmt.Println("UV publish completed successfully")
	},
}
