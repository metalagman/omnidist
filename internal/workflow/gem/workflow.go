package gem

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

const (
	defaultGemRegistry = "https://rubygems.org"
	gemHostAPIKeyEnv   = "GEM_HOST_API_KEY"
	gemHostOTPCodeEnv  = "GEM_HOST_OTP_CODE"
	rubygemsAPIKeyEnv  = "RUBYGEMS_API_KEY"
)

var gemNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var (
	lookPath = exec.LookPath
	command  = exec.Command
)

type StageOptions struct {
	Dev bool
}

type PublishOptions struct {
	DryRun   bool
	Host     string
	APIKey   string
	OTP      string
	Stdout   io.Writer
	Stderr   io.Writer
	Progress io.Writer
}

type VerificationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func CheckDependency() error {
	if _, err := lookPath("gem"); err != nil {
		return fmt.Errorf("gem executable not found in PATH. Install RubyGems/Ruby and retry")
	}
	return nil
}

// PreflightPublish validates local artifacts, tooling, and credentials without uploading gems.
func PreflightPublish(cfg *config.Config, opts PublishOptions) error {
	if err := CheckDependency(); err != nil {
		return err
	}
	result := Verify(cfg)
	if !result.Valid {
		return fmt.Errorf("staged artifact verification failed: %s", strings.Join(result.Errors, "; "))
	}
	dist, err := gemDistribution(cfg)
	if err != nil {
		return err
	}
	if _, err := publishEnv(dist, opts); err != nil {
		return err
	}
	return nil
}

func Stage(cfg *config.Config, opts StageOptions) error {
	dist, err := gemDistribution(cfg)
	if err != nil {
		return err
	}
	layout := layoutForConfig(cfg)

	version, err := resolveGemVersion(cfg, opts.Dev)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(layout.GemBuildDir); err != nil {
		return fmt.Errorf("clean gem build directory: %w", err)
	}
	if err := os.RemoveAll(layout.GemPkgDir); err != nil {
		return fmt.Errorf("clean gem package directory: %w", err)
	}
	if err := os.MkdirAll(layout.GemBuildDir, 0755); err != nil {
		return fmt.Errorf("create gem build directory: %w", err)
	}
	if err := os.MkdirAll(layout.GemPkgDir, 0755); err != nil {
		return fmt.Errorf("create gem package directory: %w", err)
	}

	for _, target := range cfg.Targets {
		if err := stageGemForTarget(layout, cfg, dist, target, version); err != nil {
			return fmt.Errorf("stage gem for %s/%s: %w", target.OS, target.Arch, err)
		}
	}
	return nil
}

func Verify(cfg *config.Config) *VerificationResult {
	result := &VerificationResult{Valid: true}
	dist, err := gemDistribution(cfg)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	layout := layoutForConfig(cfg)
	version, err := resolveStagedGemVersion(cfg, layout)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	for _, target := range cfg.Targets {
		artifact := gemArtifactPath(layout, dist, target, version)
		if _, err := os.Stat(artifact); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("missing gem artifact: %s", artifact))
			continue
		}
		if err := verifyGemArchive(artifact, cfg.Tool.Name); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}
	return result
}

func Publish(cfg *config.Config, opts PublishOptions) error {
	dist, err := gemDistribution(cfg)
	if err != nil {
		return err
	}
	layout := layoutForConfig(cfg)
	version, err := resolveStagedGemVersion(cfg, layout)
	if err != nil {
		return err
	}
	artifacts := make([]string, 0, len(cfg.Targets))
	for _, target := range cfg.Targets {
		artifacts = append(artifacts, gemArtifactPath(layout, dist, target, version))
	}
	sort.Strings(artifacts)

	env, err := publishEnv(dist, opts)
	if err != nil {
		return err
	}
	host := strings.TrimSpace(dist.Registry)
	if strings.TrimSpace(opts.Host) != "" {
		host = strings.TrimSpace(opts.Host)
	}
	host = normalizeHost(host)

	if opts.DryRun {
		if opts.Stdout != nil {
			for _, artifact := range artifacts {
				fmt.Fprintf(opts.Stdout, "gem publish dry-run: validated %s for host %s\n", filepath.Base(artifact), host)
			}
		}
		return nil
	}

	for _, artifact := range artifacts {
		writeProgressf(opts.Progress, "Publishing gem: %s\n", filepath.Base(artifact))
		args := []string{"push", artifact, "--host", host}
		if otp := strings.TrimSpace(resolveOTP(opts)); otp != "" {
			args = append(args, "--otp", otp)
		}
		cmd := command("gem", args...)
		cmd.Stdout = outputWriter(opts.Stdout)
		cmd.Stderr = outputWriter(opts.Stderr)
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gem push failed for %s: %w", filepath.Base(artifact), err)
		}
		writeProgressf(opts.Progress, "Published: %s\n", filepath.Base(artifact))
	}
	return nil
}

