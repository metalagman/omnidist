package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/paths"
)

func TestInitCreatesNPMAndUVStructure(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := Init(paths.ConfigPath); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	requiredPaths := []string{
		paths.ConfigPath,
		filepath.Join(paths.NPMDir, "@omnidist", "omnidist"),
		filepath.Join(paths.NPMDir, "@omnidist", "omnidist-linux-x64", "bin"),
		paths.UVDistDir,
		filepath.Join(paths.WorkspaceDir, ".gitignore"),
	}

	for _, p := range requiredPaths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist, stat error: %v", p, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(paths.WorkspaceDir, ".gitignore"))
	if err != nil {
		t.Fatalf("os.ReadFile(workspace .gitignore) error = %v", err)
	}
	content := string(data)
	for _, required := range []string{"dist/", "npm/", "uv/"} {
		if !strings.Contains(content, required) {
			t.Fatalf("workspace .gitignore missing %q, got:\n%s", required, content)
		}
	}

	configData, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		t.Fatalf("os.ReadFile(config) error = %v", err)
	}
	configContent := string(configData)
	if !strings.Contains(configContent, "include-readme: true") {
		t.Fatalf("generated config missing include-readme default, got:\n%s", configContent)
	}

	if _, err := os.Stat(".gitignore"); !os.IsNotExist(err) {
		t.Fatalf("expected root .gitignore to be untouched in fresh repo, got err=%v", err)
	}
}
