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
		{target: config.Target{OS: "windows", Arch: "amd64", Variant: "mingw-ucrt"}, want: "x64-mingw-ucrt"},
		{target: config.Target{OS: "windows", Arch: "amd64", Variant: "mingw32"}, want: "x64-mingw32"},
		{target: config.Target{OS: "windows", Arch: "amd64", Variant: "custom"}, want: "x64-custom"},
		{target: config.Target{OS: "windows", Arch: "arm64"}, want: "aarch64-mingw-ucrt"},
		{target: config.Target{OS: "windows", Arch: "arm64", Variant: "mingw-ucrt"}, want: "aarch64-mingw-ucrt"},
		{target: config.Target{OS: "windows", Arch: "arm64", Variant: "mingw32"}, want: "aarch64-mingw32"},
		{target: config.Target{OS: "windows", Arch: "arm64", Variant: "custom"}, want: "aarch64-custom"},
	}

	for _, tc := range tests {
		if got := gemPlatform(tc.target); got != tc.want {
			t.Errorf("gemPlatform(%+v) = %q, want %q", tc.target, got, tc.want)
		}
	}
}

func TestDocumentedGemPlatforms(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "targets.md"))
	if err != nil {
		t.Fatal(err)
	}
	foundWindowsARM64 := false
	for _, line := range strings.Split(string(data), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) != 7 {
			continue
		}
		targetParts := strings.Split(strings.Trim(cells[1], " `"), "/")
		if len(targetParts) != 2 {
			continue
		}
		target := config.Target{OS: targetParts[0], Arch: targetParts[1]}
		if target.OS == "windows" && target.Arch == "arm64" {
			foundWindowsARM64 = true
		}
		want := strings.Trim(cells[5], " `")
		if got := gemPlatform(target); got != want {
			t.Errorf("documented %s/%s platform = %q, mapper produces %q", target.OS, target.Arch, want, got)
		}
	}
	if !foundWindowsARM64 {
		t.Error("target reference has no Windows ARM64 platform mapping")
	}
}

func TestStagePreservesWindowsArchitectures(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command based gem build simulation")
	}
	cfg, layout := windowsGemConfig(t)
	for _, target := range cfg.Targets {
		binary := filepath.Join(layout.DistDir, "windows", target.Arch, "omnidist.exe")
		if err := os.MkdirAll(filepath.Dir(binary), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(binary, []byte(target.Arch), 0755); err != nil {
			t.Fatal(err)
		}
	}

	originalCommand := command
	t.Cleanup(func() { command = originalCommand })
	var outputs []string
	command = func(name string, args ...string) *exec.Cmd {
		if name != "gem" || len(args) != 5 || args[0] != "build" || args[1] != "omnidist.gemspec" || args[2] != "--strict" || args[3] != "--output" {
			t.Fatalf("unexpected gem build command: %s %v", name, args)
		}
		outputs = append(outputs, filepath.Base(args[4]))
		writeFakeGem(t, args[4], true)
		return exec.Command("sh", "-c", "exit 0")
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatalf("Stage() error = %v", err)
	}

	platforms := []string{"x64-mingw-ucrt", "aarch64-mingw-ucrt"}
	if len(outputs) != len(platforms) {
		t.Fatalf("gem build count = %d, want %d", len(outputs), len(platforms))
	}
	for i, platform := range platforms {
		wantArtifact := "omnidist-1.2.3-" + platform + ".gem"
		if outputs[i] != wantArtifact {
			t.Errorf("gem build output = %q, want %q", outputs[i], wantArtifact)
		}
		if _, err := os.Stat(filepath.Join(layout.GemPkgDir, wantArtifact)); err != nil {
			t.Errorf("staged artifact %s: %v", wantArtifact, err)
		}
		stagingDir := filepath.Join(layout.GemBuildDir, platform)
		binary, err := os.ReadFile(filepath.Join(stagingDir, "libexec", "omnidist.exe"))
		if err != nil {
			t.Errorf("read %s binary: %v", platform, err)
			continue
		}
		if string(binary) != cfg.Targets[i].Arch {
			t.Errorf("%s binary = %q, want %q", platform, binary, cfg.Targets[i].Arch)
		}
		gemspec, err := os.ReadFile(filepath.Join(stagingDir, "omnidist.gemspec"))
		if err != nil {
			t.Fatal(err)
		}
		if want := `spec.platform = Gem::Platform.new("` + platform + `")`; !strings.Contains(string(gemspec), want) {
			t.Errorf("%s gemspec missing %q", platform, want)
		}
	}
}

