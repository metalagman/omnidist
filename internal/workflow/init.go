package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
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

	return nil
}

func CreateNPMStructure(cfg *config.Config) error {
	dist, ok := cfg.Distributions["npm"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	baseDir := "npm"

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	metaDir := filepath.Join(baseDir, dist.Package)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		pkgDir := fmt.Sprintf("%s-%s-%s", dist.Package, target.OS, config.MapArchToNPM(target.Arch))
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

	if err := os.MkdirAll(filepath.Join("uv", "dist"), 0755); err != nil {
		return err
	}

	return nil
}
