package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestInitCreatesNPMAndUVStructure(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := Init(paths.ConfigPath); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cfg, err := config.LoadWithProfile(paths.ConfigPath, config.DefaultProfileName)
	if err != nil {
		t.Fatalf("config.LoadWithProfile(default) error = %v", err)
	}
	slug := slugifyName(filepath.Base(dir))
	wantNPMPackage := "@" + slug + "/" + slug
	wantUVPackage := slug
	if got := cfg.Distributions["npm"].Package; got != wantNPMPackage {
		t.Fatalf("npm package = %q, want %q", got, wantNPMPackage)
	}
	if got := cfg.Distributions["uv"].Package; got != wantUVPackage {
		t.Fatalf("uv package = %q, want %q", got, wantUVPackage)
	}

	requiredPaths := []string{
		paths.ConfigPath,
		filepath.Join(paths.WorkspaceDir, "default", "npm", wantNPMPackage),
		filepath.Join(paths.WorkspaceDir, "default", "npm", wantNPMPackage+"-linux-x64", "bin"),
		filepath.Join(paths.WorkspaceDir, "default", "uv", "dist"),
	}

	for _, p := range requiredPaths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist, stat error: %v", p, err)
		}
	}

	configData, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		t.Fatalf("os.ReadFile(config) error = %v", err)
	}
	configContent := string(configData)
	if !strings.Contains(configContent, "profiles:") {
		t.Fatalf("generated config missing profiles root, got:\n%s", configContent)
	}
	if !strings.Contains(configContent, "include-readme: true") {
		t.Fatalf("generated config missing include-readme default, got:\n%s", configContent)
	}
	if strings.Contains(configContent, "\ntool:\n") {
		t.Fatalf("generated config should not use legacy top-level fields, got:\n%s", configContent)
	}
	if !cfg.IsProfilesMode() {
		t.Fatalf("cfg.IsProfilesMode() = false, want true")
	}

	gotWorkspace := cfg.EffectiveWorkspaceDir()
	if runtime.GOOS == "windows" {
		gotWorkspace = strings.ReplaceAll(gotWorkspace, "\\", "/")
	}
	if gotWorkspace != ".omnidist/default" {
		t.Fatalf("cfg.EffectiveWorkspaceDir() = %q, want %q", gotWorkspace, ".omnidist/default")
	}

	if _, err := os.Stat(".gitignore"); !os.IsNotExist(err) {
		t.Fatalf("expected root .gitignore to be untouched in fresh repo, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.WorkspaceDir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("expected workspace .gitignore to be absent after init, got err=%v", err)
	}
}
