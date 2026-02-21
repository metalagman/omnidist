package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestGenerateGitHubReleaseWorkflow(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	content, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{
		NPXVersion: "0.1.9",
	})
	if err != nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = %v", err)
	}

	for _, want := range []string{
		`name: omnidist-release`,
		`tags:`,
		`- "v*"`,
		`NPM_PUBLISH_TOKEN: ${{ secrets.NPM_PUBLISH_TOKEN }}`,
		`UV_PUBLISH_TOKEN: ${{ secrets.UV_PUBLISH_TOKEN }}`,
		`run: npx @omnidist/omnidist@0.1.9 build`,
		`run: npx @omnidist/omnidist@0.1.9 stage`,
		`run: npx @omnidist/omnidist@0.1.9 verify`,
		`run: npx @omnidist/omnidist@0.1.9 publish`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("workflow content missing %q\n---\n%s", want, content)
		}
	}
}

func TestGenerateGitHubReleaseWorkflowDefaultsToLatest(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	content, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{})
	if err != nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = %v", err)
	}

	if !strings.Contains(content, "npx @omnidist/omnidist@latest build") {
		t.Fatalf("workflow content = %q, want latest npx version", content)
	}
}

func TestGenerateGitHubReleaseWorkflowNilConfig(t *testing.T) {
	t.Parallel()

	if _, err := GenerateGitHubReleaseWorkflow(nil, CIWorkflowOptions{}); err == nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow(nil) error = nil, want error")
	}
}

func TestWriteGitHubReleaseWorkflow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".github/workflows/omnidist-release.yml")
	content := "name: test\n"

	if err := WriteGitHubReleaseWorkflow(path, content, false); err != nil {
		t.Fatalf("WriteGitHubReleaseWorkflow() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(data) != content {
		t.Fatalf("workflow content = %q, want %q", string(data), content)
	}
}

func TestWriteGitHubReleaseWorkflowExistingWithoutForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".github/workflows/omnidist-release.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("name: old\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	err := WriteGitHubReleaseWorkflow(path, "name: new\n", false)
	if err == nil {
		t.Fatalf("WriteGitHubReleaseWorkflow() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("WriteGitHubReleaseWorkflow() error = %v, want --force hint", err)
	}
}

func TestWriteGitHubReleaseWorkflowExistingWithForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".github/workflows/omnidist-release.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("name: old\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := WriteGitHubReleaseWorkflow(path, "name: new\n", true); err != nil {
		t.Fatalf("WriteGitHubReleaseWorkflow(..., force=true) error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(data) != "name: new\n" {
		t.Fatalf("workflow content = %q, want %q", string(data), "name: new\n")
	}
}
