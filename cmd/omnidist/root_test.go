package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestUVCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"uv"})
	if err != nil {
		t.Fatalf("rootCmd.Find(uv) error = %v", err)
	}
	if cmd == nil || cmd.Name() != "uv" {
		t.Fatalf("uv command not registered")
	}
}

func TestUVHelpContainsSubcommands(t *testing.T) {
	output, err := executeCommand("uv", "--help")
	if err != nil {
		t.Fatalf("executeCommand(uv --help) error = %v", err)
	}
	for _, sub := range []string{"stage", "verify", "publish"} {
		if !strings.Contains(output, sub) {
			t.Fatalf("help output missing %q: %s", sub, output)
		}
	}
}

func TestUVPublishHelpFlags(t *testing.T) {
	output, err := executeCommand("uv", "publish", "--help")
	if err != nil {
		t.Fatalf("executeCommand(uv publish --help) error = %v", err)
	}

	for _, flag := range []string{"--dry-run", "--publish-url", "--token"} {
		if !strings.Contains(output, flag) {
			t.Fatalf("publish help missing %q: %s", flag, output)
		}
	}
}

func TestRootHelpContainsTopLevelDistributionCommands(t *testing.T) {
	output, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("executeCommand(--help) error = %v", err)
	}

	for _, cmd := range []string{"stage", "verify", "publish", "quickstart", "ci"} {
		if !strings.Contains(output, cmd) {
			t.Fatalf("root help output missing %q: %s", cmd, output)
		}
	}
}

func TestStageHelpFlags(t *testing.T) {
	output, err := executeCommand("stage", "--help")
	if err != nil {
		t.Fatalf("executeCommand(stage --help) error = %v", err)
	}

	for _, flag := range []string{"--dev", "--only"} {
		if !strings.Contains(output, flag) {
			t.Fatalf("stage help missing %q: %s", flag, output)
		}
	}
}

func TestVerifyHelpFlags(t *testing.T) {
	output, err := executeCommand("verify", "--help")
	if err != nil {
		t.Fatalf("executeCommand(verify --help) error = %v", err)
	}
	if !strings.Contains(output, "--only") {
		t.Fatalf("verify help missing --only: %s", output)
	}
}

func TestPublishHelpFlags(t *testing.T) {
	output, err := executeCommand("publish", "--help")
	if err != nil {
		t.Fatalf("executeCommand(publish --help) error = %v", err)
	}

	for _, flag := range []string{"--dry-run", "--only"} {
		if !strings.Contains(output, flag) {
			t.Fatalf("publish help missing %q: %s", flag, output)
		}
	}
}

func TestBuildHelpDocumentsLDFLagsTemplating(t *testing.T) {
	output, err := executeCommand("build", "--help")
	if err != nil {
		t.Fatalf("executeCommand(build --help) error = %v", err)
	}
	for _, want := range []string{
		"os.ExpandEnv",
		"${OMNIDIST_VERSION}",
		"OMNIDIST_GIT_COMMIT",
		"OMNIDIST_BUILD_DATE",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("build help missing %q: %s", want, output)
		}
	}
}

func TestCIHelpFlags(t *testing.T) {
	output, err := executeCommand("ci", "--help")
	if err != nil {
		t.Fatalf("executeCommand(ci --help) error = %v", err)
	}
	for _, flag := range []string{"--force", "--dry-run"} {
		if !strings.Contains(output, flag) {
			t.Fatalf("ci help missing %q: %s", flag, output)
		}
	}
}

func TestGlobalConfigFlag(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Create a custom config file in a different location
	customConfig := "custom-config.yaml"
	cfg := config.DefaultConfig()
	cfg.Tool.Name = "custom-tool"
	if err := config.Save(cfg, customConfig); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	// Run build --dry-run (or just any command that loads config)
	// We'll use ci --dry-run since we just implemented it and it's easy to check
	output, err := executeCommand("ci", "--config", customConfig, "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(ci --config --dry-run) error = %v, output=%s", err, output)
	}

	// The generated workflow should contain the custom tool name if it was used
	// Let's check if the generated workflow has something specific to the config
	// Actually, the workflow generation might not use Tool.Name in a way that's easy to see in YAML
	// Let's check if it fails if the config is missing
	output, err = executeCommand("ci", "--config", "non-existent.yaml", "--dry-run")
	if err == nil {
		t.Fatalf("executeCommand with non-existent config should fail. Output: %s", output)
	}
	if !strings.Contains(err.Error(), "non-existent.yaml") {
		t.Fatalf("error message should contain config path: %v. Output: %s", err, output)
	}
}

func TestGlobalOmnidistRootFlagUsesRootAsWorkingDirectory(t *testing.T) {
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	if err := config.Save(config.DefaultConfig(), ".omnidist/omnidist.yaml"); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	startDir := t.TempDir()
	t.Chdir(startDir)

	output, err := executeCommand("quickstart", "--omnidist-root", projectDir)
	if err != nil {
		t.Fatalf("executeCommand(quickstart --omnidist-root) error = %v, output=%s", err, output)
	}
	if !strings.Contains(output, "Config: .omnidist/omnidist.yaml") {
		t.Fatalf("quickstart output missing root-resolved config path: %s", output)
	}
}

func TestGlobalOmnidistRootWithRelativeConfigPath(t *testing.T) {
	projectDir := t.TempDir()
	customConfig := filepath.Join(projectDir, "configs", "custom.yaml")
	if err := config.Save(config.DefaultConfig(), customConfig); err != nil {
		t.Fatalf("config.Save(%q) error = %v", customConfig, err)
	}

	startDir := t.TempDir()
	t.Chdir(startDir)

	output, err := executeCommand("ci", "--omnidist-root", projectDir, "--config", "configs/custom.yaml", "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(ci --omnidist-root --config --dry-run) error = %v, output=%s", err, output)
	}
	if !strings.Contains(output, "name: omnidist-release") {
		t.Fatalf("workflow output missing expected content: %s", output)
	}
}

