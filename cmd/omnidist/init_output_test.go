package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitPrintsNextSteps(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(filepath.Join("cmd", "my-tool"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("cmd", "my-tool", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand("init")
	if err != nil {
		t.Fatalf("executeCommand(init) error = %v", err)
	}

	for _, want := range []string{
		"Created .omnidist/omnidist.yaml",
		"Next steps:",
		"Edit .omnidist/omnidist.yaml",
		"Set environment variables in .env",
		"omnidist build",
		"omnidist stage",
		"omnidist verify",
		"omnidist publish",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("init output missing %q: %s", want, output)
		}
	}
	if strings.Index(output, "omnidist verify") > strings.Index(output, "omnidist publish") {
		t.Fatalf("verify must be listed before publish: %s", output)
	}
}

func TestInitHelpExposesSafeOverrides(t *testing.T) {
	output, err := executeCommand("init", "--help")
	if err != nil {
		t.Fatalf("executeCommand(init --help) error = %v", err)
	}
	for _, flag := range []string{"--force", "--name", "--main"} {
		if !strings.Contains(output, flag) {
			t.Fatalf("init help missing %s: %s", flag, output)
		}
	}
}
