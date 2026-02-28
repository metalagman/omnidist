package npm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestStageErrors(t *testing.T) {
	t.Run("nil_config", func(t *testing.T) {
		err := Stage(nil, StageOptions{})
		if err == nil || !strings.Contains(err.Error(), "config is nil") {
			t.Fatalf("Stage(nil) error = %v, want config nil error", err)
		}
	})

	t.Run("mkdir_meta_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		if err := shared.WriteBuildVersion("1.0.0"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}
		cfg := testConfig()
		if err := createDistArtifacts(cfg); err != nil {
			t.Fatalf("createDistArtifacts() error = %v", err)
		}
		npmDist, _ := npmDistribution(cfg)
		
		// Create a file where meta directory should be
		metaDir := filepath.Join(paths.NPMDir, npmDist.Package)
		if err := os.MkdirAll(filepath.Dir(metaDir), 0755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(metaDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", metaDir, err)
		}

		err := Stage(cfg, StageOptions{})
		if err == nil || !strings.Contains(err.Error(), "failed to stage meta package") {
			t.Fatalf("Stage(mkdir meta fail) error = %v, want meta stage error", err)
		}
	})

	t.Run("stage_platform_package_fail_missing_binary", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		if err := shared.WriteBuildVersion("1.0.0"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}
		cfg := testConfig()
		// Do not create dist artifacts

		err := Stage(cfg, StageOptions{})
		if err == nil || !strings.Contains(err.Error(), "failed to stage") {
			t.Fatalf("Stage(missing binary) error = %v, want stage failure", err)
		}
	})
}

func TestCopyFileWithModeErrors(t *testing.T) {
	t.Run("source_missing", func(t *testing.T) {
		err := copyFileWithMode("missing", "dest", 0644)
		if err == nil {
			t.Fatalf("copyFileWithMode(missing) error = nil, want error")
		}
	})

	t.Run("dest_creation_fail", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		os.WriteFile(src, []byte("data"), 0644)
		
		dest := filepath.Join(dir, "dest")
		os.MkdirAll(dest, 0755) // Create dest as a directory

		err := copyFileWithMode(src, dest, 0644)
		if err == nil {
			t.Fatalf("copyFileWithMode(dest fail) error = nil, want error")
		}
	})
}

func TestEnsureWorkspaceNPMRCErrors(t *testing.T) {
	t.Run("write_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		// Create a directory where .npmrc should be
		if err := os.MkdirAll(paths.NPMRCPath, 0755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		_, err := ensureWorkspaceNPMRC("https://registry.npmjs.org")
		if err == nil || !strings.Contains(err.Error(), "open .omnidist/.npmrc") {
			t.Fatalf("ensureWorkspaceNPMRC(write fail) error = %v, want open error", err)
		}
	})
}

func TestPublishErrors(t *testing.T) {
	t.Run("nil_config", func(t *testing.T) {
		err := Publish(nil, PublishOptions{})
		if err == nil || !strings.Contains(err.Error(), "config is nil") {
			t.Fatalf("Publish(nil) error = %v, want config nil error", err)
		}
	})

	t.Run("missing_npm_distribution", func(t *testing.T) {
		cfg := &config.Config{
			Distributions: map[string]config.DistributionConfig{},
		}
		err := Publish(cfg, PublishOptions{})
		if err == nil || !strings.Contains(err.Error(), "missing required distribution: npm") {
			t.Fatalf("Publish(no npm) error = %v, want missing npm error", err)
		}
	})

	t.Run("working_dir_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.2.3")
		if err := shared.WriteBuildVersion("1.2.3"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}

		cfg := testConfig()
		err := Publish(cfg, PublishOptions{})
		if err == nil || !strings.Contains(err.Error(), "resolve package working directory") {
			t.Fatalf("Publish(missing working dir) error = %v, want working dir error", err)
		}
	})
}

func TestResolveRegistry(t *testing.T) {
	tests := []struct {
		name     string
		def      string
		override string
		want     string
	}{
		{"both empty", "", "", "https://registry.npmjs.org"},
		{"override set", "def", "over", "over"},
		{"default set", "def", "", "def"},
	}
	for _, tt := range tests {
		if got := resolveRegistry(tt.def, tt.override); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPlatformPackageName(t *testing.T) {
	tests := []struct {
		name    string
		metaPkg string
		target  config.Target
		want    string
	}{
		{"scoped", "@scope/tool", config.Target{OS: "linux", Arch: "amd64"}, "@scope/tool-linux-x64"},
		{"unscoped", "tool", config.Target{OS: "linux", Arch: "amd64"}, "tool-linux-x64"},
	}
	for _, tt := range tests {
		if got := platformPackageName(tt.metaPkg, tt.target); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCheckAuthErrors(t *testing.T) {
	t.Run("resolvePublishToken_fail", func(t *testing.T) {
		t.Setenv("NPM_PUBLISH_TOKEN", "")
		cfg := testConfig()
		err := CheckAuth(cfg, "", false)
		if err == nil || !strings.Contains(err.Error(), "NPM_PUBLISH_TOKEN") {
			t.Fatalf("CheckAuth() error = %v, want token error", err)
		}
	})

	t.Run("ensureWorkspaceNPMRC_fail", func(t *testing.T) {
		t.Setenv("NPM_PUBLISH_TOKEN", "secret")
		dir := t.TempDir()
		t.Chdir(dir)
		// Make WorkspaceDir a file to fail MkdirAll
		os.WriteFile(paths.WorkspaceDir, []byte("file"), 0644)
		
		cfg := testConfig()
		err := CheckAuth(cfg, "", false)
		if err == nil || !strings.Contains(err.Error(), "prepare npmrc") {
			t.Fatalf("CheckAuth() error = %v, want npmrc error", err)
		}
	})
}

func TestResolveNPMStageVersionInvalid(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := shared.WriteBuildVersion("invalid semver"); err != nil {
		t.Fatalf("shared.WriteBuildVersion() error = %v", err)
	}

	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}
	_, err := resolveNPMStageVersion(cfg, false)
	if err == nil || !strings.Contains(err.Error(), "invalid npm version") {
		t.Fatalf("resolveNPMStageVersion(invalid) error = %v, want validation error", err)
	}
}

func TestWriteProgressfNoop(t *testing.T) {
	writeProgressf(nil, "format %s", "args")
	// Should not panic
}
