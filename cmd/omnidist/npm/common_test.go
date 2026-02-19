package npm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
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

	got, err := getVersion(cfg, false)
	if err != nil {
		t.Fatalf("getVersion() error = %v", err)
	}
	if got != "1.2.3" {
		t.Fatalf("getVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestGetVersionFromFileTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.WriteFile("VERSION", []byte("2.4.6\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile(VERSION) error = %v", err)
	}

	cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
	got, err := getVersion(cfg, false)
	if err != nil {
		t.Fatalf("getVersion() error = %v", err)
	}
	if got != "2.4.6" {
		t.Fatalf("getVersion() = %q, want %q", got, "2.4.6")
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
			_, err := getVersion(tc.cfg, false)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("getVersion() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
