package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestResolveVersionFileSource(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		want := "v2.0.0"
		if err := os.WriteFile("VERSION", []byte(want), 0644); err != nil {
			t.Fatalf("os.WriteFile(VERSION) error = %v", err)
		}

		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		got, err := ResolveVersion(cfg, false)
		if err != nil {
			t.Fatalf("ResolveVersion(file) error = %v", err)
		}
		if got != "v2.0.0" {
			t.Fatalf("ResolveVersion(file) = %q, want %q", got, "v2.0.0")
		}
	})

	t.Run("missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		_, err := ResolveVersion(cfg, false)
		if err == nil || !strings.Contains(err.Error(), "read version file") {
			t.Fatalf("ResolveVersion(file missing) error = %v, want missing file error", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		if err := os.WriteFile("VERSION", []byte("  \n"), 0644); err != nil {
			t.Fatalf("os.WriteFile(VERSION) error = %v", err)
		}

		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		_, err := ResolveVersion(cfg, false)
		if err == nil || !strings.Contains(err.Error(), "empty version from source") {
			t.Fatalf("ResolveVersion(file empty) error = %v, want empty version error", err)
		}
	})

	t.Run("custom_relative_path", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)
		if err := os.MkdirAll("versions", 0755); err != nil {
			t.Fatalf("os.MkdirAll(versions) error = %v", err)
		}
		if err := os.WriteFile("versions/release.txt", []byte("3.4.5\n"), 0644); err != nil {
			t.Fatalf("os.WriteFile(custom version file) error = %v", err)
		}

		cfg := &config.Config{Version: config.VersionConfig{
			Source: "file",
			File:   "versions/release.txt",
		}}
		got, err := ResolveVersion(cfg, false)
		if err != nil {
			t.Fatalf("ResolveVersion(file custom path) error = %v", err)
		}
		if got != "3.4.5" {
			t.Fatalf("ResolveVersion(file custom path) = %q, want %q", got, "3.4.5")
		}
	})

	t.Run("custom_absolute_path", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		absPath := filepath.Join(tmpDir, "version.txt")
		if err := os.WriteFile(absPath, []byte("4.5.6"), 0644); err != nil {
			t.Fatalf("os.WriteFile(absolute version file) error = %v", err)
		}

		cfg := &config.Config{Version: config.VersionConfig{
			Source: "file",
			File:   absPath,
		}}
		got, err := ResolveVersion(cfg, false)
		if err != nil {
			t.Fatalf("ResolveVersion(file absolute path) error = %v", err)
		}
		if got != "4.5.6" {
			t.Fatalf("ResolveVersion(file absolute path) = %q, want %q", got, "4.5.6")
		}
	})
}

func TestResolveVersionFixedSource(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionConfig{
			Source: "fixed",
			Fixed:  " 1.2.3 ",
		}}
		got, err := ResolveVersion(cfg, false)
		if err != nil {
			t.Fatalf("ResolveVersion(fixed) error = %v", err)
		}
		if got != "1.2.3" {
			t.Fatalf("ResolveVersion(fixed) = %q, want %q", got, "1.2.3")
		}
	})

	t.Run("dev_passthrough", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionConfig{
			Source: "fixed",
			Fixed:  "1.2.3",
		}}
		got, err := ResolveVersion(cfg, true)
		if err != nil {
			t.Fatalf("ResolveVersion(fixed, dev) error = %v", err)
		}
		if got != "1.2.3" {
			t.Fatalf("ResolveVersion(fixed, dev) = %q, want %q", got, "1.2.3")
		}
	})

	t.Run("empty", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionConfig{
			Source: "fixed",
			Fixed:  " ",
		}}
		_, err := ResolveVersion(cfg, false)
		if err == nil || !strings.Contains(err.Error(), "empty version from source") {
			t.Fatalf("ResolveVersion(fixed empty) error = %v, want empty version error", err)
		}
	})
}

