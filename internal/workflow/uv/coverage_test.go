package uv

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestCheckDependencyFail(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir) // empty path

	err := CheckDependency()
	if err == nil {
		t.Fatalf("CheckDependency() with missing uv error = nil, want error")
	}
}

func TestStageErrorsExtra(t *testing.T) {
	t.Run("resolve_version_fail", func(t *testing.T) {
		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		err := Stage(cfg, StageOptions{})
		if err == nil {
			t.Fatalf("Stage(resolve version fail) error = nil, want error")
		}
	})

	t.Run("reset_dir_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		if err := shared.WriteBuildVersion("1.0.0"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}
		
		if err := os.MkdirAll(paths.WorkspaceDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		// Make UVDir a file to trigger MkdirAll error in resetUVStagingDir
		if err := os.WriteFile(paths.UVDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", paths.UVDir, err)
		}

		cfg := testConfig()
		err := Stage(cfg, StageOptions{})
		if err == nil || !strings.Contains(err.Error(), "clean uv staging directory") {
			t.Fatalf("Stage(reset fail) error = %v, want clean error", err)
		}
	})
}

func TestVerifyErrorsExtra(t *testing.T) {
	t.Run("missing_wheel", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		if err := shared.WriteBuildVersion("1.0.0"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}
		cfg := testConfig()
		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify().Valid = true, want false")
		}
		assertContainsError(t, result.Errors, "missing wheel artifact")
	})
}

func TestPublishErrorsExtra(t *testing.T) {
	t.Run("check_dependency_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv("PATH", dir) // empty path
		t.Setenv(shared.EnvVersionName, "1.0.0")
		
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		err := Publish(cfg, PublishOptions{})
		if err == nil || !strings.Contains(err.Error(), "Install uv") {
			t.Fatalf("Publish(no uv) error = %v, want install guidance", err)
		}
	})

	t.Run("no_artifacts", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		if err := shared.WriteBuildVersion("1.0.0"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}
		os.MkdirAll(paths.UVDistDir, 0755)

		// Provide dummy uv to pass CheckDependency
		binDir := filepath.Join(dir, "bin")
		os.MkdirAll(binDir, 0755)
		uvPath := filepath.Join(binDir, "uv")
		os.WriteFile(uvPath, []byte("#!/bin/sh\nexit 0"), 0755)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		cfg := testConfig()
		err := Publish(cfg, PublishOptions{})
		if err == nil || !strings.Contains(err.Error(), "missing wheel artifact") {
			t.Fatalf("Publish(no artifacts) error = %v, want missing wheel error", err)
		}
	})
}

func TestIsPyPIIndexURLNegative(t *testing.T) {
	if isPyPIIndexURL("https://npm.example.com") {
		t.Fatalf("isPyPIIndexURL(npm) = true, want false")
	}
}

func TestWriteWheelErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly")
	os.MkdirAll(path, 0555)
	defer os.Chmod(path, 0755)
	
	wheelPath := filepath.Join(path, "test.whl")
	err := writeWheel(wheelPath, &config.Config{}, config.DistributionConfig{}, config.Target{}, "1.0.0", nil)
	if err == nil {
		t.Fatalf("writeWheel(readonly) error = nil, want error")
	}
}

type mockRawZipWriter struct {
	createRawErr error
}

func (m *mockRawZipWriter) CreateRaw(h *zip.FileHeader) (io.Writer, error) {
	return nil, m.createRawErr
}
func (m *mockRawZipWriter) Close() error { return nil }

func TestAddZipFileErrors(t *testing.T) {
	err := addZipFile(&mockRawZipWriter{createRawErr: io.EOF}, "missing", nil, 0644)
	if err == nil {
		t.Fatalf("addZipFile(missing) error = nil, want error")
	}
}

func TestCollectWheelArtifactsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testConfig()
	uvDist, _ := uvDistribution(cfg)
	t.Setenv(shared.EnvVersionName, "1.0.0")
	
	// collectWheelArtifacts will fail if artifacts are missing
	_, err := collectWheelArtifacts(cfg, uvDist, "1.0.0")
	if err == nil {
		t.Fatalf("collectWheelArtifacts(missing) error = nil, want error")
	}
}
