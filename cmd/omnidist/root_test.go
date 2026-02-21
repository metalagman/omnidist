package main

import (
	"bytes"
	"strings"
	"testing"
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

	for _, cmd := range []string{"stage", "verify", "publish"} {
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

func executeCommand(args ...string) (string, error) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	return buf.String(), err
}
