package uv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

type StageOptions struct {
	Dev bool
}

type PublishOptions struct {
	DryRun     bool
	PublishURL string
	Token      string
}

type VerificationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func CheckDependency() error {
	if _, err := exec.LookPath("uv"); err != nil {
		return fmt.Errorf("uv executable not found in PATH. Install uv from https://docs.astral.sh/uv/getting-started/installation/ and retry")
	}
	return nil
}

func Stage(cfg *config.Config, opts StageOptions) error {
	uvDist, err := uvDistribution(cfg)
	if err != nil {
		return err
	}

	version, err := resolveUVVersion(cfg, opts.Dev)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(paths.UVDistDir, 0755); err != nil {
		return fmt.Errorf("create uv staging directory: %w", err)
	}

	for _, target := range cfg.Targets {
		if err := stageWheel(cfg, uvDist, target, version); err != nil {
			return fmt.Errorf("stage wheel for %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func Verify(cfg *config.Config) *VerificationResult {
	result := &VerificationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	uvDist, err := uvDistribution(cfg)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	version, err := resolveUVVersion(cfg, false)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	for _, target := range cfg.Targets {
		wheelPath, err := wheelPathForTarget(uvDist, target, version)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		if _, err := os.Stat(wheelPath); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("missing wheel artifact: %s", wheelPath))
			continue
		}

		if err := verifyWheel(cfg, uvDist, target, version, wheelPath); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result
}

func Publish(cfg *config.Config, opts PublishOptions) error {
	uvDist, err := uvDistribution(cfg)
	if err != nil {
		return err
	}

	version, err := resolveUVReleaseVersion(cfg)
	if err != nil {
		return err
	}

	artifacts, err := collectWheelArtifacts(cfg, uvDist, version)
	if err != nil {
		return err
	}

	args := buildPublishArgs(uvDist.IndexURL, opts, artifacts)
	token, err := resolvePublishToken(opts)
	if err != nil {
		return err
	}

	cmd := exec.Command("uv", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append([]string{}, os.Environ()...)
	// Always use token auth mode for uv publish.
	cmd.Env = append(cmd.Env, "UV_PUBLISH_USERNAME=__token__")
	if token != "" {
		cmd.Env = append(cmd.Env, "UV_PUBLISH_TOKEN="+token)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uv publish failed: %w", err)
	}

	return nil
}

func buildPublishArgs(defaultIndexURL string, opts PublishOptions, artifacts []string) []string {
	args := []string{"publish"}

	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	publishURL := strings.TrimSpace(defaultIndexURL)
	if v := strings.TrimSpace(opts.PublishURL); v != "" {
		publishURL = v
	}
	if publishURL != "" {
		args = append(args, "--publish-url", publishURL)
	}

	args = append(args, artifacts...)
	return args
}

func resolvePublishToken(opts PublishOptions) (string, error) {
	token := strings.TrimSpace(opts.Token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("UV_PUBLISH_TOKEN"))
	}
	if token == "" && !opts.DryRun {
		return "", fmt.Errorf("uv publish requires token auth: pass --token or set UV_PUBLISH_TOKEN")
	}
	return token, nil
}

func uvDistribution(cfg *config.Config) (config.DistributionConfig, error) {
	if cfg == nil {
		return config.DistributionConfig{}, fmt.Errorf("config is nil")
	}

	dist, ok := cfg.Distributions["uv"]
	if !ok {
		return config.DistributionConfig{}, fmt.Errorf("missing required distribution: uv")
	}

	dist.Package = strings.TrimSpace(dist.Package)
	dist.IndexURL = strings.TrimSpace(dist.IndexURL)
	dist.LinuxTag = strings.TrimSpace(dist.LinuxTag)

	if dist.Package == "" {
		return config.DistributionConfig{}, fmt.Errorf("uv distribution package is required")
	}

	if dist.LinuxTag == "" {
		dist.LinuxTag = shared.DefaultUVLinuxTag
	}

	if !isSupportedLinuxTag(dist.LinuxTag) {
		return config.DistributionConfig{}, fmt.Errorf("invalid uv linux-tag %q: expected one of %s", dist.LinuxTag, strings.Join(supportedLinuxTags(), ", "))
	}

	return dist, nil
}

func supportedLinuxTags() []string {
	return []string{"manylinux2014", "musllinux_1_2"}
}

func isSupportedLinuxTag(v string) bool {
	for _, candidate := range supportedLinuxTags() {
		if v == candidate {
			return true
		}
	}
	return false
}

func resolveUVVersion(cfg *config.Config, dev bool) (string, error) {
	version, err := shared.ResolveStageVersion(cfg, dev)
	if err != nil {
		return "", err
	}

	pep440, err := shared.ToPEP440(version)
	if err != nil {
		return "", err
	}

	return pep440, nil
}

func resolveUVReleaseVersion(cfg *config.Config) (string, error) {
	version, err := shared.ResolveReleaseVersion(cfg)
	if err != nil {
		return "", err
	}

	pep440, err := shared.ToPEP440(version)
	if err != nil {
		return "", err
	}

	return pep440, nil
}

func stageWheel(cfg *config.Config, uvDist config.DistributionConfig, target config.Target, version string) error {
	goOS, _ := shared.NormalizeGoTarget(target)
	binaryName := shared.BinaryName(cfg.Tool.Name, goOS)
	sourceBinary := filepath.Join(paths.DistDir, target.OS, config.MapArchToNPM(target.Arch), binaryName)

	binaryData, err := os.ReadFile(sourceBinary)
	if err != nil {
		return fmt.Errorf("read built binary %s: %w", sourceBinary, err)
	}

	wheelPath, err := wheelPathForTarget(uvDist, target, version)
	if err != nil {
		return err
	}

	if err := writeWheel(wheelPath, cfg, uvDist, target, version, binaryData); err != nil {
		return err
	}

	return nil
}

func wheelPathForTarget(uvDist config.DistributionConfig, target config.Target, version string) (string, error) {
	filename, err := shared.WheelFilename(uvDist.Package, version, target, uvDist.LinuxTag)
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.UVDistDir, filename), nil
}

func writeWheel(wheelPath string, cfg *config.Config, uvDist config.DistributionConfig, target config.Target, version string, binaryData []byte) error {
	platformTag, err := shared.WheelPlatformTag(target, uvDist.LinuxTag)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(wheelPath), 0755); err != nil {
		return fmt.Errorf("create wheel directory: %w", err)
	}

	f, err := os.Create(wheelPath)
	if err != nil {
		return fmt.Errorf("create wheel %s: %w", wheelPath, err)
	}
	defer f.Close()

	zipWriter := zip.NewWriter(f)
	defer zipWriter.Close()

	distName := shared.NormalizePythonDistributionName(uvDist.Package)
	distInfoDir := fmt.Sprintf("%s-%s.dist-info", distName, version)
	goOS, _ := shared.NormalizeGoTarget(target)
	binaryPath := shared.WheelBinaryPath(uvDist.Package, cfg.Tool.Name, goOS)

	if err := addZipFile(zipWriter, path.Join(distName, "__init__.py"), []byte("\"\"\"Generated by omnidist.\"\"\"\n"), 0644); err != nil {
		return err
	}

	if err := addZipFile(zipWriter, binaryPath, binaryData, 0755); err != nil {
		return err
	}

	wheelMeta := fmt.Sprintf("Wheel-Version: 1.0\nGenerator: omnidist\nRoot-Is-Purelib: false\nTag: py3-none-%s\n", platformTag)
	if err := addZipFile(zipWriter, path.Join(distInfoDir, "WHEEL"), []byte(wheelMeta), 0644); err != nil {
		return err
	}

	metadata := fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\nSummary: Binary distribution for %s\n", uvDist.Package, version, cfg.Tool.Name)
	if err := addZipFile(zipWriter, path.Join(distInfoDir, "METADATA"), []byte(metadata), 0644); err != nil {
		return err
	}

	if err := addZipFile(zipWriter, path.Join(distInfoDir, "RECORD"), []byte(""), 0644); err != nil {
		return err
	}

	return nil
}