func NormalizeVersion(version string) (string, error) {
	v := strings.TrimSpace(version)
	if v == "" {
		return "", fmt.Errorf("gem version is empty")
	}
	if i := strings.Index(v, "+"); i >= 0 {
		v = v[:i]
	}
	if strings.Contains(v, "-") {
		parts := strings.SplitN(v, "-", 2)
		suffix := strings.ReplaceAll(parts[1], "-", ".")
		suffix = strings.ReplaceAll(suffix, "+", ".")
		v = parts[0] + ".pre." + suffix
	}
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("gem version is empty")
	}
	return v, nil
}

func gemDistribution(cfg *config.Config) (config.DistributionConfig, error) {
	if cfg == nil {
		return config.DistributionConfig{}, fmt.Errorf("config is nil")
	}
	dist, ok := cfg.Distributions["gem"]
	if !ok {
		return config.DistributionConfig{}, fmt.Errorf("missing required distribution: gem")
	}
	dist.Package = strings.TrimSpace(dist.Package)
	dist.Registry = strings.TrimSpace(dist.Registry)
	dist.PublishAuth = strings.TrimSpace(dist.PublishAuth)
	dist.RepositoryURL = dist.RepositoryURLValue()
	dist.License = dist.LicenseValue()
	dist.ReadmePath = strings.TrimSpace(dist.ReadmePath)
	if dist.Package == "" {
		return config.DistributionConfig{}, fmt.Errorf("gem distribution package is required")
	}
	if !gemNamePattern.MatchString(dist.Package) {
		return config.DistributionConfig{}, fmt.Errorf("invalid gem package name %q", dist.Package)
	}
	if dist.Registry == "" {
		dist.Registry = defaultGemRegistry
	}
	if dist.PublishAuth == "" {
		dist.PublishAuth = "token"
	}
	if dist.PublishAuth != "token" && dist.PublishAuth != "trusted" {
		return config.DistributionConfig{}, fmt.Errorf("invalid gem publish-auth %q: expected token or trusted", dist.PublishAuth)
	}
	return dist, nil
}

func layoutForConfig(cfg *config.Config) paths.Layout {
	if cfg == nil {
		return paths.NewLayout(config.DefaultWorkspaceDir)
	}
	return paths.NewLayout(cfg.EffectiveWorkspaceDir())
}

func resolveGemVersion(cfg *config.Config, dev bool) (string, error) {
	version, err := shared.ResolveStageVersion(cfg, dev)
	if err != nil {
		return "", err
	}
	return NormalizeVersion(version)
}

func resolveStagedGemVersion(cfg *config.Config, layout paths.Layout) (string, error) {
	version, err := shared.ReadBuildVersionForConfig(cfg)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		version, err = shared.ResolveStageVersion(cfg, false)
		if err != nil {
			return "", err
		}
	}
	return NormalizeVersion(version)
}

