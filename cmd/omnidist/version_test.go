package main

import (
	"strings"
	"testing"
)

func TestVersionCommandOutput(t *testing.T) {
	output, err := executeCommand("version")
	if err != nil {
		t.Fatalf("executeCommand(version) error = %v", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		t.Fatalf("version output is empty")
	}
	if !strings.Contains(output, "commit:") {
		t.Fatalf("version output = %q, want commit marker", output)
	}
}