func addZipFile(zipWriter *zip.Writer, name string, data []byte, mode os.FileMode) error {
	header := &zip.FileHeader{
		Name:   name,
		Method: zip.Deflate,
	}
	header.SetMode(mode)
	header.Modified = time.Unix(0, 0)

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip entry %s: %w", name, err)
	}
	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("write zip entry %s: %w", name, err)
	}
	return nil
}

func verifyWheel(cfg *config.Config, uvDist config.DistributionConfig, target config.Target, version string, wheelPath string) error {
	zipReader, err := zip.OpenReader(wheelPath)
	if err != nil {
		return fmt.Errorf("open wheel %s: %w", wheelPath, err)
	}
	defer zipReader.Close()

	platformTag, err := shared.WheelPlatformTag(target, uvDist.LinuxTag)
	if err != nil {
		return err
	}

	goOS, _ := shared.NormalizeGoTarget(target)
	expectedBinary := shared.WheelBinaryPath(uvDist.Package, cfg.Tool.Name, goOS)
	distName := shared.NormalizePythonDistributionName(uvDist.Package)
	distInfoDir := fmt.Sprintf("%s-%s.dist-info", distName, version)
	expectedMetadata := path.Join(distInfoDir, "METADATA")
	expectedWheelMeta := path.Join(distInfoDir, "WHEEL")

	var (
		foundBinary   bool
		metadataBytes []byte
		wheelBytes    []byte
	)

	for _, file := range zipReader.File {
		switch file.Name {
		case expectedBinary:
			foundBinary = true
		case expectedMetadata:
			data, err := readZipFile(file)
			if err != nil {
				return err
			}
			metadataBytes = data
		case expectedWheelMeta:
			data, err := readZipFile(file)
			if err != nil {
				return err
			}
			wheelBytes = data
		}
	}

	if !foundBinary {
		return fmt.Errorf("missing binary %s in wheel %s", expectedBinary, wheelPath)
	}

	if len(metadataBytes) == 0 {
		return fmt.Errorf("missing METADATA in wheel %s", wheelPath)
	}
	metaText := string(metadataBytes)
	if !strings.Contains(metaText, "Name: "+uvDist.Package+"\n") {
		return fmt.Errorf("package name mismatch in METADATA for %s", wheelPath)
	}
	if !strings.Contains(metaText, "Version: "+version+"\n") {
		return fmt.Errorf("version mismatch in METADATA for %s", wheelPath)
	}

	if len(wheelBytes) == 0 {
		return fmt.Errorf("missing WHEEL metadata in wheel %s", wheelPath)
	}
	if !strings.Contains(string(wheelBytes), "Tag: py3-none-"+platformTag+"\n") {
		return fmt.Errorf("platform tag mismatch in WHEEL metadata for %s", wheelPath)
	}

	return nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open zip entry %s: %w", f.Name, err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func collectWheelArtifacts(cfg *config.Config, uvDist config.DistributionConfig, version string) ([]string, error) {
	artifacts := make([]string, 0, len(cfg.Targets))
	for _, target := range cfg.Targets {
		wheelPath, err := wheelPathForTarget(uvDist, target, version)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(wheelPath); err != nil {
			return nil, fmt.Errorf("missing wheel artifact: %s", wheelPath)
		}
		artifacts = append(artifacts, wheelPath)
	}
	sort.Strings(artifacts)
	return artifacts, nil
}
