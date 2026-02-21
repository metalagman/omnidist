package main

import (
	"os"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow"
)

func TestCICommandCreatesWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	output, err := executeCommand("ci")
	if err != nil {
		t.Fatalf("executeCommand(ci) error = %v, output=%s", err, output)
	}
	if !strings.Contains(output, workflow.DefaultCIWorkflowPath) {
		t.Fatalf("ci output missing workflow path: %s", output)
	}

	data, err := os.ReadFile(workflow.DefaultCIWorkflowPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", workflow.DefaultCIWorkflowPath, err)
	}
	content := string(data)
	for _, want := range []string{
		`tags:`,
		`- "v*"`,
		`run: npx @omnidist/omnidist@`,
		`build`,
		`stage`,
		`verify`,
		`publish`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("workflow missing %q: %s", want, content)
		}
	}
}

func TestCICommandFailsWhenWorkflowExistsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	if _, err := executeCommand("ci"); err != nil {
		t.Fatalf("executeCommand(ci) first run error = %v", err)
	}

	_, err := executeCommand("ci")
	if err == nil {
		t.Fatalf("executeCommand(ci) second run error = nil, want error")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("executeCommand(ci) error = %v, want --force hint", err)
	}
}

func TestCICommandForceOverwritesWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	if err := os.MkdirAll(".github/workflows", 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(workflow.DefaultCIWorkflowPath, []byte("name: old\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if _, err := executeCommand("ci", "--force"); err != nil {
		t.Fatalf("executeCommand(ci --force) error = %v", err)
	}

	data, err := os.ReadFile(workflow.DefaultCIWorkflowPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "name: omnidist-release") {
		t.Fatalf("workflow content not overwritten: %s", string(data))
	}
}
