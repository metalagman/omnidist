package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesNPMAndUVStructure(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := Init("omnidist.yaml"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	paths := []string{
		"omnidist.yaml",
		filepath.Join("npm", "@omnidist", "omnidist"),
		filepath.Join("npm", "@omnidist", "omnidist-linux-x64", "bin"),
		filepath.Join("uv", "dist"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist, stat error: %v", p, err)
		}
	}
}
