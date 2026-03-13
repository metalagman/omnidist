package uv

import (
	"fmt"
	"path/filepath"

	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var stageDev bool

func init() {
	Cmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVar(&stageDev, "dev", false, "Generate a dev version for wheel artifacts")
}

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Assemble uv wheel artifacts from built binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := workflow.EnsureWorkspaceGitignore(filepath.Join(paths.WorkspaceDir, ".gitignore")); err != nil {
			return fmt.Errorf("ensure workspace gitignore: %w", err)
		}
		if err := uvworkflow.CheckDependency(); err != nil {
			return err
		}

		version, err := shared.ResolveStageVersion(cfg, stageDev)
		if err != nil {
			return fmt.Errorf("resolve version: %w", err)
		}
		pep440Version, err := shared.ToPEP440(version)
		if err != nil {
			return fmt.Errorf("resolve uv version: %w", err)
		}
		fmt.Println("Version:", pep440Version)

		if err := uvworkflow.Stage(cfg, uvworkflow.StageOptions{Dev: stageDev}); err != nil {
			return fmt.Errorf("stage uv artifacts: %w", err)
		}

		fmt.Println("UV staging completed successfully")
		return nil
	},
}
