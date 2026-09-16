package gem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

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

	gzw := gzipWriter(t, file)
	defer gzw.Close()
	tw := tarWriter(t, gzw)
	defer tw.Close()

	if includeMetadata {
		writeTarEntry(t, tw, "metadata.gz", []byte("meta"))
	}
	writeTarEntry(t, tw, "data.tar.gz", []byte("data"))
}
