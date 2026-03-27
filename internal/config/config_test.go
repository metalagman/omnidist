package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	if npmDist.License != "" {
		t.Fatalf("npm license = %q, want empty default", npmDist.License)
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

func TestLoadTrimsReadmePaths(t *testing.T) {
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
readme-path: " docs/README.md "
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
    readme-path: " docs/npm.md "
  uv:
    package: "tool"
    readme-path: " docs/uv.md "
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.ReadmePath; got != "docs/README.md" {
		t.Fatalf("readme-path = %q, want %q", got, "docs/README.md")
	}
	if got := cfg.Distributions["npm"].ReadmePath; got != "docs/npm.md" {
		t.Fatalf("distributions.npm.readme-path = %q, want %q", got, "docs/npm.md")
	}
	if got := cfg.Distributions["uv"].ReadmePath; got != "docs/uv.md" {
		t.Fatalf("distributions.uv.readme-path = %q, want %q", got, "docs/uv.md")
	}
}

func TestLoadTrimsNPMLicense(t *testing.T) {
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
    license: "  Apache-2.0  "
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.Distributions["npm"].License; got != "Apache-2.0" {
		t.Fatalf("npm license = %q, want %q", got, "Apache-2.0")
	}
}

func TestLoadNormalizesNPMKeywords(t *testing.T) {
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
    keywords:
      - " ai "
      - "llm"
      - ""
      - "llm"
      - "  "
      - "cli"
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"ai", "llm", "cli"}
	if got := cfg.Distributions["npm"].Keywords; !reflect.DeepEqual(got, want) {
		t.Fatalf("npm keywords = %#v, want %#v", got, want)
	}
}

func TestLoadSupportsFixedVersionSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: fixed
  fixed: " 1.2.3 "
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.Version.Source; got != "fixed" {
		t.Fatalf("version.source = %q, want %q", got, "fixed")
	}
	if got := cfg.Version.Fixed; got != "1.2.3" {
		t.Fatalf("version.fixed = %q, want %q", got, "1.2.3")
	}
}

func TestLoadAppliesDefaultVersionFileForFileSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: file
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Version.File; got != DefaultVersionFile {
		t.Fatalf("version.file = %q, want %q", got, DefaultVersionFile)
	}
}

func TestLoadPreservesExplicitVersionFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: file
  file: "  versions/release.txt "
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Version.File; got != "versions/release.txt" {
		t.Fatalf("version.file = %q, want %q", got, "versions/release.txt")
	}
}

func TestLoadRejectsFixedVersionSourceWithoutValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: fixed
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load() error = nil, want fixed version validation error")
	}
	if !strings.Contains(err.Error(), "version.fixed is required") {
		t.Fatalf("Load() error = %v, want fixed version validation error", err)
	}
}

func TestLoadRejectsLegacyFixedVersionKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: fixed
  fixed-version: "1.2.3"
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load() error = nil, want legacy key migration error")
	}
	if !strings.Contains(err.Error(), "version.fixed-version is no longer supported") {
		t.Fatalf("Load() error = %v, want legacy key migration error", err)
	}
}

func TestLoadRejectsUnknownVersionSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	yaml := `tool:
  name: omnidist
  main: ./cmd/omnidist
version:
  source: mystery
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`

	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load() error = nil, want invalid source error")
	}
	if !strings.Contains(err.Error(), "invalid version.source") {
		t.Fatalf("Load() error = %v, want invalid version.source error", err)
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
			name: "invalid version source",
			cfg: &Config{
				Version: VersionConfig{Source: "mystery"},
				Targets: []Target{{OS: "linux", Arch: "amd64"}},
			},
			wantErr: "invalid version.source",
		},
		{
			name: "fixed version source missing fixed-version",
			cfg: &Config{
				Version: VersionConfig{Source: "fixed"},
				Targets: []Target{{OS: "linux", Arch: "amd64"}},
			},
			wantErr: "version.fixed is required",
		},
		{
			name: "fixed version source with fixed-version",
			cfg: &Config{
				Version: VersionConfig{
					Source: "fixed",
					Fixed:  "1.2.3",
				},
				Targets: []Target{{OS: "linux", Arch: "amd64"}},
			},
		},
		{
			name: "file version source without file path uses default",
			cfg: &Config{
				Version: VersionConfig{Source: "file"},
				Targets: []Target{{OS: "linux", Arch: "amd64"}},
			},
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

func TestApplyVersionDefaults(t *testing.T) {
	tests := []struct {
		name      string
		version   VersionConfig
		wantSrc   string
		wantFile  string
		wantFixed string
	}{
		{
			name:    "empty source defaults to git-tag",
			version: VersionConfig{},
			wantSrc: "git-tag",
		},
		{
			name: "file source defaults file path",
			version: VersionConfig{
				Source: " file ",
			},
			wantSrc:  "file",
			wantFile: DefaultVersionFile,
		},
		{
			name: "file source preserves explicit path",
			version: VersionConfig{
				Source: "file",
				File:   " versions/release.txt ",
			},
			wantSrc:  "file",
			wantFile: "versions/release.txt",
		},
		{
			name: "fixed source trims fixed value",
			version: VersionConfig{
				Source: " fixed ",
				Fixed:  " 1.2.3 ",
			},
			wantSrc:   "fixed",
			wantFixed: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Version: tt.version}
			applyVersionDefaults(cfg)
			if got := cfg.Version.Source; got != tt.wantSrc {
				t.Fatalf("source = %q, want %q", got, tt.wantSrc)
			}
			if got := cfg.Version.File; got != tt.wantFile {
				t.Fatalf("file = %q, want %q", got, tt.wantFile)
			}
			if got := cfg.Version.Fixed; got != tt.wantFixed {
				t.Fatalf("fixed = %q, want %q", got, tt.wantFixed)
			}
		})
	}
}

