package workflow

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

// BuildOptions controls subprocess and progress output for BuildWithOptions.
type BuildOptions struct {
	Stdout         io.Writer
	Stderr         io.Writer
	ProgressWriter io.Writer
}

// Build compiles the configured Go CLI for all configured targets into `dist/`.
// By default, subprocess and progress output are suppressed.
func Build(cfg *config.Config) error {
	return BuildWithOptions(cfg, BuildOptions{})
}

// BuildWithOptions compiles the configured Go CLI for all configured targets into `dist/`.
func BuildWithOptions(cfg *config.Config, opts BuildOptions) error {
	layout := paths.NewLayout(cfg.EffectiveWorkspaceDir())
	if err := os.MkdirAll(layout.DistDir, 0755); err != nil {
		return fmt.Errorf("create dist directory %s: %w", layout.DistDir, err)
	}

	for _, target := range cfg.Targets {
		if err := buildTarget(cfg, layout, target, opts); err != nil {
			return fmt.Errorf("failed to build %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func buildTarget(cfg *config.Config, layout paths.Layout, target config.Target, opts BuildOptions) error {
	outputDir := filepath.Join(layout.DistDir, target.OS, target.Arch)
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
	buildCmd.Stdout = commandOutputWriter(opts.Stdout)
	buildCmd.Stderr = commandOutputWriter(opts.Stderr)

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("run go build for %s: %w", outputPath, err)
	}

	if target.OS != "windows" {
		if err := os.Chmod(outputPath, 0755); err != nil {
			return fmt.Errorf("chmod built binary %s: %w", outputPath, err)
		}
	}

	if opts.ProgressWriter != nil {
		_, _ = fmt.Fprintf(opts.ProgressWriter, "Built: %s\n", outputPath)
	}
	return nil
}

func commandOutputWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
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
