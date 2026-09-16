package gem

import (
	"fmt"
	"path/filepath"

	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	gemworkflow "github.com/metalagman/omnidist/internal/workflow/gem"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	"github.com/spf13/cobra"
)

var stageDev bool

func init() {
	Cmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVar(&stageDev, "dev", false, "Generate a dev version for gem artifacts")
}

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Assemble Ruby gem artifacts from built binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := workflow.EnsureWorkspaceGitignore(filepath.Join(paths.WorkspaceDir, ".gitignore")); err != nil {
			return fmt.Errorf("ensure workspace gitignore: %w", err)
		}
		if err := gemworkflow.CheckDependency(); err != nil {
			return err
		}

		version, err := shared.ResolveStageVersion(cfg, stageDev)
		if err != nil {
			return fmt.Errorf("resolve version: %w", err)
		}
		gemVersion, err := gemworkflow.NormalizeVersion(version)
		if err != nil {
			return fmt.Errorf("resolve gem version: %w", err)
		}
		fmt.Println("Version:", gemVersion)

		if err := gemworkflow.Stage(cfg, gemworkflow.StageOptions{Dev: stageDev}); err != nil {
			return fmt.Errorf("stage gem artifacts: %w", err)
		}

		fmt.Println("Gem staging completed successfully")
		return nil
	},
}