func TestHasLegacyFixedVersionKey(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "invalid yaml",
			yaml: "version: [",
			want: false,
		},
		{
			name: "no version section",
			yaml: "tool:\n  name: omnidist\n",
			want: false,
		},
		{
			name: "version not map",
			yaml: "version: fixed\n",
			want: false,
		},
		{
			name: "version map without legacy key",
			yaml: "version:\n  source: fixed\n  fixed: 1.2.3\n",
			want: false,
		},
		{
			name: "version map with legacy key",
			yaml: "version:\n  source: fixed\n  fixed-version: 1.2.3\n",
			want: true,
		},
		{
			name: "profiles map with legacy key",
			yaml: "profiles:\n  release:\n    version:\n      source: fixed\n      fixed-version: 1.2.3\n",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasLegacyFixedVersionKey([]byte(tt.yaml))
			if got != tt.want {
				t.Fatalf("hasLegacyFixedVersionKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadWithProfileLegacyModeRuntimeDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := configFile(path, `tool:
  name: legacy
  main: ./cmd/legacy
version:
  source: env
targets:
  - os: linux
    arch: amd64
build:
  ldflags: -s -w
  tags: []
  cgo: false
`); err != nil {
		t.Fatalf("configFile() error = %v", err)
	}

	cfg, err := LoadWithProfile(path, "release")
	if err != nil {
		t.Fatalf("LoadWithProfile() error = %v", err)
	}
	if cfg.SelectedProfile() != DefaultProfileName {
		t.Fatalf("SelectedProfile() = %q, want %q", cfg.SelectedProfile(), DefaultProfileName)
	}
	if cfg.IsProfilesMode() {
		t.Fatalf("IsProfilesMode() = true, want false")
	}
	if got := cfg.EffectiveWorkspaceDir(); got != DefaultWorkspaceDir {
		t.Fatalf("EffectiveWorkspaceDir() = %q, want %q", got, DefaultWorkspaceDir)
	}
}

func TestLoadWithProfileProfilesModeSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := configFile(path, `profiles:
  default:
    tool:
      name: app-default
      main: ./cmd/app-default
    version:
      source: fixed
      fixed: 1.0.0
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
  release:
    tool:
      name: app-release
      main: ./cmd/app-release
    version:
      source: file
      file: versions/release.txt
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
`); err != nil {
		t.Fatalf("configFile() error = %v", err)
	}

	cfg, err := LoadWithProfile(path, "release")
	if err != nil {
		t.Fatalf("LoadWithProfile(release) error = %v", err)
	}
	if got := cfg.Tool.Name; got != "app-release" {
		t.Fatalf("Tool.Name = %q, want %q", got, "app-release")
	}
	if got := cfg.SelectedProfile(); got != "release" {
		t.Fatalf("SelectedProfile() = %q, want %q", got, "release")
	}
	if !cfg.IsProfilesMode() {
		t.Fatalf("IsProfilesMode() = false, want true")
	}
	if got := cfg.EffectiveWorkspaceDir(); got != ".omnidist/release" {
		t.Fatalf("EffectiveWorkspaceDir() = %q, want %q", got, ".omnidist/release")
	}
}

func TestLoadWithProfileProfilesModeDefaultSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, paths.ConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := configFile(path, `profiles:
  default:
    tool:
      name: app-default
      main: ./cmd/app-default
    version:
      source: fixed
      fixed: 1.0.0
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
`); err != nil {
		t.Fatalf("configFile() error = %v", err)
	}

	cfg, err := LoadWithProfile(path, "")
	if err != nil {
		t.Fatalf("LoadWithProfile(default) error = %v", err)
	}
	if got := cfg.SelectedProfile(); got != DefaultProfileName {
		t.Fatalf("SelectedProfile() = %q, want %q", got, DefaultProfileName)
	}
	if got := cfg.EffectiveWorkspaceDir(); got != ".omnidist/default" {
		t.Fatalf("EffectiveWorkspaceDir() = %q, want %q", got, ".omnidist/default")
	}
}

func TestLoadWithProfileErrors(t *testing.T) {
	t.Run("missing profile", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.yaml")
		if err := configFile(path, `profiles:
  default:
    tool:
      name: app
      main: ./cmd/app
    version:
      source: fixed
      fixed: 1.0.0
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
`); err != nil {
			t.Fatalf("configFile() error = %v", err)
		}

		_, err := LoadWithProfile(path, "release")
		if err == nil || !strings.Contains(err.Error(), "available profiles") {
			t.Fatalf("LoadWithProfile(missing) error = %v, want available profiles message", err)
		}
	})

	t.Run("mixed format", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.yaml")
		if err := configFile(path, `tool:
  name: app
  main: ./cmd/app
profiles:
  default:
    tool:
      name: app
      main: ./cmd/app
`); err != nil {
			t.Fatalf("configFile() error = %v", err)
		}

		_, err := LoadWithProfile(path, "")
		if err == nil || !strings.Contains(err.Error(), "mixed format") {
			t.Fatalf("LoadWithProfile(mixed) error = %v, want mixed format error", err)
		}
	})

	t.Run("invalid selected profile", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.yaml")
		if err := configFile(path, `profiles:
  default:
    tool:
      name: app
      main: ./cmd/app
    version:
      source: fixed
      fixed: 1.0.0
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
`); err != nil {
			t.Fatalf("configFile() error = %v", err)
		}

		_, err := LoadWithProfile(path, "bad/profile")
		if err == nil || !strings.Contains(err.Error(), "invalid profile name") {
			t.Fatalf("LoadWithProfile(invalid selected profile) error = %v, want invalid profile name", err)
		}
	})

	t.Run("invalid configured profile key", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.yaml")
		if err := configFile(path, `profiles:
  bad/profile:
    tool:
      name: app
      main: ./cmd/app
    version:
      source: fixed
      fixed: 1.0.0
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
`); err != nil {
			t.Fatalf("configFile() error = %v", err)
		}

		_, err := LoadWithProfile(path, "bad/profile")
		if err == nil || !strings.Contains(err.Error(), "invalid profile name") {
			t.Fatalf("LoadWithProfile(invalid configured profile) error = %v, want invalid profile name", err)
		}
	})
}

func TestRuntimeHelpersHandleNilConfig(t *testing.T) {
	var cfg *Config
	if got := cfg.EffectiveWorkspaceDir(); got != DefaultWorkspaceDir {
		t.Fatalf("nil EffectiveWorkspaceDir() = %q, want %q", got, DefaultWorkspaceDir)
	}
	if got := cfg.SelectedProfile(); got != DefaultProfileName {
		t.Fatalf("nil SelectedProfile() = %q, want %q", got, DefaultProfileName)
	}
	if cfg.IsProfilesMode() {
		t.Fatalf("nil IsProfilesMode() = true, want false")
	}
}

func TestValidateProfileNameReservedNames(t *testing.T) {
	for _, name := range []string{".", ".."} {
		err := validateProfileName(name)
		if err == nil || !strings.Contains(err.Error(), "invalid profile name") {
			t.Fatalf("validateProfileName(%q) error = %v, want invalid profile name", name, err)
		}
	}
}

func TestContainsLegacyFixedVersionKeyCoversCollections(t *testing.T) {
	if !containsLegacyFixedVersionKey(map[interface{}]interface{}{
		"version": map[interface{}]interface{}{
			"fixed-version": "1.2.3",
		},
	}) {
		t.Fatalf("containsLegacyFixedVersionKey(map[interface{}]interface{}) = false, want true")
	}

	if !containsLegacyFixedVersionKey([]interface{}{
		map[string]interface{}{
			"fixed-version": "1.2.3",
		},
	}) {
		t.Fatalf("containsLegacyFixedVersionKey([]interface{}) = false, want true")
	}
}

func configFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
