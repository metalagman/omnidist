package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

const appkitVersionPackage = "github.com/metalagman/appkit/version"

type buildMetadata struct {
	version   string
	metadata  string
	gitCommit string
	buildDate string
}

func Build(cfg *config.Config) error {
	if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
		return err
	}

	includeAppkitVersion := toolImportsPackage(cfg.Tool.Main, appkitVersionPackage)
	metadata := resolveBuildMetadata(cfg)

	for _, target := range cfg.Targets {
		if err := buildTarget(cfg, target, includeAppkitVersion, metadata); err != nil {
			return fmt.Errorf("failed to build %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func buildTarget(cfg *config.Config, target config.Target, includeAppkitVersion bool, metadata buildMetadata) error {
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
	ldflags := cfg.Build.Ldflags
	if includeAppkitVersion {
		ldflags = mergeLDFlags(ldflags, appkitVersionLDFlags(metadata))
	}
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

func resolveBuildMetadata(cfg *config.Config) buildMetadata {
	versionValue := "dev"
	if version, err := shared.ResolveVersion(cfg, false); err == nil {
		versionValue = version
	}

	versionValue, metadataValue := splitVersionMetadata(versionValue)
	commit := resolveGitCommit()
	buildDate := time.Now().UTC().Format(time.RFC3339)

	return buildMetadata{
		version:   versionValue,
		metadata:  metadataValue,
		gitCommit: commit,
		buildDate: buildDate,
	}
}

func splitVersionMetadata(version string) (string, string) {
	v := strings.TrimSpace(version)
	if v == "" {
		return "dev", ""
	}

	base, metadata, hasMetadata := strings.Cut(v, "+")
	if !hasMetadata {
		return base, ""
	}
	return base, metadata
}

func resolveGitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func toolImportsPackage(mainPath string, packagePath string) bool {
	mainPath = strings.TrimSpace(mainPath)
	packagePath = strings.TrimSpace(packagePath)
	if mainPath == "" || packagePath == "" {
		return false
	}

	out, err := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", mainPath).Output()
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == packagePath {
			return true
		}
	}
	return false
}

func appkitVersionLDFlags(metadata buildMetadata) []string {
	versionValue := strings.TrimSpace(metadata.version)
	if versionValue == "" {
		versionValue = "dev"
	}

	return []string{
		fmt.Sprintf("-X %s.version=%s", appkitVersionPackage, versionValue),
		fmt.Sprintf("-X %s.metadata=%s", appkitVersionPackage, strings.TrimSpace(metadata.metadata)),
		fmt.Sprintf("-X %s.gitCommit=%s", appkitVersionPackage, strings.TrimSpace(metadata.gitCommit)),
		fmt.Sprintf("-X %s.buildDate=%s", appkitVersionPackage, strings.TrimSpace(metadata.buildDate)),
	}
}

func mergeLDFlags(base string, extra []string) string {
	parts := make([]string, 0, 1+len(extra))

	trimmedBase := strings.TrimSpace(base)
	if trimmedBase != "" {
		parts = append(parts, trimmedBase)
	}

	for _, flag := range extra {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}
		parts = append(parts, flag)
	}

	return strings.Join(parts, " ")
}
