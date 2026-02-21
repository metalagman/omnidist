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
		".gitignore",
	}

	for _, p := range requiredPaths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist, stat error: %v", p, err)
		}
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("os.ReadFile(.gitignore) error = %v", err)
	}
	content := string(data)
	for _, required := range []string{"/.omnidist/", "/omnidist/"} {
		if !strings.Contains(content, required) {
			t.Fatalf(".gitignore missing %q, got:\n%s", required, content)
		}
	}
}
