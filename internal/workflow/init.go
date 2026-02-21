package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func Init(configPath string) error {
	cfg := config.DefaultConfig()

	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if err := CreateNPMStructure(cfg); err != nil {
		return err
	}

	if err := CreateUVStructure(cfg); err != nil {
		return err
	}

	if err := EnsureWorkspaceGitignore(filepath.Join(paths.WorkspaceDir, ".gitignore")); err != nil {
		return err
	}

	return nil
}

func CreateNPMStructure(cfg *config.Config) error {
	dist, ok := cfg.Distributions["npm"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	baseDir := paths.NPMDir

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	metaDir := filepath.Join(baseDir, dist.Package)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		pkgDir := fmt.Sprintf("%s-%s-%s", dist.Package, config.MapGoOSToNPM(target.OS), config.MapGoArchToNPM(target.Arch))
		if target.Variant != "" {
			pkgDir = fmt.Sprintf("%s-%s", pkgDir, target.Variant)
		}
		if err := os.MkdirAll(filepath.Join(baseDir, pkgDir, "bin"), 0755); err != nil {
			return err
		}
	}

	return nil
}

func CreateUVStructure(cfg *config.Config) error {
	dist, ok := cfg.Distributions["uv"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	if err := os.MkdirAll(paths.UVDistDir, 0755); err != nil {
		return err
	}

	return nil
}

func EnsureWorkspaceGitignore(path string) error {
	required := []string{
		"dist/",
		"npm/",
		"uv/",
	}

	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	var missing []string
	for _, line := range required {
		if !strings.Contains(existing, line) {
			missing = append(missing, line)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	var builder strings.Builder
	builder.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		builder.WriteString("\n")
	}
	if existing != "" {
		builder.WriteString("\n")
	}
	builder.WriteString("# omnidist generated artifacts\n")
	for _, line := range missing {
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return os.WriteFile(path, []byte(builder.String()), 0644)
}
