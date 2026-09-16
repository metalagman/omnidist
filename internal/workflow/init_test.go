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
	writeMainPackage(t, filepath.Join("cmd", "my-tool"))

	if err := Init(paths.ConfigPath, InitOptions{}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cfg, err := config.LoadWithProfile(paths.ConfigPath, config.DefaultProfileName)
	if err != nil {
		t.Fatalf("config.LoadWithProfile(default) error = %v", err)
	}
	slug := "my-tool"
	wantNPMPackage := "@" + slug + "/" + slug
	wantUVPackage := slug
	if got := cfg.Tool.Name; got != slug {
		t.Fatalf("tool name = %q, want %q", got, slug)
	}
	if got := cfg.Tool.Main; got != "./cmd/my-tool" {
		t.Fatalf("tool main = %q, want %q", got, "./cmd/my-tool")
	}
	if got := cfg.Distributions["npm"].Package; got != wantNPMPackage {
		t.Fatalf("npm package = %q, want %q", got, wantNPMPackage)
	}
	if got := cfg.Distributions["uv"].Package; got != wantUVPackage {
		t.Fatalf("uv package = %q, want %q", got, wantUVPackage)
	}
	if got := cfg.Distributions["gem"].Package; got != slug {
		t.Fatalf("gem package = %q, want %q", got, slug)
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

func TestInitRefusesOverwriteUnlessForced(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeMainPackage(t, filepath.Join("cmd", "my-tool"))
	if err := os.MkdirAll(filepath.Dir(paths.ConfigPath), 0755); err != nil {
		t.Fatal(err)
	}
	const sentinel = "do not replace\n"
	if err := os.WriteFile(paths.ConfigPath, []byte(sentinel), 0644); err != nil {
		t.Fatal(err)
	}

	err := Init(paths.ConfigPath, InitOptions{})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("Init(existing) error = %v, want actionable --force error", err)
	}
	data, readErr := os.ReadFile(paths.ConfigPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got := string(data); got != sentinel {
		t.Fatalf("existing config changed: got %q", got)
	}

	if err := Init(paths.ConfigPath, InitOptions{Force: true}); err != nil {
		t.Fatalf("Init(force) error = %v", err)
	}
	cfg, err := config.LoadWithProfile(paths.ConfigPath, config.DefaultProfileName)
	if err != nil {
		t.Fatalf("load forced config: %v", err)
	}
	if cfg.Tool.Name != "my-tool" {
		t.Fatalf("forced config tool name = %q", cfg.Tool.Name)
	}
}

func TestInitDiscoveryRequiresUnambiguousMain(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		err := Init(paths.ConfigPath, InitOptions{})
		if err == nil || !strings.Contains(err.Error(), "--name and --main") {
			t.Fatalf("Init(no main) error = %v", err)
		}
		if _, statErr := os.Stat(paths.ConfigPath); !os.IsNotExist(statErr) {
			t.Fatalf("config written after discovery failure: %v", statErr)
		}
	})

	t.Run("ambiguous", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		writeMainPackage(t, filepath.Join("cmd", "alpha"))
		writeMainPackage(t, filepath.Join("cmd", "beta"))
		err := Init(paths.ConfigPath, InitOptions{})
		if err == nil || !strings.Contains(err.Error(), "multiple Go main packages") {
			t.Fatalf("Init(ambiguous) error = %v", err)
		}
		if _, statErr := os.Stat(paths.ConfigPath); !os.IsNotExist(statErr) {
			t.Fatalf("config written after discovery failure: %v", statErr)
		}
	})

	t.Run("explicit override", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := Init(paths.ConfigPath, InitOptions{ToolName: "Acme CLI", ToolMain: "tools/acme"}); err != nil {
			t.Fatalf("Init(explicit) error = %v", err)
		}
		cfg, err := config.LoadWithProfile(paths.ConfigPath, config.DefaultProfileName)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Tool.Name != "acme-cli" || cfg.Tool.Main != "./tools/acme" {
			t.Fatalf("explicit identity = %#v", cfg.Tool)
		}
	})
}

func writeMainPackage(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