func TestGlobalConfigPathFromEnv(t *testing.T) {
	projectDir := t.TempDir()
	t.Chdir(projectDir)

	customConfig := filepath.Join(projectDir, "configs", "custom.yaml")
	if err := config.Save(config.DefaultConfig(), customConfig); err != nil {
		t.Fatalf("config.Save(%q) error = %v", customConfig, err)
	}
	t.Setenv("OMNIDIST_CONFIG", customConfig)

	output, err := executeCommand("quickstart")
	if err != nil {
		t.Fatalf("executeCommand(quickstart with OMNIDIST_CONFIG) error = %v, output=%s", err, output)
	}
	if !strings.Contains(output, "Config: "+customConfig) {
		t.Fatalf("quickstart output missing env-config path %q: %s", customConfig, output)
	}
}

func TestGetConfigPathUsesViperConfigFileUsed(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := config.Save(config.DefaultConfig(), path); err != nil {
		t.Fatalf("config.Save(%q) error = %v", path, err)
	}
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("viper.ReadInConfig() error = %v", err)
	}

	if got := getConfigPath(); got != path {
		t.Fatalf("getConfigPath() = %q, want %q", got, path)
	}
}

func TestGlobalOmnidistRootFromEnv(t *testing.T) {
	projectDir := t.TempDir()
	if err := config.Save(config.DefaultConfig(), filepath.Join(projectDir, ".omnidist", "omnidist.yaml")); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	startDir := t.TempDir()
	t.Chdir(startDir)
	t.Setenv("OMNIDIST_OMNIDIST_ROOT", projectDir)

	output, err := executeCommand("quickstart")
	if err != nil {
		t.Fatalf("executeCommand(quickstart with OMNIDIST_OMNIDIST_ROOT) error = %v, output=%s", err, output)
	}
	if !strings.Contains(output, "Config: .omnidist/omnidist.yaml") {
		t.Fatalf("quickstart output missing root-resolved config path: %s", output)
	}
}

func TestGlobalProfileFromEnv(t *testing.T) {
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	configPath := filepath.Join(projectDir, ".omnidist", "omnidist.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(configPath), err)
	}
	content := `profiles:
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
  release:
    tool:
      name: app
      main: ./cmd/app
    version:
      source: fixed
      fixed: 2.0.0
    targets:
      - os: linux
        arch: amd64
    build:
      ldflags: -s -w
      tags: []
      cgo: false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", configPath, err)
	}
	t.Setenv("OMNIDIST_PROFILE", "release")

	output, err := executeCommand("ci", "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(ci --dry-run with OMNIDIST_PROFILE) error = %v, output=%s", err, output)
	}
	if !strings.Contains(output, "path: .omnidist/release/dist/**/*") {
		t.Fatalf("ci dry-run output missing release workspace path: %s", output)
	}
	if !strings.Contains(output, "omnidist --profile 'release' build") {
		t.Fatalf("ci dry-run output missing profile-aware command: %s", output)
	}
}

func TestGlobalOmnidistRootFlagInvalidPath(t *testing.T) {
	output, err := executeCommand("quickstart", "--omnidist-root", filepath.Join(t.TempDir(), "missing-root"))
	if err == nil {
		t.Fatalf("executeCommand with invalid --omnidist-root should fail. Output: %s", output)
	}
	if !strings.Contains(err.Error(), "stat --omnidist-root") {
		t.Fatalf("error = %v, want stat --omnidist-root context", err)
	}
}

func TestGlobalOmnidistRootFlagPathIsFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "root-file")
	if err := os.WriteFile(filePath, []byte("not-a-dir"), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", filePath, err)
	}

	output, err := executeCommand("quickstart", "--omnidist-root", filePath)
	if err == nil {
		t.Fatalf("executeCommand with file --omnidist-root should fail. Output: %s", output)
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("error = %v, want directory type validation error", err)
	}
}

func TestRootHelpContainsOmnidistRootFlag(t *testing.T) {
	output, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("executeCommand(--help) error = %v", err)
	}
	if !strings.Contains(output, "--omnidist-root") {
		t.Fatalf("root help missing --omnidist-root: %s", output)
	}
	if !strings.Contains(output, "--profile") {
		t.Fatalf("root help missing --profile: %s", output)
	}
}

func executeCommand(args ...string) (string, error) {
	origWD, wdErr := os.Getwd()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// Reset global state
	cfgFile = ""
	omnidistRoot = ""
	profileName = ""
	initRootErr = nil
	viper.Reset()
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Reset all flags in the command tree to their default values
	resetFlags(rootCmd)

	err := rootCmd.Execute()
	if wdErr == nil {
		_ = os.Chdir(origWD)
	}
	rootCmd.SetArgs(nil)
	return buf.String(), err
}

func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, c := range cmd.Commands() {
		resetFlags(c)
	}
}

func TestExecuteUsesRootCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs(nil)

	if err := Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(buf.String(), "omnidist") {
		t.Fatalf("Execute() output = %q, want omnidist help", buf.String())
	}
}

func TestMainHelpPathDoesNotExit(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs(nil)

	main()
	if !strings.Contains(buf.String(), "omnidist") {
		t.Fatalf("main() output = %q, want omnidist help", buf.String())
	}
}
