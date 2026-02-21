package uv

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestStageAndVerifyPasses(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("VERSION", "1.2.3")

	cfg := testConfig()
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatalf("createDistArtifacts() error = %v", err)
	}

	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatalf("Stage() error = %v", err)
	}
	if _, err := os.Stat(paths.UVPyprojectPath); err != nil {
		t.Fatalf("staging pyproject missing: %v", err)
	}
	version, err := readStagingPyprojectVersion()
	if err != nil {
		t.Fatalf("readStagingPyprojectVersion() error = %v", err)
	}
	if version != "1.2.3" {
		t.Fatalf("readStagingPyprojectVersion() = %q, want %q", version, "1.2.3")
	}

	result := Verify(cfg)
	if !result.Valid {
		t.Fatalf("Verify().Valid = false, errors = %v", result.Errors)
	}
}

func TestVerifyDetectsMissingBinary(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("VERSION", "1.2.3")

	cfg := testConfig()
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatalf("createDistArtifacts() error = %v", err)
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatalf("Stage() error = %v", err)
	}

	uvDist := cfg.Distributions["uv"]
	version := "1.2.3"
	wheelPath, err := wheelPathForTarget(uvDist, cfg.Targets[0], version)
	if err != nil {
		t.Fatalf("wheelPathForTarget() error = %v", err)
	}
	if err := stripBinaryFromWheel(wheelPath); err != nil {
		t.Fatalf("stripBinaryFromWheel() error = %v", err)
	}

	result := Verify(cfg)
	if result.Valid {
		t.Fatalf("Verify().Valid = true, want false")
	}

	found := false
	for _, errMsg := range result.Errors {
		if strings.Contains(errMsg, "missing binary") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Verify() errors = %v, want missing binary error", result.Errors)
	}
}

func TestCheckDependencyMissing(t *testing.T) {
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())

	err := CheckDependency()
	if err == nil {
		t.Fatalf("CheckDependency() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "Install uv") {
		t.Fatalf("CheckDependency() error = %q, want install guidance", err.Error())
	}

	t.Setenv("PATH", originalPath)
}

