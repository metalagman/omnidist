package shared

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestNormalizeHelpers(t *testing.T) {
	t.Run("NormalizePythonDistributionName", func(t *testing.T) {
		got := NormalizePythonDistributionName("@scope/tool-name.v2")
		if got != "scope_tool_name_v2" {
			t.Fatalf("NormalizePythonDistributionName() = %q, want %q", got, "scope_tool_name_v2")
		}
	})

	t.Run("NormalizeGoTarget", func(t *testing.T) {
		goOS, goArch := NormalizeGoTarget(config.Target{OS: " linux ", Arch: " arm64 "})
		if goOS != "linux" || goArch != "arm64" {
			t.Fatalf("NormalizeGoTarget() = (%q,%q), want (%q,%q)", goOS, goArch, "linux", "arm64")
		}
	})

	t.Run("BinaryName", func(t *testing.T) {
		if got := BinaryName("tool", "windows"); got != "tool.exe" {
			t.Fatalf("BinaryName(windows) = %q, want %q", got, "tool.exe")
		}
		if got := BinaryName("tool", "linux"); got != "tool" {
			t.Fatalf("BinaryName(linux) = %q, want %q", got, "tool")
		}
	})
}

func TestWheelFilenameAndBinaryPath(t *testing.T) {
	target := config.Target{OS: "linux", Arch: "amd64"}
	got, err := WheelFilename("@scope/tool-name", "1.2.3", target, DefaultUVLinuxTag)
	if err != nil {
		t.Fatalf("WheelFilename() error = %v", err)
	}
	if got != "scope_tool_name-1.2.3-py3-none-manylinux2014_x86_64.whl" {
		t.Fatalf("WheelFilename() = %q, want expected wheel name", got)
	}

	binaryPath := WheelBinaryPath("@scope/tool-name", "tool", "windows")
	if binaryPath != "scope_tool_name/bin/tool.exe" {
		t.Fatalf("WheelBinaryPath() = %q, want %q", binaryPath, "scope_tool_name/bin/tool.exe")
	}
}

func TestResolveVersionValidation(t *testing.T) {
	t.Run("nil_config", func(t *testing.T) {
		_, err := ResolveVersion(nil, false)
		if err == nil || !strings.Contains(err.Error(), "config is nil") {
			t.Fatalf("ResolveVersion(nil) error = %v, want config nil error", err)
		}
	})

	t.Run("unknown_source", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionConfig{Source: "mystery"}}
		_, err := ResolveVersion(cfg, false)
		if err == nil || !strings.Contains(err.Error(), "unknown version source") {
			t.Fatalf("ResolveVersion(unknown) error = %v, want unknown source error", err)
		}
	})

	t.Run("env_empty", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}
		t.Setenv(EnvVersionName, "  ")
		_, err := ResolveVersion(cfg, false)
		if err == nil || !strings.Contains(err.Error(), "empty version from source") {
			t.Fatalf("ResolveVersion(env empty) error = %v, want empty version error", err)
		}
	})
}

func TestResolveVersionGitTag(t *testing.T) {
	repo := initGitRepoWithCommit(t)
	t.Chdir(repo)
	runGit(t, repo, "tag", "v1.2.3")

	cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}

	release, err := ResolveVersion(cfg, false)
	if err != nil {
		t.Fatalf("ResolveVersion(release) error = %v", err)
	}
	if release != "1.2.3" {
		t.Fatalf("ResolveVersion(release) = %q, want %q", release, "1.2.3")
	}

	dev, err := ResolveVersion(cfg, true)
	if err != nil {
		t.Fatalf("ResolveVersion(dev) error = %v", err)
	}
	matched, err := regexp.MatchString(`^1\.2\.3-dev\.0\.g[0-9a-f]+$`, dev)
	if err != nil {
		t.Fatalf("regexp.MatchString() error = %v", err)
	}
	if !matched {
		t.Fatalf("ResolveVersion(dev) = %q, want semver dev format", dev)
	}
}

