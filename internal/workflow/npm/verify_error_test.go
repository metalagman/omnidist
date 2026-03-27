package npm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestVerifyErrors(t *testing.T) {
	t.Run("resolveNPMVersion_fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		cfg := testConfig()
		cfg.Version.Source = "file"
		// no version file, no build version file, no staged package
		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "read version file")
	})

	t.Run("platform_missing_package_json", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		// Remove a platform package.json
		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		os.Remove(filepath.Join(paths.NPMDir, pkgName, "package.json"))

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("Missing package.json for %s", pkgName))
	})

	t.Run("platform_os_mismatch", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		pkgDir := filepath.Join(paths.NPMDir, pkgName)
		pkgJSON, _ := readPackageJSON(pkgDir)
		pkgJSON["os"] = []interface{}{"wrong-os"}
		writePackageJSON(pkgDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("os mismatch in %s", pkgName))
	})

	t.Run("platform_cpu_mismatch", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		pkgDir := filepath.Join(paths.NPMDir, pkgName)
		pkgJSON, _ := readPackageJSON(pkgDir)
		pkgJSON["cpu"] = []interface{}{"wrong-cpu"}
		writePackageJSON(pkgDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("cpu mismatch in %s", pkgName))
	})

	t.Run("platform_missing_binary", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		pkgDir := filepath.Join(paths.NPMDir, pkgName)
		binaryName := cfg.Tool.Name
		if target.OS == "windows" {
			binaryName += ".exe"
		}
		os.Remove(filepath.Join(pkgDir, "bin", binaryName))

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("Missing binary %s in %s", binaryName, pkgName))
	})

	t.Run("platform_postinstall_forbidden", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		pkgDir := filepath.Join(paths.NPMDir, pkgName)
		pkgJSON, _ := readPackageJSON(pkgDir)
		pkgJSON["scripts"] = map[string]interface{}{"postinstall": "do-something"}
		writePackageJSON(pkgDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("Scripts.postinstall found in %s", pkgName))
	})

	t.Run("meta_missing_package_json", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		os.Remove(filepath.Join(metaDir, "package.json"))

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "Missing meta package.json")
	})

	t.Run("meta_postinstall_forbidden", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		pkgJSON, _ := readPackageJSON(metaDir)
		pkgJSON["scripts"] = map[string]interface{}{"postinstall": "do-something"}
		writePackageJSON(metaDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "Scripts.postinstall found in meta package")
	})

	t.Run("meta_keywords_mismatch", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		npmDist := cfg.Distributions["npm"]
		npmDist.Keywords = []string{"ai", "llm", "cli"}
		cfg.Distributions["npm"] = npmDist
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		pkgJSON, _ := readPackageJSON(metaDir)
		pkgJSON["keywords"] = []interface{}{"ai"}
		writePackageJSON(metaDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "Meta package keywords mismatch")
	})

	t.Run("configured_license_mismatch", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		npmDist := cfg.Distributions["npm"]
		npmDist.License = "MIT"
		cfg.Distributions["npm"] = npmDist
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		metaJSON, _ := readPackageJSON(metaDir)
		metaJSON["license"] = "Apache-2.0"
		writePackageJSON(metaDir, metaJSON)

		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		pkgDir := filepath.Join(paths.NPMDir, pkgName)
		pkgJSON, _ := readPackageJSON(pkgDir)
		pkgJSON["license"] = "Apache-2.0"
		writePackageJSON(pkgDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "Meta package license mismatch")
		assertContainsError(t, result.Errors, fmt.Sprintf("license mismatch in %s", pkgName))
	})

	t.Run("meta_missing_optionalDependencies", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		pkgJSON, _ := readPackageJSON(metaDir)
		delete(pkgJSON, "optionalDependencies")
		writePackageJSON(metaDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "Missing optionalDependencies in meta package")
	})

	t.Run("meta_missing_platform_package_in_optionalDependencies", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		pkgJSON, _ := readPackageJSON(metaDir)
		optionalDeps := pkgJSON["optionalDependencies"].(map[string]interface{})
		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		delete(optionalDeps, pkgName)
		writePackageJSON(metaDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("Missing %s in optionalDependencies", pkgName))
	})

	t.Run("meta_version_mismatch_in_optionalDependencies", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		pkgJSON, _ := readPackageJSON(metaDir)
		optionalDeps := pkgJSON["optionalDependencies"].(map[string]interface{})
		target := cfg.Targets[0]
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		optionalDeps[pkgName] = "2.0.0"
		writePackageJSON(metaDir, pkgJSON)

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, fmt.Sprintf("Version mismatch for %s in optionalDependencies", pkgName))
	})

	t.Run("meta_missing_shim", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		t.Setenv(shared.EnvVersionName, "1.0.0")
		cfg := testConfig()
		createDistArtifacts(cfg)
		shared.WriteBuildVersion("1.0.0")
		Stage(cfg, StageOptions{})

		metaDir := filepath.Join(paths.NPMDir, cfg.Distributions["npm"].Package)
		os.Remove(filepath.Join(metaDir, cfg.Tool.Name+".js"))

		result := Verify(cfg)
		if result.Valid {
			t.Fatalf("Verify() = valid, want invalid")
		}
		assertContainsError(t, result.Errors, "Missing shim in meta package")
	})
}

func assertContainsError(t *testing.T, errors []string, want string) {
	t.Helper()
	for _, err := range errors {
		if strings.Contains(err, want) {
			return
		}
	}
	t.Fatalf("Errors %v do not contain %q", errors, want)
}
