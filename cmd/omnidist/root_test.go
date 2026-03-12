package main

import (
	"bytes"
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

func executeCommand(args ...string) (string, error) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// Reset global state
	cfgFile = ""
	viper.Reset()
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Reset all flags in the command tree to their default values
	resetFlags(rootCmd)

	err := rootCmd.Execute()
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
