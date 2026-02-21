package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigIncludesUV(t *testing.T) {
	cfg := DefaultConfig()
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
}

func TestLoadAppliesUVMissingDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "omnidist.yaml")

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
}

func TestLoadRejectsInvalidUVLinuxTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "omnidist.yaml")

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