func TestBuildPublishArgs(t *testing.T) {
	artifacts := []string{
		filepath.Join(paths.UVDistDir, "b.whl"),
		filepath.Join(paths.UVDistDir, "a.whl"),
	}
	sort.Strings(artifacts)

	got := buildPublishArgs("https://upload.pypi.org/legacy/", PublishOptions{
		DryRun:     true,
		PublishURL: "https://pypi.internal/legacy/",
	}, artifacts)

	want := []string{
		"publish",
		"--dry-run",
		"--publish-url", "https://pypi.internal/legacy/",
		filepath.Join(paths.UVDistDir, "a.whl"),
		filepath.Join(paths.UVDistDir, "b.whl"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildPublishArgs() = %#v, want %#v", got, want)
	}
}

func TestUVDistValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr string
	}{
		{
			name:    "nil_config",
			cfg:     nil,
			wantErr: "config is nil",
		},
		{
			name: "empty_package",
			cfg: &config.Config{Distributions: map[string]config.DistributionConfig{
				"uv": {Package: "", LinuxTag: "manylinux2014"},
			}},
			wantErr: "uv distribution package is required",
		},
		{
			name: "invalid_linux_tag",
			cfg: &config.Config{Distributions: map[string]config.DistributionConfig{
				"uv": {Package: "omnidist", LinuxTag: "bad"},
			}},
			wantErr: "invalid uv linux-tag",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := uvDistribution(tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("uvDistribution() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestResolvePublishToken(t *testing.T) {
	tests := []struct {
		name     string
		opts     PublishOptions
		envToken string
		want     string
		wantErr  bool
	}{
		{
			name:     "uses_flag_token",
			opts:     PublishOptions{Token: "flag-token"},
			envToken: "env-token",
			want:     "flag-token",
		},
		{
			name:     "uses_env_token",
			opts:     PublishOptions{},
			envToken: "env-token",
			want:     "env-token",
		},
		{
			name:    "dry_run_allows_missing_token",
			opts:    PublishOptions{DryRun: true},
			want:    "",
			wantErr: false,
		},
		{
			name:    "publish_requires_token",
			opts:    PublishOptions{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("UV_PUBLISH_TOKEN", tc.envToken)
			got, err := resolvePublishToken(tc.opts)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolvePublishToken() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePublishToken() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolvePublishToken() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveUVReleaseVersion(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		envVer  string
		want    string
		wantErr bool
	}{
		{
			name:   "exact_semver",
			cfg:    &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer: "1.2.3",
			want:   "1.2.3",
		},
		{
			name:    "non_semver_fails",
			cfg:     &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer:  "1.2.3-dev.1.gabc123",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VERSION", tc.envVer)
			got, err := resolveUVReleaseVersion(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveUVReleaseVersion() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveUVReleaseVersion() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveUVReleaseVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveUVPublishVersion(t *testing.T) {
	tests := []struct {
		name         string
		writePyproj  string
		writeVersion string
		cfg          *config.Config
		envVer       string
		want         string
		wantErr      bool
	}{
		{
			name:        "uses_staging_pyproject_version",
			writePyproj: "2.0.0.dev7+abc123",
			cfg:         &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer:      "9.9.9",
			want:        "2.0.0.dev7+abc123",
		},
		{
			name:         "uses_build_version",
			writeVersion: "1.2.3-4-gabc123",
			cfg:          &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer:       "9.9.9",
			want:         "1.2.3.dev4+abc123",
		},
		{
			name:    "fallback_to_release_version",
			cfg:     &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer:  "1.2.3",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "fallback_release_must_be_semver",
			cfg:     &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer:  "1.2.3-dev.1.gabc123",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("VERSION", tc.envVer)
			if tc.writePyproj != "" {
				if err := writeStagingPyproject("omnidist", tc.writePyproj); err != nil {
					t.Fatalf("writeStagingPyproject() error = %v", err)
				}
			}
			if tc.writeVersion != "" {
				if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
					t.Fatalf("os.MkdirAll(%q) error = %v", paths.DistDir, err)
				}
				if err := os.WriteFile(paths.DistVersionPath, []byte(tc.writeVersion+"\n"), 0644); err != nil {
					t.Fatalf("os.WriteFile(%q) error = %v", paths.DistVersionPath, err)
				}
			}

			got, err := resolveUVPublishVersion(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveUVPublishVersion() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveUVPublishVersion() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveUVPublishVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStagingPyprojectRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := writeStagingPyproject("omnidist", "1.2.3.dev4+abc123"); err != nil {
		t.Fatalf("writeStagingPyproject() error = %v", err)
	}

	got, err := readStagingPyprojectVersion()
	if err != nil {
		t.Fatalf("readStagingPyprojectVersion() error = %v", err)
	}
	if got != "1.2.3.dev4+abc123" {
		t.Fatalf("readStagingPyprojectVersion() = %q, want %q", got, "1.2.3.dev4+abc123")
	}
}

func TestValidatePublishVersionPolicy(t *testing.T) {
	tests := []struct {
		name     string
		indexURL string
		version  string
		wantErr  bool
	}{
		{
			name:     "pypi_rejects_local_version",
			indexURL: "https://upload.pypi.org/legacy/",
			version:  "1.2.3.dev4+abc123",
			wantErr:  true,
		},
		{
			name:     "testpypi_rejects_local_version",
			indexURL: "https://test.pypi.org/legacy/",
			version:  "1.2.3+meta",
			wantErr:  true,
		},
		{
			name:     "pypi_accepts_non_local",
			indexURL: "https://upload.pypi.org/legacy/",
			version:  "1.2.3.dev4",
			wantErr:  false,
		},
		{
			name:     "custom_index_allows_local",
			indexURL: "https://packages.example.com/legacy/",
			version:  "1.2.3.dev4+abc123",
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validatePublishVersionPolicy(tc.indexURL, tc.version)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validatePublishVersionPolicy() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validatePublishVersionPolicy() error = %v", err)
			}
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Tool: config.ToolConfig{Name: "omnidist"},
		Version: config.VersionConfig{
			Source: "env",
		},
		Targets: []config.Target{
			{OS: "linux", Arch: "amd64"},
			{OS: "win32", Arch: "amd64"},
		},
		Distributions: map[string]config.DistributionConfig{
			"uv": {
				Package:  "omnidist",
				IndexURL: "https://upload.pypi.org/legacy/",
				LinuxTag: "manylinux2014",
			},
		},
	}
}

func createDistArtifacts(cfg *config.Config) error {
	for _, target := range cfg.Targets {
		binaryName := cfg.Tool.Name
		if target.OS == "win32" {
			binaryName += ".exe"
		}
		outPath := filepath.Join(paths.DistDir, target.OS, config.MapArchToNPM(target.Arch), binaryName)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte("binary"), 0755); err != nil {
			return err
		}
	}
	return nil
}

func stripBinaryFromWheel(wheelPath string) error {
	reader, err := zip.OpenReader(wheelPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	tmpPath := wheelPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(f)
	for _, entry := range reader.File {
		if strings.Contains(entry.Name, "/bin/") {
			continue
		}

		rc, err := entry.Open()
		if err != nil {
			writer.Close()
			f.Close()
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			writer.Close()
			f.Close()
			return err
		}

		header := entry.FileHeader
		w, err := writer.CreateHeader(&header)
		if err != nil {
			writer.Close()
			f.Close()
			return err
		}
		if _, err := w.Write(data); err != nil {
			writer.Close()
			f.Close()
			return err
		}
	}

	if err := writer.Close(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, wheelPath); err != nil {
		return err
	}
	return nil
}
