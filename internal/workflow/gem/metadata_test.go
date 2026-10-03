package gem

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestGemspecConfiguredMetadata(t *testing.T) {
	t.Parallel()
	dist := config.GemDistributionConfig{
		Package:     "omnidist",
		Description: `Go's CLI \path #{raise "do not execute"}`,
		Keywords:    []string{`go's`, `cli\tools`, `#{raise "do not execute"}`},
		License:     `MIT #{raise "do not execute"}`,
	}
	content := gemspecContent(dist, "omnidist", "1.2.3", "x86_64-linux")
	for _, want := range []string{
		`spec.summary = 'Go\'s CLI \\path #{raise "do not execute"}'`,
		`spec.description = 'Go\'s CLI \\path #{raise "do not execute"}. Includes prebuilt binaries for the current platform.'`,
		`spec.licenses = ['MIT #{raise "do not execute"}']`,
		`"keywords" => 'go\'s,cli\\tools,#{raise "do not execute"}'`,
		`"rubygems_mfa_required" => "true"`,
		`"source_code_uri" => "https://example.invalid"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("gemspec missing literal %q:\n%s", want, content)
		}
	}
}

func TestGemspecMetadataFallback(t *testing.T) {
	t.Parallel()
	content := gemspecContent(config.GemDistributionConfig{Package: "omnidist"}, "omnidist", "1.2.3", "x86_64-linux")
	for _, want := range []string{
		`spec.summary = "Prebuilt omnidist CLI packaged by omnidist"`,
		`spec.description = "Prebuilt omnidist CLI packaged by omnidist with platform-specific binaries"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("gemspec missing fallback %q", want)
		}
	}
	if strings.Contains(content, `"keywords"`) {
		t.Fatal("fallback gemspec unexpectedly contains keywords metadata")
	}
}

func TestVerifyConfiguredGemMetadata(t *testing.T) {
	const valid = "--- !ruby/object:Gem::Specification\nsummary: Distribute Go binaries\ndescription: Distribute Go binaries. Includes prebuilt binaries for the current platform.\nlicenses: [MIT]\nmetadata:\n  keywords: golang,cli\n"
	for _, tc := range []struct {
		name, metadata, wantError string
	}{
		{"matching", valid, ""},
		{"summary changed", strings.Replace(valid, "summary: Distribute Go binaries", "summary: Old description", 1), "summary mismatch"},
		{"summary missing", strings.Replace(valid, "summary: Distribute Go binaries\n", "", 1), "summary mismatch"},
		{"description changed", strings.Replace(valid, "description: Distribute Go binaries", "description: Old description", 1), "description mismatch"},
		{"keywords changed", strings.Replace(valid, "golang,cli", "golang,old", 1), "keywords mismatch"},
		{"keywords missing", strings.Replace(valid, "  keywords: golang,cli\n", "", 1), "keywords mismatch"},
		{"license changed", strings.Replace(valid, "licenses: [MIT]", "licenses: [Apache-2.0]", 1), "licenses mismatch"},
		{"malformed YAML", "summary: [unterminated\n", "parse gem metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, layout := windowsGemConfig(t)
			cfg.Description = "Distribute Go binaries"
			cfg.Keywords = []string{"golang", "cli"}
			cfg.License = "MIT"
			cfg.Distributions.Gem.License = ""
			for _, target := range cfg.Targets {
				writeMetadataGem(t, gemArtifactPath(layout, *cfg.Distributions.Gem, target, "1.2.3"), tc.metadata)
			}
			result := Verify(cfg)
			if tc.wantError == "" {
				if !result.Valid {
					t.Fatalf("Verify() errors = %v", result.Errors)
				}
			} else if result.Valid || !strings.Contains(strings.Join(result.Errors, "\n"), tc.wantError) {
				t.Fatalf("Verify() = %#v, want %s", result, tc.wantError)
			}
		})
	}
}

func TestVerifyGemClearedMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "description: Project\nkeywords: [project]\nlicense: Apache-2.0\ntool:\n  name: omnidist\ndistributions:\n  gem:\n    package: omnidist\n    description: ''\n    keywords: []\n    license: ''\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	dist, err := cfg.RequireGem()
	if err != nil {
		t.Fatal(err)
	}
	valid := "summary: Prebuilt omnidist CLI packaged by omnidist\ndescription: Prebuilt omnidist CLI packaged by omnidist with platform-specific binaries\nlicenses: [MIT]\nmetadata: {}\n"
	for _, tc := range []struct{ data, want string }{
		{valid, ""},
		{strings.Replace(valid, "metadata: {}", "metadata: {keywords: project}", 1), "keywords mismatch"},
		{strings.Replace(valid, "summary: Prebuilt omnidist CLI packaged by omnidist", "summary: Project", 1), "summary mismatch"},
		{strings.Replace(valid, "licenses: [MIT]", "licenses: [Apache-2.0]", 1), "licenses mismatch"},
	} {
		var zipped bytes.Buffer
		gz := gzipWriter(t, &zipped)
		if _, err := gz.Write([]byte(tc.data)); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		err := verifyConfiguredGemMetadata(&zipped, dist, "omnidist")
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("verify cleared metadata error = %v, want %q", err, tc.want)
		}
	}
}

func TestConfiguredGemspecStrictBuild(t *testing.T) {
	gem, err := exec.LookPath("gem")
	if err != nil {
		t.Skip("native RubyGems not installed")
	}
	for _, license := range []string{"MIT", "MIT OR Apache-2.0"} {
		t.Run(license, func(t *testing.T) {
			dir := t.TempDir()
			for _, sub := range []string{"exe", "libexec"} {
				if err := os.Mkdir(filepath.Join(dir, sub), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, sub, "omnidist"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			dist := config.GemDistributionConfig{Package: "omnidist", Description: "Distribute Go binaries", Keywords: []string{"golang", "cli"}, License: license}
			if err := os.WriteFile(filepath.Join(dir, "omnidist.gemspec"), []byte(gemspecContent(dist, "omnidist", "1.2.3", "x86_64-linux")), 0644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(gem, "build", "omnidist.gemspec", "--strict")
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			if license == "MIT" && err != nil {
				t.Fatalf("strict gem build: %v\n%s", err, output)
			}
			if license != "MIT" && err == nil {
				t.Fatal("strict gem build accepted unsupported compound license")
			}
		})
	}
}

func writeMetadataGem(t *testing.T, path, metadata string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	tw := tarWriter(t, file)
	defer tw.Close()
	var meta bytes.Buffer
	metaZip := gzipWriter(t, &meta)
	if _, err := metaZip.Write([]byte(metadata)); err != nil {
		t.Fatal(err)
	}
	if err := metaZip.Close(); err != nil {
		t.Fatal(err)
	}
	writeTarEntry(t, tw, "metadata.gz", meta.Bytes())
	var data bytes.Buffer
	dataZip := gzipWriter(t, &data)
	dataTar := tarWriter(t, dataZip)
	writeTarEntry(t, dataTar, "exe/omnidist", []byte("wrapper"))
	writeTarEntry(t, dataTar, "libexec/omnidist", []byte("binary"))
	if err := dataTar.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dataZip.Close(); err != nil {
		t.Fatal(err)
	}
	writeTarEntry(t, tw, "data.tar.gz", data.Bytes())
}
