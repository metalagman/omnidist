package npm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/metalagman/omnidist/internal/config"
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
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		if err := runStage(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error staging:", err)
			os.Exit(1)
		}

		fmt.Println("Staging completed successfully")
	},
}

func runStage(cfg *config.Config) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}

	version, err := getVersion(cfg, flagDev)
	if err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		if err := stagePlatformPackage(cfg, npmDist, target, version); err != nil {
			return fmt.Errorf("failed to stage %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	if err := stageMetaPackage(cfg, npmDist, version); err != nil {
		return fmt.Errorf("failed to stage meta package: %w", err)
	}

	return nil
}

func stagePlatformPackage(cfg *config.Config, npmDist config.DistributionConfig, target config.Target, version string) error {
	pkgName := platformPackageName(npmDist.Package, target)
	pkgDir := filepath.Join("npm", pkgName)

	if err := os.MkdirAll(filepath.Join(pkgDir, "bin"), 0755); err != nil {
		return err
	}

	binaryName := cfg.Tool.Name
	if target.OS == "win32" {
		binaryName += ".exe"
	}

	srcPath := filepath.Join("dist", target.OS, config.MapArchToNPM(target.Arch), binaryName)
	dstPath := filepath.Join(pkgDir, "bin", binaryName)

	if err := copyFile(srcPath, dstPath); err != nil {
		return err
	}

	if target.OS != "win32" {
		if err := os.Chmod(dstPath, 0755); err != nil {
			return err
		}
	}

	pkgJSON := map[string]interface{}{
		"name":        pkgName,
		"version":     version,
		"description": npmDist.Package + " binary for " + target.OS + "/" + target.Arch,
		"os":          []string{target.OS},
		"cpu":         []string{config.MapArchToNPM(target.Arch)},
		"bin": map[string]string{
			cfg.Tool.Name: "bin/" + binaryName,
		},
		"files": []string{"bin"},
	}

	return writePackageJSON(pkgDir, pkgJSON)
}

func stageMetaPackage(cfg *config.Config, npmDist config.DistributionConfig, version string) error {
	metaDir := filepath.Join("npm", npmDist.Package)

	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	optionalDeps := make(map[string]string)
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(npmDist.Package, target)
		optionalDeps[pkgName] = version
	}

	pkgJSON := map[string]interface{}{
		"name":                 npmDist.Package,
		"version":              version,
		"description":          "Meta package for " + cfg.Tool.Name,
		"bin":                  map[string]string{cfg.Tool.Name: cfg.Tool.Name + ".js"},
		"optionalDependencies": optionalDeps,
		"engines":              map[string]string{"node": ">=16"},
		"files":                []string{cfg.Tool.Name + ".js"},
	}

	if err := writePackageJSON(metaDir, pkgJSON); err != nil {
		return err
	}

	shimPath := filepath.Join(metaDir, cfg.Tool.Name+".js")
	if err := writeShim(shimPath, cfg.Tool.Name, npmDist.Package); err != nil {
		return err
	}

	return nil
}
