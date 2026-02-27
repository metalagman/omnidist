package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestNPMCommandFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based npm shim test")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	if err := setupCommandFlowProject(); err != nil {
		t.Fatalf("setupCommandFlowProject() error = %v", err)
	}
	if err := installFakeTool(t, dir, "npm", "#!/bin/sh\ncase \"$1\" in\n  whoami) exit 0 ;;\n  publish) exit 0 ;;\n  *) exit 1 ;;\nesac\n"); err != nil {
		t.Fatalf("installFakeTool(npm) error = %v", err)
	}

	_, err := executeCommand("stage", "--only", "npm")
	if err != nil {
		t.Fatalf("executeCommand(stage --only npm) error = %v", err)
	}

	_, err = executeCommand("verify", "--only", "npm")
	if err != nil {
		t.Fatalf("executeCommand(verify --only npm) error = %v", err)
	}

	_, err = executeCommand("publish", "--only", "npm", "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(publish --only npm --dry-run) error = %v", err)
	}
}

func TestUVCommandFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based uv shim test")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	if err := setupCommandFlowProject(); err != nil {
		t.Fatalf("setupCommandFlowProject() error = %v", err)
	}
	if err := installFakeTool(t, dir, "uv", "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("installFakeTool(uv) error = %v", err)
	}

	_, err := executeCommand("stage", "--only", "uv")
	if err != nil {
		t.Fatalf("executeCommand(stage --only uv) error = %v", err)
	}

	_, err = executeCommand("verify", "--only", "uv")
	if err != nil {
		t.Fatalf("executeCommand(verify --only uv) error = %v", err)
	}

	_, err = executeCommand("publish", "--only", "uv", "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(publish --only uv --dry-run) error = %v", err)
	}
}

func setupCommandFlowProject() error {
	cfg := config.DefaultConfig()
	cfg.Version.Source = "env"
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		return err
	}
	for _, target := range cfg.Targets {
		binaryName := cfg.Tool.Name
		if target.OS == "windows" {
			binaryName += ".exe"
		}
		outPath := filepath.Join(paths.DistDir, target.OS, target.Arch, binaryName)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte("binary"), 0755); err != nil {
			return err
		}
	}
	return shared.WriteBuildVersion("1.2.3")
}

func installFakeTool(t *testing.T, dir string, name string, script string) error {
	t.Helper()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	toolPath := filepath.Join(binDir, name)
	if err := os.WriteFile(toolPath, []byte(script), 0755); err != nil {
		return err
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return nil
}
