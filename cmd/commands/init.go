package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/metalagman/go2npm/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap npm workspace packaging in existing Go repo",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.DefaultConfig()

		if err := config.Save(cfg, "go2npm.yaml"); err != nil {
			fmt.Fprintln(os.Stderr, "Error saving config:", err)
			os.Exit(1)
		}

		fmt.Println("Created go2npm.yaml")

		if err := createNpmStructure(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error creating npm structure:", err)
			os.Exit(1)
		}

		fmt.Println("Created npm/ directory structure")
	},
}

func createNpmStructure(cfg *config.Config) error {
	baseDir := "npm"

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	metaDir := filepath.Join(baseDir, cfg.NPM.Package)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		pkgDir := fmt.Sprintf("%s-%s-%s", cfg.NPM.Package, target.OS, target.CPU)
		if target.Variant != "" {
			pkgDir = fmt.Sprintf("%s-%s", pkgDir, target.Variant)
		}
		if err := os.MkdirAll(filepath.Join(baseDir, pkgDir, "bin"), 0755); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	AddCommand(initCmd)
}