func TestResolveReleaseVersionGitTagRejectsNonSemverTag(t *testing.T) {
	repo := initGitRepoWithCommit(t)
	t.Chdir(repo)
	runGit(t, repo, "tag", "release-1")

	cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}
	_, err := ResolveReleaseVersion(cfg)
	if err == nil || !strings.Contains(err.Error(), "not exact semver") {
		t.Fatalf("ResolveReleaseVersion(non-semver-tag) error = %v, want semver validation error", err)
	}
}

func TestResolveVersionGitTagDevRejectsNonSemverTag(t *testing.T) {
	repo := initGitRepoWithCommit(t)
	t.Chdir(repo)
	runGit(t, repo, "tag", "release-1")

	cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}
	_, err := ResolveVersion(cfg, true)
	if err == nil || !strings.Contains(err.Error(), "not exact semver") {
		t.Fatalf("ResolveVersion(dev, non-semver-tag) error = %v, want semver validation error", err)
	}
}

func TestResolveVersionGitTagDevNoTags(t *testing.T) {
	repo := initGitRepoWithCommit(t)
	t.Chdir(repo)
	// No tags

	cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}
	dev, err := ResolveVersion(cfg, true)
	if err != nil {
		t.Fatalf("ResolveVersion(dev, no tags) error = %v", err)
	}
	// Should return just the hash (at least 7 chars)
	if len(dev) < 7 {
		t.Fatalf("ResolveVersion(dev, no tags) = %q, want git hash", dev)
	}
}

func TestResolveVersionGitTagNoRepo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	// Not a git repo

	cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}
	_, err := ResolveVersion(cfg, false)
	if err == nil {
		t.Fatalf("ResolveVersion(git-tag, no repo) error = nil, want error")
	}
}

func TestResolveReleaseVersionGitTag(t *testing.T) {
	t.Run("exact_tag", func(t *testing.T) {
		repo := initGitRepoWithCommit(t)
		t.Chdir(repo)
		runGit(t, repo, "tag", "v2.3.4")

		cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}
		got, err := ResolveReleaseVersion(cfg)
		if err != nil {
			t.Fatalf("ResolveReleaseVersion() error = %v", err)
		}
		if got != "2.3.4" {
			t.Fatalf("ResolveReleaseVersion() = %q, want %q", got, "2.3.4")
		}
	})

	t.Run("not_at_exact_tag", func(t *testing.T) {
		repo := initGitRepoWithCommit(t)
		t.Chdir(repo)
		runGit(t, repo, "tag", "v2.3.4")
		if err := os.WriteFile(filepath.Join(repo, "next.txt"), []byte("next"), 0644); err != nil {
			t.Fatalf("os.WriteFile(next.txt) error = %v", err)
		}
		runGit(t, repo, "add", "next.txt")
		runGit(t, repo, "commit", "-m", "next")

		cfg := &config.Config{Version: config.VersionConfig{Source: "git-tag"}}
		_, err := ResolveReleaseVersion(cfg)
		if err == nil || !strings.Contains(err.Error(), "HEAD is not at an exact semver tag") {
			t.Fatalf("ResolveReleaseVersion() error = %v, want exact tag error", err)
		}
	})
}

func TestResolveReleaseVersionUnknownSource(t *testing.T) {
	cfg := &config.Config{Version: config.VersionConfig{Source: "unknown"}}
	_, err := ResolveReleaseVersion(cfg)
	if err == nil || !strings.Contains(err.Error(), "unknown version source") {
		t.Fatalf("ResolveReleaseVersion(unknown) error = %v, want unknown source error", err)
	}
}

func initGitRepoWithCommit(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "tester@example.com")
	runGit(t, repo, "config", "user.name", "tester")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("os.WriteFile(README.md) error = %v", err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "init")

	return repo
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}