func TestResolveReleaseVersionFileSource(t *testing.T) {
	t.Run("valid_exact_semver", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		want := "1.2.3"
		if err := os.WriteFile("VERSION", []byte(want), 0644); err != nil {
			t.Fatalf("os.WriteFile(VERSION) error = %v", err)
		}

		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		got, err := ResolveReleaseVersion(cfg)
		if err != nil {
			t.Fatalf("ResolveReleaseVersion(file) error = %v", err)
		}
		if got != want {
			t.Fatalf("ResolveReleaseVersion(file) = %q, want %q", got, want)
		}
	})

	t.Run("not_exact_semver", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		if err := os.WriteFile("VERSION", []byte("1.2.3-beta"), 0644); err != nil {
			t.Fatalf("os.WriteFile(VERSION) error = %v", err)
		}

		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		_, err := ResolveReleaseVersion(cfg)
		if err == nil || !strings.Contains(err.Error(), "is not exact semver") {
			t.Fatalf("ResolveReleaseVersion(file non-semver) error = %v, want semver validation error", err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		_, err := ResolveReleaseVersion(cfg)
		if err == nil || !strings.Contains(err.Error(), "read version file") {
			t.Fatalf("ResolveReleaseVersion(file missing) error = %v, want missing file error", err)
		}
	})
}

func TestResolveReleaseVersionEmptyEnv(t *testing.T) {
	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}
	t.Setenv(EnvVersionName, "  ")
	_, err := ResolveReleaseVersion(cfg)
	if err == nil || !strings.Contains(err.Error(), "empty version from source") {
		t.Fatalf("ResolveReleaseVersion(env empty) error = %v, want empty version error", err)
	}
}

func TestWriteBuildVersionErrors(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		err := WriteBuildVersion("  ")
		if err == nil || !strings.Contains(err.Error(), "version is empty") {
			t.Fatalf("WriteBuildVersion(empty) error = %v, want empty error", err)
		}
	})

	t.Run("mkdir_fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)
		// Create a file where paths.DistDir should be
		if err := os.WriteFile(paths.WorkspaceDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", paths.WorkspaceDir, err)
		}
		err := WriteBuildVersion("1.0.0")
		if err == nil || !strings.Contains(err.Error(), "create dist directory") {
			t.Fatalf("WriteBuildVersion(mkdir fail) error = %v, want mkdir error", err)
		}
	})

	t.Run("write_fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)
		if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.DistDir, err)
		}
		// Create a directory where paths.DistVersionPath should be
		if err := os.MkdirAll(paths.DistVersionPath, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.DistVersionPath, err)
		}
		err := WriteBuildVersion("1.0.0")
		if err == nil || !strings.Contains(err.Error(), "write build version file") {
			t.Fatalf("WriteBuildVersion(write fail) error = %v, want write error", err)
		}
	})
}

func TestReadBuildVersionErrors(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)
		if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.DistDir, err)
		}
		if err := os.WriteFile(paths.DistVersionPath, []byte("  "), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", paths.DistVersionPath, err)
		}
		_, err := ReadBuildVersion()
		if err == nil || !strings.Contains(err.Error(), "empty build version in") {
			t.Fatalf("ReadBuildVersion(empty) error = %v, want empty error", err)
		}
	})
}

func TestResolveStageVersionReadBuildVersionError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", paths.DistDir, err)
	}
	// Create a directory where DistVersionPath should be to trigger a read error
	if err := os.MkdirAll(paths.DistVersionPath, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", paths.DistVersionPath, err)
	}

	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}
	_, err := ResolveStageVersion(cfg, false)
	if err == nil || !strings.Contains(err.Error(), "read build version") {
		t.Fatalf("ResolveStageVersion(read build version error) error = %v, want read error", err)
	}
}

func TestToPEP440EdgeCases(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := ToPEP440("  ")
		if err == nil || !strings.Contains(err.Error(), "version is empty") {
			t.Fatalf("ToPEP440(empty) error = %v, want empty error", err)
		}
	})

	t.Run("short_dev_part", func(t *testing.T) {
		got, err := ToPEP440("1.2.3-dev.5")
		if err != nil {
			t.Fatalf("ToPEP440(short dev) error = %v", err)
		}
		if got != "1.2.3.dev5" {
			t.Fatalf("ToPEP440(short dev) = %q, want %q", got, "1.2.3.dev5")
		}
	})
}

func TestReadOptionalProjectREADMEError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// Create a directory where README.md should be a file
	if err := os.Mkdir(ProjectREADMEPath, 0755); err != nil {
		t.Fatalf("os.Mkdir(%q) error = %v", ProjectREADMEPath, err)
	}
	_, exists, err := ReadOptionalProjectREADME()
	if err == nil || !strings.Contains(err.Error(), "read project README") {
		t.Fatalf("ReadOptionalProjectREADME() error = %v, want read error", err)
	}
	if exists {
		t.Fatalf("ReadOptionalProjectREADME() exists = true, want false")
	}
}

