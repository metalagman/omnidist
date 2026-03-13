package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestCreateNPMStructureSkipsWhenMissing(t *testing.T) {
	cfg := &config.Config{
		Distributions: map[string]config.DistributionConfig{},
	}
	if err := CreateNPMStructure(cfg); err != nil {
		t.Fatalf("CreateNPMStructure() with no npm dist error = %v", err)
	}

	cfg = &config.Config{
		Distributions: map[string]config.DistributionConfig{
			"npm": {Package: "  "},
		},
	}
	if err := CreateNPMStructure(cfg); err != nil {
		t.Fatalf("CreateNPMStructure() with empty package error = %v", err)
	}
}

func TestCreateUVStructureSkipsWhenMissing(t *testing.T) {
	cfg := &config.Config{
		Distributions: map[string]config.DistributionConfig{},
	}
	if err := CreateUVStructure(cfg); err != nil {
		t.Fatalf("CreateUVStructure() with no uv dist error = %v", err)
	}

	cfg = &config.Config{
		Distributions: map[string]config.DistributionConfig{
			"uv": {Package: "  "},
		},
	}
	if err := CreateUVStructure(cfg); err != nil {
		t.Fatalf("CreateUVStructure() with empty package error = %v", err)
	}
}

func TestEnsureWorkspaceGitignoreScenarios(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	if err := EnsureWorkspaceGitignore(path); err != nil {
		t.Fatalf("EnsureWorkspaceGitignore(fresh) error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if got := string(data); got != "*\n!.gitignore\n!omnidist.yaml\n" {
		t.Fatalf("fresh workspace gitignore content = %q", got)
	}

	const existing = "node_modules/\n"
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	if err := EnsureWorkspaceGitignore(path); err != nil {
		t.Fatalf("EnsureWorkspaceGitignore(existing) error = %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if got := string(data); got != existing {
		t.Fatalf("existing workspace gitignore should remain unchanged, got %q", got)
	}
}

func TestInitErrorMkdirAll(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Create a file where .omnidist should be
	os.WriteFile(paths.WorkspaceDir, []byte("i am a file"), 0644)

	err := Init(paths.ConfigPath)
	if err == nil {
		t.Fatalf("Init() error = nil, want error (mkdir fail)")
	}
}
