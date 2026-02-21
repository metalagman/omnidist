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

func executeCommand(args ...string) (string, error) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	return buf.String(), err
}
