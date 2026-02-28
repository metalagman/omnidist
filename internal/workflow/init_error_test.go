package workflow

import (
	"os"
	"path/filepath"
	"strings"
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

	// 1. Fresh file
	if err := EnsureWorkspaceGitignore(path); err != nil {
		t.Fatalf("EnsureWorkspaceGitignore(fresh) error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "dist/") {
		t.Fatalf("missing dist/")
	}

	// 2. Existing file without newline
	os.WriteFile(path, []byte("node_modules/"), 0644)
	if err := EnsureWorkspaceGitignore(path); err != nil {
		t.Fatalf("EnsureWorkspaceGitignore(no newline) error = %v", err)
	}
	data, _ = os.ReadFile(path)
	if !strings.HasPrefix(string(data), "node_modules/\n\n# omnidist") {
		t.Fatalf("unexpected content after no-newline setup:\n%s", string(data))
	}

	// 3. Existing file WITH newline
	os.WriteFile(path, []byte("node_modules/\n"), 0644)
	if err := EnsureWorkspaceGitignore(path); err != nil {
		t.Fatalf("EnsureWorkspaceGitignore(with newline) error = %v", err)
	}
	data, _ = os.ReadFile(path)
	if !strings.HasPrefix(string(data), "node_modules/\n\n# omnidist") {
		t.Fatalf("unexpected content after with-newline setup:\n%s", string(data))
	}

	// 4. Already has everything
	if err := EnsureWorkspaceGitignore(path); err != nil {
		t.Fatalf("EnsureWorkspaceGitignore(already has) error = %v", err)
	}
	dataAfter, _ := os.ReadFile(path)
	if string(data) != string(dataAfter) {
		t.Fatalf("gitignore changed unexpectedly when already complete")
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
