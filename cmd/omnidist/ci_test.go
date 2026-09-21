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
		`FORCE_JAVASCRIPT_ACTIONS_TO_NODE24: "true"`,
		`prepare:`,
		`publish_npm:`,
		`publish_uv:`,
		`release:`,
		`needs: prepare`,
		`run: go run ./cmd/omnidist build`,
		`run: go run ./cmd/omnidist stage --only 'npm,uv,gem'`,
		`run: go run ./cmd/omnidist verify --only 'npm,uv,gem'`,
		`run: tar -czf omnidist-staged.tgz .omnidist`,
		`path: .omnidist/dist/**/*`,
		`if-no-files-found: error`,
		`run: go run ./cmd/omnidist npm publish`,
		`run: go run ./cmd/omnidist uv publish`,
		`actions/setup-node@v6`,
		`node-version: '24'`,
		`sha256sum * > checksums.txt`,
		`uses: softprops/action-gh-release@v2`,
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

func TestCICommandDryRunPrintsWorkflow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	output, err := executeCommand("ci", "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(ci --dry-run) error = %v, output=%s", err, output)
	}

	// Verify workflow file was NOT created
	if _, err := os.Stat(workflow.DefaultCIWorkflowPath); !os.IsNotExist(err) {
		t.Fatalf("workflow file created in dry-run mode")
	}

	// Verify output contains workflow content
	for _, want := range []string{
		`name: omnidist-release`,
		`on:`,
		`push:`,
		`tags:`,
		`- "v*"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry-run output missing %q: %s", want, output)
		}
	}
}

func TestCICommandDryRunRespectsConfiguredDistributions(t *testing.T) {
	for _, selected := range []string{"npm", "uv", "gem"} {
		t.Run(selected, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			cfg := config.DefaultConfig()
			switch selected {
			case "npm":
				cfg.Distributions.UV = nil
				cfg.Distributions.Gem = nil
			case "uv":
				cfg.Distributions.NPM = nil
				cfg.Distributions.Gem = nil
			case "gem":
				cfg.Distributions.NPM = nil
				cfg.Distributions.UV = nil
			}
			if err := config.Save(cfg, paths.ConfigPath); err != nil {
				t.Fatal(err)
			}

			output, err := executeCommand("ci", "--dry-run")
			if err != nil {
				t.Fatalf("executeCommand(ci --dry-run) error = %v", err)
			}
			for _, want := range []string{"stage --only '" + selected + "'", "publish_" + selected + ":"} {
				if !strings.Contains(output, want) {
					t.Fatalf("%s-only workflow missing %q: %s", selected, want, output)
				}
			}
			for _, other := range []string{"npm", "uv", "gem"} {
				if other != selected && strings.Contains(output, "publish_"+other+":") {
					t.Fatalf("%s-only workflow contains publish_%s: %s", selected, other, output)
				}
			}
		})
	}
}

func TestCICommandDryRunInfersSelectedDistributionsFromLegacyConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := config.DefaultConfig()
	cfg.Distributions = config.DistributionConfigs{
		NPM: cfg.Distributions.NPM,
	}
	if err := config.Save(cfg, paths.ConfigPath); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand("ci", "--dry-run")
	if err != nil {
		t.Fatalf("executeCommand(ci --dry-run) error = %v", err)
	}
	for _, want := range []string{"stage --only 'npm'", "publish_npm:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("legacy npm-only workflow missing %q: %s", want, output)
		}
	}
	for _, unwanted := range []string{"publish_uv:", "publish_gem:"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("legacy npm-only workflow contains %q: %s", unwanted, output)
		}
	}
}
