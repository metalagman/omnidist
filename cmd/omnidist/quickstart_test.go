package main

import (
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestQuickstartWithoutConfigPrintsInitFlow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	output, err := executeCommand("quickstart")
	if err != nil {
		t.Fatalf("executeCommand(quickstart) error = %v", err)
	}

	for _, want := range []string{"omnidist init", "omnidist build", "omnidist ci"} {
		if !strings.Contains(output, want) {
			t.Fatalf("quickstart output missing %q: %s", want, output)
		}
	}
}

func TestQuickstartWithEnvVersionHintsVar(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := config.DefaultConfig()
	cfg.Version.Source = "env"
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	output, err := executeCommand("quickstart")
	if err != nil {
		t.Fatalf("executeCommand(quickstart) error = %v", err)
	}
	if !strings.Contains(output, shared.EnvVersionName) {
		t.Fatalf("quickstart output missing env version name %q: %s", shared.EnvVersionName, output)
	}
}
