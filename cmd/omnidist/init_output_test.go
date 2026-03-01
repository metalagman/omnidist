package main

import (
	"strings"
	"testing"
)

func TestInitPrintsNextSteps(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

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
		"omnidist publish",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("init output missing %q: %s", want, output)
		}
	}
}
