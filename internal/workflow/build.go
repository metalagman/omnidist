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

func Build(cfg *config.Config) error {
	if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		if err := buildTarget(cfg, target); err != nil {
			return fmt.Errorf("failed to build %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func buildTarget(cfg *config.Config, target config.Target) error {
	outputDir := filepath.Join(paths.DistDir, target.OS, config.MapArchToNPM(target.Arch))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputName := cfg.Tool.Name
	if target.OS == "win32" {
		outputName += ".exe"
	}
	outputPath := filepath.Join(outputDir, outputName)

	args := []string{"build"}
	ldflags := renderBuildLDFlags(cfg.Build.Ldflags)
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	for _, tag := range cfg.Build.Tags {
		args = append(args, "-tags", tag)
	}
	args = append(args, "-o", outputPath, cfg.Tool.Main)

	buildCmd := exec.Command("go", args...)
	buildCmd.Env = append(os.Environ(), "GOOS="+config.MapOSToGo(target.OS), "GOARCH="+config.MapArchFromNPM(target.Arch))
	if cfg.Build.CGO {
		buildCmd.Env = append(buildCmd.Env, "CGO_ENABLED=1")
	} else {
		buildCmd.Env = append(buildCmd.Env, "CGO_ENABLED=0")
	}
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return err
	}

	if target.OS != "win32" {
		if err := os.Chmod(outputPath, 0755); err != nil {
			return err
		}
	}

	fmt.Printf("Built: %s\n", outputPath)
	return nil
}

func renderBuildLDFlags(template string) string {
	return strings.TrimSpace(os.ExpandEnv(template))
}
