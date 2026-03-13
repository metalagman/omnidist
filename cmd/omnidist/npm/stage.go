package npm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	"github.com/spf13/cobra"
)

var flagDev bool

func init() {
	Cmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVar(&flagDev, "dev", false, "Generate dev version (appends -dev.<commits> to version)")
}

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Assemble npm packages from built artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := workflow.EnsureWorkspaceGitignore(filepath.Join(paths.WorkspaceDir, ".gitignore")); err != nil {
			return fmt.Errorf("ensure workspace gitignore: %w", err)
		}

		version, err := resolveStageVersionForOutput(cfg, flagDev)
		if err != nil {
			return fmt.Errorf("resolve version: %w", err)
		}
		fmt.Println("Version:", version)

		if err := runStage(cfg); err != nil {
			return fmt.Errorf("stage: %w", err)
		}

		fmt.Println("Staging completed successfully")
		return nil
	},
}

func runStage(cfg *config.Config) error {
	return npmworkflow.Stage(cfg, npmworkflow.StageOptions{
		Dev: flagDev,
	})
}

func resolveStageVersionForOutput(cfg *config.Config, dev bool) (string, error) {
	if dev {
		return shared.ResolveVersion(cfg, true)
	}

	version, err := shared.ReadBuildVersionForConfig(cfg)
	if err != nil {
		layout := paths.NewLayout(cfg.EffectiveWorkspaceDir())
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("missing build version file %s; run `omnidist build` before `omnidist npm stage`", layout.DistVersionPath)
		}
		return "", fmt.Errorf("read build version: %w", err)
	}

	return version, nil
}
