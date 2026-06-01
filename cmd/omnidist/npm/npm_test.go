package npm

import (
	"bytes"
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

func TestLoadConfigUsesViperConfigKey(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	customPath := filepath.Join(t.TempDir(), "custom-config.yaml")
	if err := config.Save(config.DefaultConfig(), customPath); err != nil {
		t.Fatalf("config.Save(%q) error = %v", customPath, err)
	}
	viper.Set("config", customPath)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.Tool.Name != "omnidist" {
		t.Fatalf("cfg.Tool.Name = %q, want %q", cfg.Tool.Name, "omnidist")
	}
}

func TestGetSelectedProfile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	if got := getSelectedProfile(); got != config.DefaultProfileName {
		t.Fatalf("getSelectedProfile() = %q, want %q", got, config.DefaultProfileName)
	}

	viper.Set("profile", "release")
	if got := getSelectedProfile(); got != "release" {
		t.Fatalf("getSelectedProfile() with profile set = %q, want %q", got, "release")
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
		{name: "trust", cmd: func() error { return trustCmd.RunE(trustCmd, nil) }},
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
		"trust":   false,
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

func TestTrustCommandPrintsAllPackages(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	viper.Reset()
	t.Cleanup(viper.Reset)
	flagTrustWorkflowFile = ""
	flagTrustRepo = ""
	flagTrustEnvironment = ""
	flagTrustStagePublish = false
	flagTrustApply = false

	cfg := config.DefaultConfig()
	npmDist := cfg.Distributions["npm"]
	npmDist.PublishAuth = "trusted"
	npmDist.RepositoryURL = "git+https://github.com/metalagman/omnidist.git"
	cfg.Distributions["npm"] = npmDist
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save(%q) error = %v", paths.ConfigPath, err)
	}

	var stdout bytes.Buffer
	trustCmd.SetOut(&stdout)
	trustCmd.SetErr(&stdout)
	defer trustCmd.SetOut(nil)
	defer trustCmd.SetErr(nil)

	if err := trustCmd.RunE(trustCmd, nil); err != nil {
		t.Fatalf("trustCmd.RunE() error = %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"npm trust github @omnidist/omnidist --repo metalagman/omnidist --file omnidist-release.yml --allow-publish --yes",
		"npm trust github @omnidist/omnidist-linux-x64 --repo metalagman/omnidist --file omnidist-release.yml --allow-publish --yes",
		"Printed 6 npm trust command(s)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("trust output missing %q\n---\n%s", want, output)
		}
	}
}

func TestTrustCommandApplyRunsNPMTrust(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based npm shim test")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	viper.Reset()
	t.Cleanup(viper.Reset)
	flagTrustWorkflowFile = "publish.yml"
	flagTrustRepo = "metalagman/omnidist"
	flagTrustEnvironment = "release"
	flagTrustStagePublish = true
	flagTrustApply = true

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save(%q) error = %v", paths.ConfigPath, err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", binDir, err)
	}
	npmPath := filepath.Join(binDir, "npm")
	logPath := filepath.Join(dir, "npm-trust.log")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shellScriptQuote(logPath) + "\n" +
		"case \"$1\" in\n" +
		"  trust)\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"  *)\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(npmPath, []byte(script), 0755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", npmPath, err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	trustCmd.SetOut(&stdout)
	trustCmd.SetErr(&stdout)
	defer trustCmd.SetOut(nil)
	defer trustCmd.SetErr(nil)

	if err := trustCmd.RunE(trustCmd, nil); err != nil {
		t.Fatalf("trustCmd.RunE() error = %v", err)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", logPath, err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "trust github @omnidist/omnidist --repo metalagman/omnidist --file publish.yml --env release --allow-publish --allow-stage-publish --yes") {
		t.Fatalf("npm trust log missing expected command\n---\n%s", logText)
	}
	if !strings.Contains(stdout.String(), "Configured npm trusted publishing for 6 package(s)") {
		t.Fatalf("stdout missing completion message\n---\n%s", stdout.String())
	}
}

func shellScriptQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
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