func stageGemForTarget(layout paths.Layout, cfg *config.Config, dist config.DistributionConfig, target config.Target, version string) error {
	srcBinary := binaryPath(layout, cfg.Tool.Name, target)
	if _, err := os.Stat(srcBinary); err != nil {
		return fmt.Errorf("missing built binary %s; run `omnidist build` before `omnidist gem stage`", srcBinary)
	}
	platform := gemPlatform(target)
	stagingDir := filepath.Join(layout.GemBuildDir, platform)
	if err := os.MkdirAll(filepath.Join(stagingDir, "libexec"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(stagingDir, "exe"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(stagingDir, "lib"), 0755); err != nil {
		return err
	}
	binaryName := cfg.Tool.Name
	if target.OS == "windows" {
		binaryName += ".exe"
	}
	if err := copyFile(srcBinary, filepath.Join(stagingDir, "libexec", binaryName), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "exe", cfg.Tool.Name), []byte(wrapperScript(cfg.Tool.Name, binaryName)), 0755); err != nil {
		return fmt.Errorf("write gem executable wrapper: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "lib", dist.Package+".rb"), []byte("# frozen_string_literal: true\n"), 0644); err != nil {
		return fmt.Errorf("write gem library stub: %w", err)
	}
	if dist.IncludeREADMEEnabled() {
		readmePath, required := shared.ResolveProjectREADMEPath(cfg.ReadmePath, dist.ReadmePath)
		if data, exists, err := shared.ReadProjectREADME(readmePath, required); err != nil {
			return err
		} else if exists {
			if err := os.WriteFile(filepath.Join(stagingDir, "README.md"), data, 0644); err != nil {
				return fmt.Errorf("write gem README: %w", err)
			}
		}
	}
	if data, err := readOptionalProjectLicense(); err == nil && len(data) > 0 {
		if err := os.WriteFile(filepath.Join(stagingDir, "LICENSE"), data, 0644); err != nil {
			return fmt.Errorf("write gem LICENSE: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(stagingDir, dist.Package+".gemspec"), []byte(gemspecContent(dist, cfg.Tool.Name, version, platform)), 0644); err != nil {
		return fmt.Errorf("write gemspec: %w", err)
	}

	artifactPath, err := filepath.Abs(gemArtifactPath(layout, dist, target, version))
	if err != nil {
		return fmt.Errorf("resolve gem artifact path: %w", err)
	}
	cmd := command("gem", "build", dist.Package+".gemspec", "--strict", "--output", artifactPath)
	cmd.Dir = stagingDir
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		text := strings.TrimSpace(stderr.String())
		if text != "" {
			return fmt.Errorf("gem build failed: %w: %s", err, text)
		}
		return fmt.Errorf("gem build failed: %w", err)
	}
	return nil
}

func gemspecContent(dist config.DistributionConfig, executable string, version string, platform string) string {
	license := dist.LicenseValue()
	if license == "" {
		license = "MIT"
	}
	readmeLine := ""
	if dist.IncludeREADMEEnabled() {
		readmeLine = "  spec.files << \"README.md\" if File.exist?(\"README.md\")\n"
	}
	repositoryURL := repositoryURLOrDefault(dist)
	allowedPushHostLine := ""
	if host := normalizeHost(dist.Registry); host != defaultGemRegistry {
		allowedPushHostLine = fmt.Sprintf("    \"allowed_push_host\" => %q,\n", host)
	}
	return fmt.Sprintf(`Gem::Specification.new do |spec|
  spec.name = %q
  spec.version = %q
  spec.summary = %q
  spec.description = %q
  spec.authors = ["omnidist"]
  spec.licenses = [%q]
  spec.homepage = %q
  spec.platform = Gem::Platform.new(%q)
  spec.required_ruby_version = ">= 3.1"
  spec.bindir = "exe"
  spec.executables = [%q]
  spec.require_paths = ["lib"]
  spec.files = Dir["lib/**/*", "exe/*", "libexec/**/*", "*.gemspec"].sort
  spec.metadata = {
    "rubygems_mfa_required" => "true",
    "source_code_uri" => %q,
%s  }
%s  spec.files << "LICENSE" if File.exist?("LICENSE")
end
`, dist.Package, version, fmt.Sprintf("Prebuilt %s CLI packaged by omnidist", executable), fmt.Sprintf("Prebuilt %s CLI packaged by omnidist with platform-specific binaries", executable), license, repositoryURL, platform, executable, repositoryURL, allowedPushHostLine, readmeLine)
}

func repositoryURLOrDefault(dist config.DistributionConfig) string {
	if v := dist.RepositoryURLValue(); v != "" {
		return v
	}
	return "https://example.invalid"
}

func gemPlatform(target config.Target) string {
	switch target.OS {
	case "darwin":
		if target.Arch == "amd64" {
			return "x86_64-darwin"
		}
		return "arm64-darwin"
	case "linux":
		if target.Variant == "musl" {
			if target.Arch == "amd64" {
				return "x86_64-linux-musl"
			}
			return "aarch64-linux-musl"
		}
		if target.Arch == "amd64" {
			return "x86_64-linux"
		}
		return "aarch64-linux"
	case "windows":
		switch target.Variant {
		case "mingw32":
			return "x64-mingw32"
		case "mingw-ucrt", "":
			return "x64-mingw-ucrt"
		default:
			return "x64-" + target.Variant
		}
	default:
		return target.Arch + "-" + target.OS
	}
}

func binaryPath(layout paths.Layout, toolName string, target config.Target) string {
	name := toolName
	if target.OS == "windows" {
		name += ".exe"
	}
	return filepath.Join(layout.DistDir, target.OS, target.Arch, name)
}

func gemArtifactPath(layout paths.Layout, dist config.DistributionConfig, target config.Target, version string) string {
	return filepath.Join(layout.GemPkgDir, fmt.Sprintf("%s-%s-%s.gem", dist.Package, version, gemPlatform(target)))
}

func wrapperScript(executable string, binaryName string) string {
	return fmt.Sprintf(`#!/usr/bin/env ruby
gem_root = File.expand_path("..", __dir__)
binary = File.join(gem_root, "libexec", %q)
exec(binary, *ARGV)
`, binaryName)
}

func copyFile(src string, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func readOptionalProjectLicense() ([]byte, error) {
	for _, candidate := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt"} {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read project license %s: %w", candidate, err)
		}
	}
	return nil, nil
}

func verifyGemArchive(path string, executable string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open gem artifact %s: %w", path, err)
	}
	defer file.Close()
	tr := tar.NewReader(file)
	foundDataTar := false
	foundMetadataGz := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("scan gem artifact %s: %w", path, err)
		}
		if hdr.Name == "metadata.gz" {
			foundMetadataGz = true
		}
		if hdr.Name == "data.tar.gz" {
			foundDataTar = true
			data, err := io.ReadAll(io.LimitReader(tr, hdr.Size))
			if err != nil {
				return fmt.Errorf("read data.tar.gz from gem artifact %s: %w", path, err)
			}
			if err := verifyGemData(data, executable); err != nil {
				return fmt.Errorf("invalid gem artifact %s: %w", path, err)
			}
		}
	}
	if !foundDataTar {
		return fmt.Errorf("invalid gem artifact %s: missing data.tar.gz", path)
	}
	if !foundMetadataGz {
		return fmt.Errorf("invalid gem artifact %s: missing metadata.gz", path)
	}
	return nil
}

