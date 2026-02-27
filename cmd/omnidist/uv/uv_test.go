package uv

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	"github.com/spf13/viper"
)

func TestLoadConfigUsesViperConfigFile(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	customPath := filepath.Join(t.TempDir(), "custom-config.yaml")
	if err := config.Save(config.DefaultConfig(), customPath); err != nil {
		t.Fatalf("config.Save(%q) error = %v", customPath, err)
	}
	viper.SetConfigFile(customPath)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.Tool.Name != "omnidist" {
		t.Fatalf("cfg.Tool.Name = %q, want %q", cfg.Tool.Name, "omnidist")
	}
}

func TestLoadConfigFallsBackToDefaultPath(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	if err := config.Save(config.DefaultConfig(), paths.ConfigPath); err != nil {
		t.Fatalf("config.Save(%q) error = %v", paths.ConfigPath, err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.Tool.Main != "./cmd/omnidist" {
		t.Fatalf("cfg.Tool.Main = %q, want %q", cfg.Tool.Main, "./cmd/omnidist")
	}
}

func TestSubcommandsFailWhenUVDependencyMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	stageDev = false
	publishDryRun = true
	publishURL = ""
	publishLegacyURL = ""
	publishToken = ""
	t.Setenv("PATH", "")

	tests := []struct {
		name string
		cmd  func() error
	}{
		{name: "stage", cmd: func() error { return stageCmd.RunE(stageCmd, nil) }},
		{name: "verify", cmd: func() error { return verifyCmd.RunE(verifyCmd, nil) }},
		{name: "publish", cmd: func() error { return publishCmd.RunE(publishCmd, nil) }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd()
			if err == nil {
				t.Fatalf("%s command error = nil, want error", tc.name)
			}
			if !strings.Contains(err.Error(), "uv executable not found in PATH") {
				t.Fatalf("%s command error = %v, want uv dependency message", tc.name, err)
			}
		})
	}
}

func TestSubcommandsReachConfigLoadWhenUVExists(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	stageDev = false
	publishDryRun = true
	publishURL = ""
	publishLegacyURL = ""
	publishToken = ""

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", binDir, err)
	}
	uvPath := filepath.Join(binDir, "uv")
	if err := os.WriteFile(uvPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", uvPath, err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name string
		cmd  func() error
	}{
		{name: "stage", cmd: func() error { return stageCmd.RunE(stageCmd, nil) }},
		{name: "verify", cmd: func() error { return verifyCmd.RunE(verifyCmd, nil) }},
		{name: "publish", cmd: func() error { return publishCmd.RunE(publishCmd, nil) }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd()
			if err == nil {
				t.Fatalf("%s command error = nil, want error", tc.name)
			}
			if !strings.Contains(err.Error(), "load config:") {
				t.Fatalf("%s command error = %v, want load config context", tc.name, err)
			}
		})
	}
}

func TestCmdIncludesExpectedSubcommands(t *testing.T) {
	want := map[string]bool{
		"stage":   false,
		"verify":  false,
		"publish": false,
	}

	for _, sub := range Cmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("Cmd missing subcommand %q", name)
		}
	}
}

func TestStageVerifyPublishCommandsSucceed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based uv shim test")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	viper.Reset()
	t.Cleanup(viper.Reset)
	stageDev = false
	publishDryRun = true
	publishURL = ""
	publishLegacyURL = ""
	publishToken = ""

	cfg := config.DefaultConfig()
	cfg.Version.Source = "env"
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save(%q) error = %v", paths.ConfigPath, err)
	}
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatalf("createDistArtifacts() error = %v", err)
	}
	if err := shared.WriteBuildVersion("1.2.3"); err != nil {
		t.Fatalf("shared.WriteBuildVersion() error = %v", err)
	}
	if err := installFakeUV(t, dir); err != nil {
		t.Fatalf("installFakeUV() error = %v", err)
	}

	if err := stageCmd.RunE(stageCmd, nil); err != nil {
		t.Fatalf("stageCmd.RunE() error = %v", err)
	}
	if err := verifyCmd.RunE(verifyCmd, nil); err != nil {
		t.Fatalf("verifyCmd.RunE() error = %v", err)
	}
	if err := publishCmd.RunE(publishCmd, nil); err != nil {
		t.Fatalf("publishCmd.RunE() error = %v", err)
	}
}

func installFakeUV(t *testing.T, dir string) error {
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	uvPath := filepath.Join(binDir, "uv")
	if err := os.WriteFile(uvPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		return err
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return nil
}

func createDistArtifacts(cfg *config.Config) error {
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
	return nil
}
