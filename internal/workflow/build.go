package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

// Build compiles the configured Go CLI for all configured targets into `dist/`.
func Build(cfg *config.Config) error {
	if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
		return fmt.Errorf("create dist directory %s: %w", paths.DistDir, err)
	}

	for _, target := range cfg.Targets {
		if err := buildTarget(cfg, target); err != nil {
			return fmt.Errorf("failed to build %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func buildTarget(cfg *config.Config, target config.Target) error {
	outputDir := filepath.Join(paths.DistDir, target.OS, target.Arch)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create target output directory %s: %w", outputDir, err)
	}

	outputName := cfg.Tool.Name
	if target.OS == "windows" {
		outputName += ".exe"
	}
	outputPath := filepath.Join(outputDir, outputName)

	args := []string{"build"}
	ldflags := renderBuildLDFlags(cfg.Build.Ldflags)
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	if tags := buildTagsFlagValue(cfg.Build.Tags); tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, "-o", outputPath, cfg.Tool.Main)

	buildCmd := exec.Command("go", args...)
	buildCmd.Env = append(os.Environ(), "GOOS="+target.OS, "GOARCH="+target.Arch)
	if cfg.Build.CGO {
		buildCmd.Env = append(buildCmd.Env, "CGO_ENABLED=1")
	} else {
		buildCmd.Env = append(buildCmd.Env, "CGO_ENABLED=0")
	}
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("run go build for %s: %w", outputPath, err)
	}

	if target.OS != "windows" {
		if err := os.Chmod(outputPath, 0755); err != nil {
			return fmt.Errorf("chmod built binary %s: %w", outputPath, err)
		}
	}

	fmt.Printf("Built: %s\n", outputPath)
	return nil
}

func renderBuildLDFlags(template string) string {
	return strings.TrimSpace(os.ExpandEnv(template))
}

func buildTagsFlagValue(tags []string) string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		filtered = append(filtered, tag)
	}
	return strings.Join(filtered, ",")
}
