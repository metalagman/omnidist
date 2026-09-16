package gem

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestCheckDependencyMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := CheckDependency(); err == nil || !strings.Contains(err.Error(), "gem executable not found") {
		t.Fatalf("CheckDependency() error = %v, want RubyGems installation guidance", err)
	}
}

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "1.2.3", want: "1.2.3"},
		{in: "1.2.3-dev.4.gabc123", want: "1.2.3.pre.dev.4.gabc123"},
		{in: "1.2.3-dev.4+gabc123", want: "1.2.3.pre.dev.4"},
		{in: "", wantErr: true},
	}

	for _, tc := range tests {
		got, err := NormalizeVersion(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("NormalizeVersion(%q) error = nil, want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeVersion(%q) error = %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGemPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target config.Target
		want   string
	}{
		{target: config.Target{OS: "darwin", Arch: "amd64"}, want: "x86_64-darwin"},
		{target: config.Target{OS: "darwin", Arch: "arm64"}, want: "arm64-darwin"},
		{target: config.Target{OS: "linux", Arch: "amd64"}, want: "x86_64-linux"},
		{target: config.Target{OS: "linux", Arch: "arm64", Variant: "musl"}, want: "aarch64-linux-musl"},
		{target: config.Target{OS: "windows", Arch: "amd64"}, want: "x64-mingw-ucrt"},
		{target: config.Target{OS: "windows", Arch: "amd64", Variant: "mingw32"}, want: "x64-mingw32"},
	}

	for _, tc := range tests {
		if got := gemPlatform(tc.target); got != tc.want {
			t.Fatalf("gemPlatform(%+v) = %q, want %q", tc.target, got, tc.want)
		}
	}
}

func TestGemspecContent(t *testing.T) {
	t.Parallel()

	dist := config.DistributionConfig{
		Package:       "omnidist",
		Registry:      "https://rubygems.org",
		RepositoryURL: "https://github.com/metalagman/omnidist",
		License:       "MIT",
		IncludeREADME: boolPtr(true),
	}
	content := gemspecContent(dist, "omnidist", "1.2.3", "x86_64-linux")
	for _, want := range []string{
		`spec.name = "omnidist"`,
		`spec.version = "1.2.3"`,
		`spec.platform = Gem::Platform.new("x86_64-linux")`,
		`spec.required_ruby_version = ">= 3.1"`,
		`"rubygems_mfa_required" => "true"`,
		`"source_code_uri" => "https://github.com/metalagman/omnidist"`,
		`spec.executables = ["omnidist"]`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("gemspecContent() missing %q\n---\n%s", want, content)
		}
	}
}

func TestGemspecContentAddsAllowedPushHostForCustomRegistry(t *testing.T) {
	t.Parallel()

	dist := config.DistributionConfig{
		Package:       "tool",
		Registry:      "https://gems.example.com",
		RepositoryURL: "https://example.com/tool",
		IncludeREADME: boolPtr(true),
	}
	content := gemspecContent(dist, "tool", "1.2.3", "x86_64-linux")
	if !strings.Contains(content, `"allowed_push_host" => "https://gems.example.com"`) {
		t.Fatalf("gemspecContent() missing allowed_push_host\n---\n%s", content)
	}
}

func TestPublishEnv(t *testing.T) {
	t.Run("token_requires_key", func(t *testing.T) {
		t.Setenv(gemHostAPIKeyEnv, "")
		t.Setenv(rubygemsAPIKeyEnv, "")
		_, err := publishEnv(config.DistributionConfig{PublishAuth: "token"}, PublishOptions{})
		if err == nil {
			t.Fatalf("publishEnv(token) error = nil, want error")
		}
	})

	t.Run("trusted_allows_missing_key", func(t *testing.T) {
		t.Setenv(gemHostAPIKeyEnv, "")
		t.Setenv(rubygemsAPIKeyEnv, "")
		env, err := publishEnv(config.DistributionConfig{PublishAuth: "trusted"}, PublishOptions{})
		if err != nil {
			t.Fatalf("publishEnv(trusted) error = %v", err)
		}
		if len(env) == 0 {
			t.Fatalf("publishEnv(trusted) env is empty")
		}
	})
}

func TestPublishProgressIdentifiesLastSuccessfulGem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command based failure simulation")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := config.DefaultConfig()
	cfg.Targets = []config.Target{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}
	if err := os.MkdirAll(filepath.Dir(paths.DistVersionPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.DistVersionPath, []byte("1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dist := cfg.Distributions["gem"]
	layout := layoutForConfig(cfg)
	artifacts := []string{
		gemArtifactPath(layout, dist, cfg.Targets[0], "1.2.3"),
		gemArtifactPath(layout, dist, cfg.Targets[1], "1.2.3"),
	}
	sort.Strings(artifacts)

	originalCommand := command
	t.Cleanup(func() { command = originalCommand })
	invocations := 0
	command = func(name string, args ...string) *exec.Cmd {
		invocations++
		if invocations == 2 {
			return exec.Command("sh", "-c", "exit 7")
		}
		return exec.Command("sh", "-c", "exit 0")
	}

	var progress bytes.Buffer
	err := Publish(cfg, PublishOptions{APIKey: "secret", Progress: &progress})
	if err == nil || !strings.Contains(err.Error(), filepath.Base(artifacts[1])) {
		t.Fatalf("Publish() error = %v, want second artifact context", err)
	}
	output := progress.String()
	if !strings.Contains(output, "Published: "+filepath.Base(artifacts[0])) {
		t.Fatalf("progress missing last successful artifact:\n%s", output)
	}
	if strings.Contains(output, "Published: "+filepath.Base(artifacts[1])) {
		t.Fatalf("failed artifact reported as published:\n%s", output)
	}
}

func TestVerifyGemArchive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.gem")
	writeFakeGem(t, goodPath, true)
	if err := verifyGemArchive(goodPath, "omnidist"); err != nil {
		t.Fatalf("verifyGemArchive(good) error = %v", err)
	}

	badPath := filepath.Join(dir, "bad.gem")
	writeFakeGem(t, badPath, false)
	if err := verifyGemArchive(badPath, "omnidist"); err == nil {
		t.Fatalf("verifyGemArchive(bad) error = nil, want error")
	}
}

func TestResolveStagedGemVersionFallsBack(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Runtime.WorkspaceDir = filepath.Join(dir, ".omnidist")
	t.Setenv("OMNIDIST_VERSION", "1.2.3")
	cfg.Version.Source = "env"
	got, err := resolveStagedGemVersion(cfg, paths.NewLayout(cfg.EffectiveWorkspaceDir()))
	if err != nil {
		t.Fatalf("resolveStagedGemVersion() error = %v", err)
	}
	if got != "1.2.3" {
		t.Fatalf("resolveStagedGemVersion() = %q, want %q", got, "1.2.3")
	}
}

func writeFakeGem(t *testing.T, path string, includeMetadata bool) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", path, err)
	}
	defer file.Close()

	tw := tarWriter(t, file)
	defer tw.Close()

	if includeMetadata {
		writeTarEntry(t, tw, "metadata.gz", []byte("meta"))
	}
	var data bytes.Buffer
	gzw := gzipWriter(t, &data)
	dataTar := tarWriter(t, gzw)
	writeTarEntry(t, dataTar, "exe/omnidist", []byte("wrapper"))
	writeTarEntry(t, dataTar, "libexec/omnidist", []byte("binary"))
	if err := dataTar.Close(); err != nil {
		t.Fatalf("close data tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close data gzip: %v", err)
	}
	writeTarEntry(t, tw, "data.tar.gz", data.Bytes())
}