func TestResolveProjectREADMEPath(t *testing.T) {
	path, required := ResolveProjectREADMEPath("docs/root.md", "docs/dist.md")
	if path != filepath.Clean("docs/dist.md") || !required {
		t.Fatalf("ResolveProjectREADMEPath(dist) = (%q, %v)", path, required)
	}

	path, required = ResolveProjectREADMEPath(" docs/root.md ", "")
	if path != filepath.Clean("docs/root.md") || !required {
		t.Fatalf("ResolveProjectREADMEPath(global) = (%q, %v)", path, required)
	}

	path, required = ResolveProjectREADMEPath("", "")
	if path != ProjectREADMEPath || required {
		t.Fatalf("ResolveProjectREADMEPath(default) = (%q, %v)", path, required)
	}
}

func TestReadProjectREADMERequired(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	_, exists, err := ReadProjectREADME("docs/missing.md", true)
	if err == nil || !strings.Contains(err.Error(), "read project README") {
		t.Fatalf("ReadProjectREADME(required missing) error = %v, want read error", err)
	}
	if exists {
		t.Fatalf("ReadProjectREADME(required missing) exists = true, want false")
	}

	if err := os.MkdirAll("docs", 0755); err != nil {
		t.Fatalf("os.MkdirAll(docs) error = %v", err)
	}
	if err := os.WriteFile("docs/custom.md", []byte("hello"), 0644); err != nil {
		t.Fatalf("os.WriteFile(custom.md) error = %v", err)
	}
	data, exists, err := ReadProjectREADME("docs/custom.md", true)
	if err != nil {
		t.Fatalf("ReadProjectREADME(required) error = %v", err)
	}
	if !exists || string(data) != "hello" {
		t.Fatalf("ReadProjectREADME(required) = (%q, %v), want (%q, true)", string(data), exists, "hello")
	}
}

func TestWheelPlatformTagUnsupported(t *testing.T) {
	tests := []struct {
		name   string
		target config.Target
	}{
		{name: "linux_386", target: config.Target{OS: "linux", Arch: "386"}},
		{name: "darwin_386", target: config.Target{OS: "darwin", Arch: "386"}},
		{name: "windows_386", target: config.Target{OS: "windows", Arch: "386"}},
		{name: "unknown_os", target: config.Target{OS: "solaris", Arch: "amd64"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := WheelPlatformTag(tc.target, "")
			if err == nil {
				t.Fatalf("WheelPlatformTag(%+v) error = nil, want error", tc.target)
			}
		})
	}
}

func TestWheelPlatformTagLinuxDefault(t *testing.T) {
	target := config.Target{OS: "linux", Arch: "amd64"}
	got, err := WheelPlatformTag(target, "  ")
	if err != nil {
		t.Fatalf("WheelPlatformTag(linux default) error = %v", err)
	}
	if !strings.HasPrefix(got, DefaultUVLinuxTag) {
		t.Fatalf("WheelPlatformTag(linux default) = %q, want prefix %q", got, DefaultUVLinuxTag)
	}
}

func TestWheelFilenameError(t *testing.T) {
	target := config.Target{OS: "unknown", Arch: "amd64"}
	_, err := WheelFilename("pkg", "1.0.0", target, "")
	if err == nil {
		t.Fatalf("WheelFilename(unknown) error = nil, want error")
	}
}

func TestResolveStageVersionFallback(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	// No build version file

	t.Setenv(EnvVersionName, "1.2.3")
	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}
	got, err := ResolveStageVersion(cfg, false)
	if err != nil {
		t.Fatalf("ResolveStageVersion(fallback) error = %v", err)
	}
	if got != "1.2.3" {
		t.Fatalf("ResolveStageVersion(fallback) = %q, want %q", got, "1.2.3")
	}
}

func TestResolveStageVersionDev(t *testing.T) {
	t.Setenv(EnvVersionName, "1.2.3-dev.1")
	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}
	got, err := ResolveStageVersion(cfg, true)
	if err != nil {
		t.Fatalf("ResolveStageVersion(dev) error = %v", err)
	}
	if got != "1.2.3-dev.1" {
		t.Fatalf("ResolveStageVersion(dev) = %q, want %q", got, "1.2.3-dev.1")
	}
}
