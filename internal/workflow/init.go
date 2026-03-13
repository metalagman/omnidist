package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"gopkg.in/yaml.v3"
)

// Init writes a default config and creates initial staging directories.
func Init(configPath string) error {
	cfg := config.DefaultConfig()

	if err := saveProfilesConfig(cfg, configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	profileCfg, err := config.LoadWithProfile(configPath, config.DefaultProfileName)
	if err != nil {
		return fmt.Errorf("load generated profile config: %w", err)
	}

	if err := CreateNPMStructure(profileCfg); err != nil {
		return err
	}

	if err := CreateUVStructure(profileCfg); err != nil {
		return err
	}

	return nil
}

// CreateNPMStructure creates the npm workspace directories for configured targets.
func CreateNPMStructure(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	dist, ok := cfg.Distributions["npm"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	layout := paths.NewLayout(cfg.EffectiveWorkspaceDir())
	baseDir := layout.NPMDir

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("create npm base directory %s: %w", baseDir, err)
	}

	metaDir := filepath.Join(baseDir, dist.Package)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("create npm meta directory %s: %w", metaDir, err)
	}

	for _, target := range cfg.Targets {
		pkgDir := fmt.Sprintf("%s-%s-%s", dist.Package, config.MapGoOSToNPM(target.OS), config.MapGoArchToNPM(target.Arch))
		if target.Variant != "" {
			pkgDir = fmt.Sprintf("%s-%s", pkgDir, target.Variant)
		}
		if err := os.MkdirAll(filepath.Join(baseDir, pkgDir, "bin"), 0755); err != nil {
			return fmt.Errorf("create npm platform bin directory for %s: %w", pkgDir, err)
		}
	}

	return nil
}

// CreateUVStructure creates the uv staging directory when uv distribution is configured.
func CreateUVStructure(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	dist, ok := cfg.Distributions["uv"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	layout := paths.NewLayout(cfg.EffectiveWorkspaceDir())
	if err := os.MkdirAll(layout.UVDistDir, 0755); err != nil {
		return fmt.Errorf("create uv dist directory %s: %w", layout.UVDistDir, err)
	}

	return nil
}

// EnsureWorkspaceGitignore creates `.gitignore` for workspace layout control when missing.
func EnsureWorkspaceGitignore(path string) error {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("gitignore path %s is a directory", path)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat gitignore %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create gitignore directory %s: %w", filepath.Dir(path), err)
	}

	content := "*\n!.gitignore\n!omnidist.yaml\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write gitignore %s: %w", path, err)
	}
	return nil
}

func saveProfilesConfig(cfg *config.Config, configPath string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	var file struct {
		Profiles map[string]config.Config `yaml:"profiles"`
	}
	file.Profiles = map[string]config.Config{
		config.DefaultProfileName: *cfg,
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal profile config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config file %s: %w", configPath, err)
	}
	return nil
}
