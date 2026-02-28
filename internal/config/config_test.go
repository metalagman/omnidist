package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/paths"
)

func TestDefaultConfigIncludesUV(t *testing.T) {
	cfg := DefaultConfig()
	npmDist, ok := cfg.Distributions["npm"]
	if !ok {
		t.Fatalf("DefaultConfig() missing npm distribution")
	}
	if !npmDist.IncludeREADMEEnabled() {
		t.Fatalf("npm include-readme default = false, want true")
	}
	uvDist, ok := cfg.Distributions["uv"]
	if !ok {
		t.Fatalf("DefaultConfig() missing uv distribution")
	}
	if uvDist.Package != "omnidist" {
		t.Fatalf("uv package = %q, want %q", uvDist.Package, "omnidist")
	}
	if uvDist.LinuxTag != "manylinux2014" {
		t.Fatalf("uv linux tag = %q, want %q", uvDist.LinuxTag, "manylinux2014")
	}
	if !uvDist.IncludeREADMEEnabled() {
		t.Fatalf("uv include-readme default = false, want true")
	}
}

func TestLoadAppliesUVMissingDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: env
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
distributions:
  npm:
    package: "@scope/tool"
  uv:
    package: "tool"
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	uvDist := cfg.Distributions["uv"]
	if uvDist.IndexURL != "https://upload.pypi.org/legacy/" {
		t.Fatalf("uv index-url = %q, want default", uvDist.IndexURL)
	}
	if uvDist.LinuxTag != "manylinux2014" {
		t.Fatalf("uv linux-tag = %q, want default", uvDist.LinuxTag)
	}
	if !uvDist.IncludeREADMEEnabled() {
		t.Fatalf("uv include-readme = false, want default true")
	}

	npmDist := cfg.Distributions["npm"]
	if !npmDist.IncludeREADMEEnabled() {
		t.Fatalf("npm include-readme = false, want default true")
	}
}

func TestLoadPreservesExplicitIncludeReadmeFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: env
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
distributions:
  npm:
    package: "@scope/tool"
    include-readme: false
  uv:
    package: "tool"
    include-readme: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Distributions["npm"].IncludeREADMEEnabled() {
		t.Fatalf("npm include-readme = true, want false")
	}
	if cfg.Distributions["uv"].IncludeREADMEEnabled() {
		t.Fatalf("uv include-readme = true, want false")
	}
}

func TestLoadRejectsInvalidUVLinuxTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: env
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
distributions:
  uv:
    package: "tool"
    linux-tag: "badtag"
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load() error = nil, want invalid linux-tag error")
	}
	if !strings.Contains(err.Error(), "invalid distributions.uv.linux-tag") {
		t.Fatalf("Load() error = %v, want linux-tag validation error", err)
	}
}

func TestLoadMissingFileIncludesPathContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "omnidist.yaml")

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load(%q) error = nil, want error", path)
	}
	if !strings.Contains(err.Error(), "read config file "+path) {
		t.Fatalf("Load(%q) error = %v, want read context with path", path, err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load(%q) error = %v, want os.ErrNotExist", path, err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	cfg := DefaultConfig()
	cfg.Tool.Name = "mytool"

	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Tool.Name != "mytool" {
		t.Fatalf("loaded tool name = %q, want %q", got.Tool.Name, "mytool")
	}
}

func TestSaveInvalidPath(t *testing.T) {
	// Use a path that is likely to fail (e.g., a directory that exists as a file)
	dir := t.TempDir()
	path := filepath.Join(dir, "isfile")
	if err := os.WriteFile(path, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	configPath := filepath.Join(path, "omnidist.yaml")

	err := Save(DefaultConfig(), configPath)
	if err == nil {
		t.Fatalf("Save to invalid path error = nil, want error")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: "config is nil",
		},
		{
			name: "missing target os",
			cfg: &Config{
				Targets: []Target{{Arch: "amd64"}},
			},
			wantErr: "targets[0].os is required",
		},
		{
			name: "missing target arch",
			cfg: &Config{
				Targets: []Target{{OS: "linux"}},
			},
			wantErr: "targets[0].arch is required",
		},
		{
			name: "invalid win32 os",
			cfg: &Config{
				Targets: []Target{{OS: "win32", Arch: "amd64"}},
			},
			wantErr: "use Go GOOS value \"windows\"",
		},
		{
			name: "invalid x64 arch",
			cfg: &Config{
				Targets: []Target{{OS: "linux", Arch: "x64"}},
			},
			wantErr: "use Go GOARCH value \"amd64\"",
		},
		{
			name: "invalid npm access",
			cfg: &Config{
				Targets: []Target{{OS: "linux", Arch: "amd64"}},
				Distributions: map[string]DistributionConfig{
					"npm": {Access: "invalid"},
				},
			},
			wantErr: "invalid distributions.npm.access \"invalid\"",
		},
		{
			name: "missing uv package",
			cfg: &Config{
				Targets: []Target{{OS: "linux", Arch: "amd64"}},
				Distributions: map[string]DistributionConfig{
					"uv": {LinuxTag: "manylinux2014"},
				},
			},
			wantErr: "distributions.uv.package is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if err == nil {
				if tt.wantErr != "" {
					t.Fatalf("validate() error = nil, want %q", tt.wantErr)
				}
				return
			}
			if tt.wantErr == "" {
				t.Fatalf("validate() error = %v, want nil", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestMapGoArchToNPM(t *testing.T) {
	tests := map[string]string{
		"amd64": "x64",
		"arm64": "arm64",
		"386":   "x86",
		"mips":  "mips",
	}
	for in, want := range tests {
		if got := MapGoArchToNPM(in); got != want {
			t.Fatalf("MapGoArchToNPM(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapGoOSToNPM(t *testing.T) {
	tests := map[string]string{
		"windows": "win32",
		"linux":   "linux",
		"darwin":  "darwin",
	}
	for in, want := range tests {
		if got := MapGoOSToNPM(in); got != want {
			t.Fatalf("MapGoOSToNPM(%q) = %q, want %q", in, got, want)
		}
	}
}
