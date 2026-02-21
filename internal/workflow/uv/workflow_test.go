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
