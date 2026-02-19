package npm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteShimResolvesScopedPlatformPackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	shimPath := filepath.Join(dir, "omnidist.js")

	if err := writeShim(shimPath, "omnidist", "@omnidist/omnidist"); err != nil {
		t.Fatalf("writeShim() error = %v", err)
	}

	data, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", shimPath, err)
	}

	shim := string(data)
	if !strings.Contains(shim, "const platformPkgName = '@omnidist/omnidist-' + platformKey;") {
		t.Fatalf("shim does not use scoped platform package name: %q", shim)
	}
	if !strings.Contains(shim, "require.resolve(platformPkgName + '/package.json', { paths: [__dirname] });") {
		t.Fatalf("shim does not resolve package via require.resolve: %q", shim)
	}
	if strings.Contains(shim, "const metaParts =") {
		t.Fatalf("shim still contains old sibling directory logic: %q", shim)
	}
}