func TestVerifyRequiresWindowsARM64Artifact(t *testing.T) {
	t.Parallel()
	cfg, layout := windowsGemConfig(t)
	writeFakeGem(t, filepath.Join(layout.GemPkgDir, "omnidist-1.2.3-x64-mingw-ucrt.gem"), true)
	armArtifact := filepath.Join(layout.GemPkgDir, "omnidist-1.2.3-aarch64-mingw-ucrt.gem")

	result := Verify(cfg)
	if result.Valid || len(result.Errors) != 1 || !strings.Contains(result.Errors[0], armArtifact) {
		t.Fatalf("Verify() = %+v, want missing ARM64 artifact %s", result, armArtifact)
	}
	writeFakeGem(t, armArtifact, true)
	if result := Verify(cfg); !result.Valid {
		t.Fatalf("Verify() with both Windows artifacts = %+v", result)
	}
}

func TestPublishDryRunListsWindowsArchitectures(t *testing.T) {
	cfg, _ := windowsGemConfig(t)
	originalCommand := command
	t.Cleanup(func() { command = originalCommand })
	command = func(name string, args ...string) *exec.Cmd {
		t.Fatalf("dry-run invoked external command: %s %v", name, args)
		return nil
	}
	var output bytes.Buffer
	if err := Publish(cfg, PublishOptions{DryRun: true, Stdout: &output}); err != nil {
		t.Fatalf("Publish(dry-run) error = %v", err)
	}
	for _, artifact := range []string{
		"omnidist-1.2.3-x64-mingw-ucrt.gem",
		"omnidist-1.2.3-aarch64-mingw-ucrt.gem",
	} {
		if count := strings.Count(output.String(), artifact); count != 1 {
			t.Errorf("dry-run lists %s %d times, want once:\n%s", artifact, count, output.String())
		}
	}
}

func windowsGemConfig(t *testing.T) (*config.Config, paths.Layout) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Tool.Name = "omnidist"
	cfg.Distributions.Gem.Package = "omnidist"
	cfg.Distributions.Gem.IncludeREADME = boolPtr(false)
	cfg.Runtime.WorkspaceDir = t.TempDir()
	cfg.Targets = []config.Target{
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
	}
	layout := layoutForConfig(cfg)
	if err := os.MkdirAll(layout.DistDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.DistVersionPath, []byte("1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.GemPkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	return cfg, layout
}

func TestGemspecContent(t *testing.T) {
	t.Parallel()

	dist := config.GemDistributionConfig{
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

	dist := config.GemDistributionConfig{
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
		_, err := publishEnv(config.GemDistributionConfig{PublishAuth: "token"}, PublishOptions{})
		if err == nil {
			t.Fatalf("publishEnv(token) error = nil, want error")
		}
	})

	t.Run("trusted_allows_missing_key", func(t *testing.T) {
		t.Setenv(gemHostAPIKeyEnv, "")
		t.Setenv(rubygemsAPIKeyEnv, "")
		env, err := publishEnv(config.GemDistributionConfig{PublishAuth: "trusted"}, PublishOptions{})
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

	dist := *cfg.Distributions.Gem
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
	if err := verifyGemArchive(goodPath, "omnidist", config.GemDistributionConfig{}); err != nil {
		t.Fatalf("verifyGemArchive(good) error = %v", err)
	}

	badPath := filepath.Join(dir, "bad.gem")
	writeFakeGem(t, badPath, false)
	if err := verifyGemArchive(badPath, "omnidist", config.GemDistributionConfig{}); err == nil {
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
