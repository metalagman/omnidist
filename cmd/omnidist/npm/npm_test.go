package npm

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

func TestResolveStageVersionForOutput(t *testing.T) {
	cfg := &config.Config{Version: config.VersionConfig{Source: "env"}}

	t.Run("dev_uses_resolve_version", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv(shared.EnvVersionName, "1.2.3-dev.1.gabc123")

		got, err := resolveStageVersionForOutput(cfg, true)
		if err != nil {
			t.Fatalf("resolveStageVersionForOutput(dev=true) error = %v", err)
		}
		if got != "1.2.3-dev.1.gabc123" {
			t.Fatalf("resolveStageVersionForOutput(dev=true) = %q, want %q", got, "1.2.3-dev.1.gabc123")
		}
	})

	t.Run("missing_build_version_file", func(t *testing.T) {
		t.Chdir(t.TempDir())

		_, err := resolveStageVersionForOutput(cfg, false)
		if err == nil {
			t.Fatalf("resolveStageVersionForOutput(dev=false) error = nil, want error")
		}
		if !strings.Contains(err.Error(), "missing build version file") {
			t.Fatalf("resolveStageVersionForOutput(dev=false) error = %v, want missing build version message", err)
		}
	})

	t.Run("uses_build_version_file", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := shared.WriteBuildVersion("2.3.4"); err != nil {
			t.Fatalf("shared.WriteBuildVersion() error = %v", err)
		}

		got, err := resolveStageVersionForOutput(cfg, false)
		if err != nil {
			t.Fatalf("resolveStageVersionForOutput(dev=false) error = %v", err)
		}
		if got != "2.3.4" {
			t.Fatalf("resolveStageVersionForOutput(dev=false) = %q, want %q", got, "2.3.4")
		}
	})
}

func TestRunStageNilConfigReturnsError(t *testing.T) {
	err := runStage(nil)
	if err == nil {
		t.Fatalf("runStage(nil) error = nil, want error")
	}
}

func TestSubcommandsReturnConfigLoadErrorWhenMissingConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	flagDev = false
	flagDryRun = true
	flagTag = ""
	flagRegistry = ""
	flagOTP = ""

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
	t.Helper()
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

func TestLoadConfigErrorIncludesPathContext(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.SetConfigFile(filepath.Join(t.TempDir(), "missing.yaml"))
	_, err := loadConfig()
	if err == nil {
		t.Fatalf("loadConfig() error = nil, want error")
	}
	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("loadConfig() error = %v, want missing file/read context", err)
	}
}

func TestStageVerifyPublishCommandsSucceed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based npm shim test")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	viper.Reset()
	t.Cleanup(viper.Reset)
	flagDev = false
	flagDryRun = true
	flagTag = ""
	flagRegistry = ""
	flagOTP = ""

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
	if err := installFakeNPM(t, dir); err != nil {
		t.Fatalf("installFakeNPM() error = %v", err)
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

func installFakeNPM(t *testing.T, dir string) error {
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	npmPath := filepath.Join(binDir, "npm")
	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  whoami)\n" +
		"    echo test-user\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"  publish)\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"  *)\n" +
		"    echo unsupported command: $1 >&2\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(npmPath, []byte(script), 0755); err != nil {
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
