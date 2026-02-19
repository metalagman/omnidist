package npm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagDryRun   bool
	flagTag      string
	flagRegistry string
	flagOTP      string
)

func init() {
	Cmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Run without actually publishing")
	publishCmd.Flags().StringVar(&flagTag, "tag", "", "NPM tag to use")
	publishCmd.Flags().StringVar(&flagRegistry, "registry", "", "NPM registry URL")
	publishCmd.Flags().StringVar(&flagOTP, "otp", "", "One-time password for 2FA")
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to npm registry",
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkAuth(); err != nil {
			fmt.Fprintln(os.Stderr, "NPM authentication failed:", err)
			fmt.Fprintln(os.Stderr, "Run 'npm login' to authenticate")
			os.Exit(1)
		}

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
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}

	platformPackages := []string{}
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(npmDist.Package, target)
		platformPackages = append(platformPackages, pkgName)
	}

	access := npmDist.Access
	if access == "" {
		access = "public"
	}

	fmt.Println("Publishing platform packages first...")
	for _, pkgName := range platformPackages {
		pkgDir := filepath.Join("npm", pkgName)
		if err := publishPackage(pkgDir, npmDist.Registry, access); err != nil {
			return fmt.Errorf("failed to publish %s: %w", pkgName, err)
		}
		fmt.Printf("Published: %s\n", pkgName)
	}

	fmt.Println("Publishing meta package...")
	metaDir := filepath.Join("npm", npmDist.Package)
	if err := publishPackage(metaDir, npmDist.Registry, access); err != nil {
		return fmt.Errorf("failed to publish meta package: %w", err)
	}
	fmt.Printf("Published: %s\n", npmDist.Package)

	return nil
}

func buildPublishArgs(defaultRegistry, defaultAccess string) []string {
	args := []string{"publish"}

	access := strings.TrimSpace(defaultAccess)
	if access == "" {
		access = "public"
	}
	args = append(args, "--access", access)

	if flagDryRun {
		args = append(args, "--dry-run")
	}
	if flagTag != "" {
		args = append(args, "--tag", flagTag)
	}

	registry := strings.TrimSpace(defaultRegistry)
	if flagRegistry != "" {
		registry = strings.TrimSpace(flagRegistry)
	}
	if registry != "" {
		args = append(args, "--registry", registry)
	}
	if flagOTP != "" {
		args = append(args, "--otp", flagOTP)
	}

	return args
}

func publishPackage(dir, defaultRegistry, defaultAccess string) error {
	args := buildPublishArgs(defaultRegistry, defaultAccess)

	execCmd := exec.Command("npm", args...)
	execCmd.Dir = dir
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}
