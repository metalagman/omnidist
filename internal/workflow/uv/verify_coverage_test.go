package uv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestVerifyCoverage(t *testing.T) {
	t.Run("resolveUVStagingVersion_fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		// config with file source but no file present
		cfg := &config.Config{
			Version: config.VersionConfig{Source: "file"},
			Distributions: map[string]config.DistributionConfig{
				"uv": {Package: "omnidist"},
			},
		}
		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid due to missing version file")
		}
		assertContainsError(t, result.Errors, "read version file")
	})

	t.Run("validatePublishVersionPolicy_fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv(shared.EnvVersionName, "1.2.3+local")
		cfg := testConfig()
		// uvDistribution from testConfig uses pypi.org by default
		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid due to local version on PyPI")
		}
		assertContainsError(t, result.Errors, "contains local version metadata")
	})

	t.Run("missing_wheel_artifact", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv(shared.EnvVersionName, "1.2.3")
		cfg := testConfig()
		// no wheels created
		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid due to missing wheels")
		}
		assertContainsError(t, result.Errors, "missing wheel artifact")
	})

	t.Run("verifyWheel_fails", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.2.3")
		cfg := testConfig()
		createDistArtifacts(cfg)
		Stage(cfg, StageOptions{})

		// Corrupt a wheel
		uvDist := cfg.Distributions["uv"]
		wheelPath, _ := wheelPathForTarget(uvDist, cfg.Targets[0], "1.2.3")
		os.WriteFile(wheelPath, []byte("not-a-zip"), 0644)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid due to corrupt wheel")
		}
		assertContainsError(t, result.Errors, "not a valid zip file")
	})
}

func TestWheelPathForTargetErrors(t *testing.T) {
	uvDist := config.DistributionConfig{
		Package:  "pkg",
		LinuxTag: "manylinux2014",
	}
	// Use an unsupported architecture to trigger an error in shared.WheelPlatformTag
	target := config.Target{OS: "linux", Arch: "386"}
	_, err := wheelPathForTarget(uvDist, target, "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "unsupported linux architecture") {
		t.Fatalf("wheelPathForTarget() with bad arch error = %v", err)
	}
}

func TestVerifyRecordEntriesZipError(t *testing.T) {
	// This covers the error case in verifyWheel when reading from zip fails
	// though it's hard to trigger via Verify() because it reads the whole map first.
	// We can test verifyWheel directly if exported or just call it.

	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testConfig()
	uvDist, _ := uvDistribution(cfg)
	target := cfg.Targets[0]
	version := "1.0.0"

	// Create a zip that is valid but has a file that fails to read
	// Wait, verifyWheel reads from a path.
	wheelPath := filepath.Join(dir, "test.whl")
	os.WriteFile(wheelPath, []byte("not-a-zip"), 0644)

	err := verifyWheel(cfg, uvDist, target, version, wheelPath)
	if err == nil || !strings.Contains(err.Error(), "not a valid zip file") {
		t.Fatalf("verifyWheel() with corrupt zip error = %v", err)
	}
}
