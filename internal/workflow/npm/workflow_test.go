package npm

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestWriteShimResolvesScopedPlatformPackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	shimPath := filepath.Join(dir, "omnidist.js")

	if err := writeShim(shimPath, "omnidist", "@omnidist/omnidist"); err != nil {
		t.Fatalf("writeShim() error = %v", err)
	}

	data, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", shimPath, err)
	}

	shim := string(data)
	if !strings.Contains(shim, "const platformPkgName = '@omnidist/omnidist-' + platformKey;") {
		t.Fatalf("shim does not use scoped platform package name: %q", shim)
	}
	if !strings.Contains(shim, "require.resolve(platformPkgName + '/package.json', { paths: [__dirname] });") {
		t.Fatalf("shim does not resolve package via require.resolve: %q", shim)
	}
	if strings.Contains(shim, "const metaParts =") {
		t.Fatalf("shim still contains old sibling directory logic: %q", shim)
	}
}

func TestNPMDistribution(t *testing.T) {
	t.Parallel()

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
			name: "missing_npm_distribution",
			cfg: &config.Config{
				Distributions: map[string]config.DistributionConfig{},
			},
			wantErr: "missing required distribution: npm",
		},
		{
			name: "empty_package",
			cfg: &config.Config{
				Distributions: map[string]config.DistributionConfig{
					"npm": {Package: "   "},
				},
			},
			wantErr: "npm distribution package is required",
		},
		{
			name: "invalid_access",
			cfg: &config.Config{
				Distributions: map[string]config.DistributionConfig{
					"npm": {Package: "@scope/pkg", Access: "private"},
				},
			},
			wantErr: "invalid npm access",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := npmDistribution(tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("npmDistribution() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestNPMDistributionTrimsFields(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Distributions: map[string]config.DistributionConfig{
			"npm": {
				Package:  " @omnidist/omnidist ",
				Registry: " https://registry.npmjs.org ",
				Access:   " public ",
			},
		},
	}

	dist, err := npmDistribution(cfg)
	if err != nil {
		t.Fatalf("npmDistribution() error = %v", err)
	}

	if dist.Package != "@omnidist/omnidist" {
		t.Fatalf("dist.Package = %q, want %q", dist.Package, "@omnidist/omnidist")
	}
	if dist.Registry != "https://registry.npmjs.org" {
		t.Fatalf("dist.Registry = %q, want %q", dist.Registry, "https://registry.npmjs.org")
	}
	if dist.Access != "public" {
		t.Fatalf("dist.Access = %q, want %q", dist.Access, "public")
	}
}

func TestGetVersionFromEnvTrimsWhitespace(t *testing.T) {
	t.Setenv("VERSION", " 1.2.3 \n")
	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}

	got, err := shared.ResolveVersion(cfg, false)
	if err != nil {
		t.Fatalf("ResolveVersion() error = %v", err)
	}
	if got != "1.2.3" {
		t.Fatalf("ResolveVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestGetVersionFromFileTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.WriteFile("VERSION", []byte("2.4.6\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile(VERSION) error = %v", err)
	}

	cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
	got, err := shared.ResolveVersion(cfg, false)
	if err != nil {
		t.Fatalf("ResolveVersion() error = %v", err)
	}
	if got != "2.4.6" {
		t.Fatalf("ResolveVersion() = %q, want %q", got, "2.4.6")
	}
}

func TestGetVersionErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		setup   func(t *testing.T)
		wantErr string
	}{
		{
			name:    "unknown_source",
			cfg:     &config.Config{Version: config.VersionConfig{Source: "bad-source"}},
			wantErr: "unknown version source",
		},
		{
			name: "missing_env",
			cfg:  &config.Config{Version: config.VersionConfig{Source: "env"}},
			setup: func(t *testing.T) {
				t.Setenv("VERSION", "")
			},
			wantErr: "empty version from source",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			_, err := shared.ResolveVersion(tc.cfg, false)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ResolveVersion() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestBuildPublishArgs(t *testing.T) {
	t.Parallel()

	got := buildPublishArgs("https://registry.npmjs.org", "public", PublishOptions{})
	want := []string{
		"publish",
		"--access", "public",
		"--registry", "https://registry.npmjs.org",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildPublishArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildPublishArgsFlagOverrides(t *testing.T) {
	t.Parallel()

	got := buildPublishArgs("https://registry.npmjs.org", "restricted", PublishOptions{
		DryRun:   true,
		Tag:      "next",
		Registry: "https://npm.example.internal",
		OTP:      "123456",
	})
	want := []string{
		"publish",
		"--access", "restricted",
		"--dry-run",
		"--tag", "next",
		"--registry", "https://npm.example.internal",
		"--otp", "123456",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildPublishArgs() = %#v, want %#v", got, want)
	}
}

func TestNPMTokenConfigKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		registry string
		want     string
		wantErr  string
	}{
		{
			name:     "npmjs",
			registry: "https://registry.npmjs.org",
			want:     "//registry.npmjs.org/:_authToken",
		},
		{
			name:     "host_without_scheme",
			registry: "registry.npmjs.org",
			want:     "//registry.npmjs.org/:_authToken",
		},
		{
			name:     "protocol_relative",
			registry: "//registry.npmjs.org",
			want:     "//registry.npmjs.org/:_authToken",
		},
		{
			name:     "registry_with_path",
			registry: "https://npm.example.internal/repository/npm-private",
			want:     "//npm.example.internal/repository/npm-private/:_authToken",
		},
		{
			name:     "missing_host",
			registry: "https://",
			wantErr:  "missing host",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := npmTokenConfigKey(tc.registry)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("npmTokenConfigKey(%q) error = %v, want substring %q", tc.registry, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("npmTokenConfigKey(%q) error = %v", tc.registry, err)
			}
			if got != tc.want {
				t.Fatalf("npmTokenConfigKey(%q) = %q, want %q", tc.registry, got, tc.want)
			}
		})
	}
}

func TestEnsureWorkspaceNPMRC(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	npmrcPath, err := ensureWorkspaceNPMRC("https://registry.npmjs.org")
	if err != nil {
		t.Fatalf("ensureWorkspaceNPMRC() error = %v", err)
	}

	wantPath := filepath.Join(dir, paths.NPMRCPath)
	if npmrcPath != wantPath {
		t.Fatalf("ensureWorkspaceNPMRC() path = %q, want %q", npmrcPath, wantPath)
	}

	data, err := os.ReadFile(npmrcPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", npmrcPath, err)
	}
	content := string(data)
	for _, want := range []string{
		"registry=https://registry.npmjs.org",
		"//registry.npmjs.org/:_authToken=${NPM_PUBLISH_TOKEN}",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf(".npmrc missing %q, got:\n%s", want, content)
		}
	}
}

func TestResolvePublishToken(t *testing.T) {
	tests := []struct {
		name       string
		publishTok string
		dryRun     bool
		want       string
		wantErr    bool
	}{
		{
			name:       "prefers_publish_token",
			publishTok: "publish-token",
			want:       "publish-token",
		},
		{
			name:    "missing_token_fails_non_dry_run",
			wantErr: true,
		},
		{
			name:    "dry_run_allows_missing_token",
			dryRun:  true,
			want:    "",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NPM_PUBLISH_TOKEN", tc.publishTok)
			got, err := resolvePublishToken(tc.dryRun)
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

func TestEnsureWorkingDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr string
	}{
		{
			name: "valid_directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
		},
		{
			name: "empty_directory",
			setup: func(t *testing.T) string {
				return "  "
			},
			wantErr: "working directory is empty",
		},
		{
			name: "path_is_file",
			setup: func(t *testing.T) string {
				file := filepath.Join(t.TempDir(), "not-a-dir")
				if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
				return file
			},
			wantErr: "not a directory",
		},
		{
			name: "missing_directory",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing")
			},
			wantErr: "no such file or directory",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setup(t)
			got, err := ensureWorkingDir(dir)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ensureWorkingDir(%q) error = %v, want substring %q", dir, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ensureWorkingDir(%q) error = %v", dir, err)
			}
			if got == "" {
				t.Fatalf("ensureWorkingDir(%q) returned empty path", dir)
			}
		})
	}
}

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

func TestVerifyDetectsPlatformVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("VERSION", "2.0.0")

	cfg := testConfig()
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatalf("createDistArtifacts() error = %v", err)
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatalf("Stage() error = %v", err)
	}

	target := cfg.Targets[0]
	pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
	pkgDir := filepath.Join(paths.NPMDir, pkgName)
	pkgJSON, err := readPackageJSON(pkgDir)
	if err != nil {
		t.Fatalf("readPackageJSON(%q) error = %v", pkgDir, err)
	}
	pkgJSON["version"] = "9.9.9"
	if err := writePackageJSON(pkgDir, pkgJSON); err != nil {
		t.Fatalf("writePackageJSON(%q) error = %v", pkgDir, err)
	}

	result := Verify(cfg)
	if result.Valid {
		t.Fatalf("Verify().Valid = true, want false")
	}

	foundMismatch := false
	for _, errMsg := range result.Errors {
		if strings.Contains(errMsg, "Version mismatch in "+pkgName) {
			foundMismatch = true
			break
		}
	}
	if !foundMismatch {
		t.Fatalf("Verify() errors = %v, want version mismatch for %s", result.Errors, pkgName)
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