func verifyGemData(data []byte, executable string) error {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("read data.tar.gz: %w", err)
	}
	defer gzr.Close()

	wrapperPath := filepath.ToSlash(filepath.Join("exe", executable))
	binaryPath := filepath.ToSlash(filepath.Join("libexec", executable))
	foundWrapper := false
	foundBinary := false
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("scan data.tar.gz: %w", err)
		}
		name := filepath.ToSlash(hdr.Name)
		if name == wrapperPath {
			foundWrapper = true
		}
		if name == binaryPath || name == binaryPath+".exe" {
			foundBinary = true
		}
	}
	if !foundWrapper {
		return fmt.Errorf("missing %s", wrapperPath)
	}
	if !foundBinary {
		return fmt.Errorf("missing %s binary", binaryPath)
	}
	return nil
}

func publishEnv(dist config.DistributionConfig, opts PublishOptions) ([]string, error) {
	env := append([]string{}, os.Environ()...)
	key := strings.TrimSpace(opts.APIKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv(gemHostAPIKeyEnv))
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv(rubygemsAPIKeyEnv))
	}
	if key == "" && !opts.DryRun && dist.PublishAuth != "trusted" {
		return nil, fmt.Errorf("gem publish requires API key auth: pass --api-key or set %s / %s", gemHostAPIKeyEnv, rubygemsAPIKeyEnv)
	}
	if key != "" {
		env = append(env, gemHostAPIKeyEnv+"="+key)
	}
	return env, nil
}

func outputWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func writeProgressf(w io.Writer, format string, args ...interface{}) {
	if w != nil {
		fmt.Fprintf(w, format, args...)
	}
}

func normalizeHost(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return defaultGemRegistry
	}
	if parsed, err := url.Parse(v); err == nil && parsed.Scheme != "" {
		return v
	}
	return v
}

func resolveOTP(opts PublishOptions) string {
	if value := strings.TrimSpace(opts.OTP); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(gemHostOTPCodeEnv))
}
