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
