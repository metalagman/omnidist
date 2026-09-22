package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurationReferenceCoversPersistedFields(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	fields := []string{
		"tool.name", "tool.main", "version.source", "version.file", "version.fixed",
		"readme-path", "targets[].os", "targets[].arch", "targets[].variant",
		"build.ldflags", "build.tags", "build.cgo", "enabled-distributions", "distributions",
		"package", "aliases", "registry", "access", "publish-auth", "repository-url", "license",
		"keywords", "index-url", "linux-tag", "include-readme",
	}
	for _, field := range fields {
		if !strings.Contains(doc, "`"+field+"`") {
			t.Errorf("configuration reference does not document %s", field)
		}
	}
}

func TestDualNPMMetaPackageDocumentation(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{
			path: filepath.Join("..", "..", "README.md"),
			want: []string{"aliases:", "npm install -g omnidist", "npm install -g @omnidist/omnidist"},
		},
		{
			path: filepath.Join("..", "..", "docs", "configuration.md"),
			want: []string{"`aliases`", "primary package followed by aliases", "shared platform package set"},
		},
		{
			path: filepath.Join("..", "..", "docs", "releases.md"),
			want: []string{"every meta package and platform package", "primary package followed by aliases", "partial npm release"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			data, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			doc := string(data)
			for _, want := range tt.want {
				if !strings.Contains(doc, want) {
					t.Errorf("%s missing dual npm package documentation %q", tt.path, want)
				}
			}
		})
	}
}

func TestRepositoryConfigUsesDualNPMMetaPackages(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", ".omnidist", "omnidist.yaml"))
	if err != nil {
		t.Fatalf("Load(repository config) error = %v", err)
	}
	npmDist, err := cfg.RequireNPM()
	if err != nil {
		t.Fatalf("RequireNPM() error = %v", err)
	}
	if npmDist.Package != "omnidist" {
		t.Errorf("npm package = %q, want omnidist", npmDist.Package)
	}
	if len(npmDist.Aliases) != 1 || npmDist.Aliases[0] != "@omnidist/omnidist" {
		t.Errorf("npm aliases = %q, want [@omnidist/omnidist]", npmDist.Aliases)
	}
	if npmDist.PlatformPackage != "@omnidist/omnidist" {
		t.Errorf("npm platform-package = %q, want @omnidist/omnidist", npmDist.PlatformPackage)
	}
}

func TestConfigurationReferenceProfileExampleLoads(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(data), "```yaml\n")
	if start < 0 {
		t.Fatal("configuration reference has no YAML example")
	}
	start += len("```yaml\n")
	end := strings.Index(string(data)[start:], "\n```")
	if end < 0 {
		t.Fatal("configuration reference YAML fence is not closed")
	}
	example := string(data)[start : start+end]
	if strings.Contains(example, "enabled-distributions:") {
		t.Fatalf("canonical profiles example contains legacy enabled-distributions:\n%s", example)
	}
	path := filepath.Join(t.TempDir(), "omnidist.yaml")
	if err := os.WriteFile(path, []byte(example), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWithProfile(path, DefaultProfileName); err != nil {
		t.Fatalf("documented profiles example does not load: %v", err)
	}
}

func TestConfigurationReferenceRejectsStaleDistributionSemantics(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, stale := range []string{
		"Missing entries receive compatibility defaults",
		"generated configs emit all three",
		"compatibility defaults are `@omnidist/omnidist`",
	} {
		if strings.Contains(doc, stale) {
			t.Fatalf("configuration reference contains stale claim %q", stale)
		}
	}
	for _, current := range []string{
		"Section presence enables npm, uv, or gem",
		"loading never invents an identity",
		"selector that names a missing section now fails",
	} {
		if !strings.Contains(doc, current) {
			t.Fatalf("configuration reference missing current semantic %q", current)
		}
	}
}

func TestReadmeKeepsSafeReleaseOrderAndReferenceLinks(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	sectionStart := strings.Index(doc, "## Safe quick start")
	sectionEnd := strings.Index(doc, "## Release flow")
	if sectionStart < 0 || sectionEnd <= sectionStart {
		t.Fatal("README safe quick-start section is missing")
	}
	quickstart := doc[sectionStart:sectionEnd]
	positions := []int{
		strings.Index(quickstart, "omnidist build"),
		strings.Index(quickstart, "omnidist stage"),
		strings.Index(quickstart, "omnidist verify"),
		strings.Index(quickstart, "omnidist publish --dry-run"),
	}
	for i, position := range positions {
		if position < 0 || (i > 0 && position <= positions[i-1]) {
			t.Fatalf("README safe quick-start order is invalid: %v", positions)
		}
	}
	for _, link := range []string{"docs/configuration.md", "docs/releases.md", "docs/targets.md"} {
		if !strings.Contains(doc, link) {
			t.Errorf("README missing reference link %s", link)
		}
	}
}
