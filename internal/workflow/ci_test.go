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
		`prepare:`,
		`publish_npm:`,
		`publish_uv:`,
		`release:`,
		`needs: prepare`,
		`NPM_PUBLISH_TOKEN: ${{ secrets.NPM_PUBLISH_TOKEN }}`,
		`UV_PUBLISH_TOKEN: ${{ secrets.UV_PUBLISH_TOKEN }}`,
		`run: npm install -g '@omnidist/omnidist@0.1.9'`,
		`run: omnidist build`,
		`run: omnidist stage`,
		`run: omnidist verify`,
		`run: tar -czf omnidist-staged.tgz .omnidist`,
		`name: dist`,
		`path: .omnidist/dist/**/*`,
		`if-no-files-found: error`,
		`uses: actions/upload-artifact@v4`,
		`uses: actions/download-artifact@v4`,
		`run: omnidist npm publish`,
		`run: omnidist uv publish`,
		`permissions:`,
		`contents: write`,
		`path: .omnidist/dist`,
		`merge-multiple: true`,
		`run: |`,
		`find .omnidist/dist -type f ! -name VERSION -print0 | sort -z`,
		`sha256sum * > checksums.txt`,
		`uses: softprops/action-gh-release@v2`,
		`release-assets/*`,
		`generate_release_notes: true`,
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

	if !strings.Contains(content, `npm install -g '@omnidist/omnidist@latest'`) {
		t.Fatalf("workflow content = %q, want latest install version", content)
	}
}

func TestGenerateGitHubReleaseWorkflowNilConfig(t *testing.T) {
	t.Parallel()

	if _, err := GenerateGitHubReleaseWorkflow(nil, CIWorkflowOptions{}); err == nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow(nil) error = nil, want error")
	}
}

func TestGenerateGitHubReleaseWorkflowQuotesVersionRange(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	content, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{
		NPXVersion: ">=0.1.0 <2.0.0 || 3.0.0",
	})
	if err != nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = %v", err)
	}

	want := `run: npm install -g '@omnidist/omnidist@>=0.1.0 <2.0.0 || 3.0.0'`
	if !strings.Contains(content, want) {
		t.Fatalf("workflow content missing quoted range %q\n---\n%s", want, content)
	}
}

func TestGenerateGitHubReleaseWorkflowRejectsInvalidNPXPackage(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	_, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{
		NPXPackage: "@omnidist/omnidist;echo pwned",
	})
	if err == nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid npx package name") {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = %v, want invalid package error", err)
	}
}

func TestGenerateGitHubReleaseWorkflowRejectsInvalidNPXVersion(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	_, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{
		NPXVersion: "latest; echo pwned",
	})
	if err == nil {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid npx version spec") {
		t.Fatalf("GenerateGitHubReleaseWorkflow() error = %v, want invalid version error", err)
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

func TestShellQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "''"},
		{input: "simple", want: "'simple'"},
		{input: "don't", want: "'don'\"'\"'t'"},
	}

	for _, tc := range tests {
		got := shellQuote(tc.input)
		if got != tc.want {
			t.Fatalf("shellQuote(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestWriteGitHubReleaseWorkflowErrors(t *testing.T) {
	t.Parallel()

	t.Run("empty_path", func(t *testing.T) {
		err := WriteGitHubReleaseWorkflow("  ", "content", false)
		if err == nil || !strings.Contains(err.Error(), "workflow path is empty") {
			t.Fatalf("WriteGitHubReleaseWorkflow(empty path) error = %v", err)
		}
	})

	t.Run("empty_content", func(t *testing.T) {
		err := WriteGitHubReleaseWorkflow("path", "  ", false)
		if err == nil || !strings.Contains(err.Error(), "workflow content is empty") {
			t.Fatalf("WriteGitHubReleaseWorkflow(empty content) error = %v", err)
		}
	})

	t.Run("mkdir_fail", func(t *testing.T) {
		dir := t.TempDir()
		// Create a file where a directory should be
		path := filepath.Join(dir, "file")
		if err := os.WriteFile(path, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}
		workflowPath := filepath.Join(path, "workflow.yml")

		err := WriteGitHubReleaseWorkflow(workflowPath, "name: test\n", true)
		if err == nil || !strings.Contains(err.Error(), "create workflow directory") {
			t.Fatalf("WriteGitHubReleaseWorkflow(mkdir fail) error = %v", err)
		}
	})

	t.Run("write_fail", func(t *testing.T) {
		dir := t.TempDir()
		// Create a directory where the file should be
		path := filepath.Join(dir, "workflow.yml")
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}

		err := WriteGitHubReleaseWorkflow(path, "name: test\n", true)
		if err == nil || !strings.Contains(err.Error(), "write workflow file") {
			t.Fatalf("WriteGitHubReleaseWorkflow(write fail) error = %v", err)
		}
	})
}
