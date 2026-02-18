package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/metalagman/go2npm/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagDryRun   bool
	flagTag      string
	flagRegistry string
	flagOTP      string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to npm registry",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		if err := runPublish(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error publishing:", err)
			os.Exit(1)
		}

		fmt.Println("Publish completed successfully")
	},
}

func runPublish(cfg *config.Config) error {
	platformPackages := []string{}
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(cfg.NPM.Package, target)
		platformPackages = append(platformPackages, pkgName)
	}

	fmt.Println("Publishing platform packages first...")
	for _, pkgName := range platformPackages {
		pkgDir := filepath.Join("npm", pkgName)
		if err := publishPackage(pkgDir, cfg.NPM.Registry); err != nil {
			return fmt.Errorf("failed to publish %s: %w", pkgName, err)
		}
		fmt.Printf("Published: %s\n", pkgName)
	}

	fmt.Println("Publishing meta package...")
	metaDir := filepath.Join("npm", cfg.NPM.Package)
	if err := publishPackage(metaDir, cfg.NPM.Registry); err != nil {
		return fmt.Errorf("failed to publish meta package: %w", err)
	}
	fmt.Printf("Published: %s\n", cfg.NPM.Package)

	return nil
}

func publishPackage(dir, registry string) error {
	args := []string{"publish", "--access", "public"}

	if flagDryRun {
		args = append(args, "--dry-run")
	}
	if flagTag != "" {
		args = append(args, "--tag", flagTag)
	}
	if flagRegistry != "" {
		args = append(args, "--registry", flagRegistry)
	}
	if flagOTP != "" {
		args = append(args, "--otp", flagOTP)
	}

	execCmd := exec.Command("npm", args...)
	execCmd.Dir = dir
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

func init() {
	publishCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Simulate publish")
	publishCmd.Flags().StringVar(&flagTag, "tag", "latest", "NPM dist-tag")
	publishCmd.Flags().StringVar(&flagRegistry, "registry", "", "NPM registry URL")
	publishCmd.Flags().StringVar(&flagOTP, "otp", "", "NPM one-time password")

	AddCommand(publishCmd)
}
