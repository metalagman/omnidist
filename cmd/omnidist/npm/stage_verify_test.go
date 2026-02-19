package npm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestRunStageAndVerifyPasses(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("VERSION", "1.2.3")

	origFlagDev := flagDev
	flagDev = false
	t.Cleanup(func() {
		flagDev = origFlagDev
	})

	cfg := testConfig()
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatalf("createDistArtifacts() error = %v", err)
	}

	if err := runStage(cfg); err != nil {
		t.Fatalf("runStage() error = %v", err)
	}

	result := runVerify(cfg)
	if !result.Valid {
		t.Fatalf("runVerify().Valid = false, errors = %v", result.Errors)
	}
}

func TestRunVerifyDetectsPlatformVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("VERSION", "2.0.0")

	origFlagDev := flagDev
	flagDev = false
	t.Cleanup(func() {
		flagDev = origFlagDev
	})

	cfg := testConfig()
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatalf("createDistArtifacts() error = %v", err)
	}
	if err := runStage(cfg); err != nil {
		t.Fatalf("runStage() error = %v", err)
	}

	target := cfg.Targets[0]
	pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
	pkgDir := filepath.Join("npm", pkgName)
	pkgJSON, err := readPackageJSON(pkgDir)
	if err != nil {
		t.Fatalf("readPackageJSON(%q) error = %v", pkgDir, err)
	}
	pkgJSON["version"] = "9.9.9"
	if err := writePackageJSON(pkgDir, pkgJSON); err != nil {
		t.Fatalf("writePackageJSON(%q) error = %v", pkgDir, err)
	}

	result := runVerify(cfg)
	if result.Valid {
		t.Fatalf("runVerify().Valid = true, want false")
	}

	foundMismatch := false
	for _, errMsg := range result.Errors {
		if strings.Contains(errMsg, "Version mismatch in "+pkgName) {
			foundMismatch = true
			break
		}
	}
	if !foundMismatch {
		t.Fatalf("runVerify() errors = %v, want version mismatch for %s", result.Errors, pkgName)
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Tool: config.ToolConfig{
			Name: "omnidist",
		},
		Version: config.VersionConfig{
			Source: "env",
		},
		Targets: []config.Target{
			{OS: "linux", Arch: "amd64"},
			{OS: "win32", Arch: "amd64"},
		},
		Distributions: map[string]config.DistributionConfig{
			"npm": {
				Package:  "@omnidist/omnidist",
				Registry: "https://registry.npmjs.org",
				Access:   "public",
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

		outPath := filepath.Join("dist", target.OS, config.MapArchToNPM(target.Arch), binaryName)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte("binary"), 0755); err != nil {
			return err
		}
	}
	return nil
}
