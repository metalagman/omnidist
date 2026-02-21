package uv

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

var pyprojectVersionPattern = regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"\s*$`)

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

	if err := resetUVStagingDir(); err != nil {
		return err
	}
	if err := writeStagingPyproject(uvDist.Package, version); err != nil {
		return fmt.Errorf("write uv staging pyproject: %w", err)
	}

	for _, target := range cfg.Targets {
		if err := stageWheel(cfg, uvDist, target, version); err != nil {
			return fmt.Errorf("stage wheel for %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	return nil
}

func resetUVStagingDir() error {
	if err := os.RemoveAll(paths.UVDistDir); err != nil {
		return fmt.Errorf("clean uv staging directory: %w", err)
	}
	if err := os.MkdirAll(paths.UVDistDir, 0755); err != nil {
		return fmt.Errorf("create uv staging directory: %w", err)
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

	version, err := resolveUVStagingVersion(cfg, false)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	if err := validatePublishVersionPolicy(uvDist.IndexURL, version); err != nil {
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

	version, err := resolveUVPublishVersion(cfg)
	if err != nil {
		return err
	}
	if err := validatePublishVersionPolicy(uvDist.IndexURL, version); err != nil {
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

func validatePublishVersionPolicy(indexURL string, version string) error {
	if !isPyPIIndexURL(indexURL) {
		return nil
	}
	if strings.Contains(version, "+") {
		return fmt.Errorf("version %q contains local version metadata (+...), which PyPI/TestPyPI rejects; restage with a publishable version (e.g. exact semver tag or env VERSION without +)", version)
	}
	return nil
}

func isPyPIIndexURL(indexURL string) bool {
	u, err := url.Parse(strings.TrimSpace(indexURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "upload.pypi.org" || host == "test.pypi.org"
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

func resolveUVPublishVersion(cfg *config.Config) (string, error) {
	version, err := readStagingPyprojectVersion()
	if err == nil {
		return version, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read uv staging pyproject version: %w", err)
	}

	version, err = shared.ReadBuildVersion()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("read build version: %w", err)
		}
		return resolveUVReleaseVersion(cfg)
	}

	pep440, err := shared.ToPEP440(version)
	if err != nil {
		return "", fmt.Errorf("convert build version to PEP 440: %w", err)
	}

	return pep440, nil
}

func resolveUVStagingVersion(cfg *config.Config, dev bool) (string, error) {
	version, err := readStagingPyprojectVersion()
	if err == nil {
		return version, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read uv staging pyproject version: %w", err)
	}
	return resolveUVVersion(cfg, dev)
}

func writeStagingPyproject(pkg string, version string) error {
	name := strings.TrimSpace(pkg)
	v := strings.TrimSpace(version)
	if name == "" {
		return fmt.Errorf("package name is empty")
	}
	if v == "" {
		return fmt.Errorf("version is empty")
	}

	if err := os.MkdirAll(paths.UVDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`[project]
name = %s
version = %s

[tool.omnidist]
generated = true
`, strconv.Quote(name), strconv.Quote(v))

	return os.WriteFile(paths.UVPyprojectPath, []byte(content), 0644)
}

func readStagingPyprojectVersion() (string, error) {
	data, err := os.ReadFile(paths.UVPyprojectPath)
	if err != nil {
		return "", err
	}
	match := pyprojectVersionPattern.FindStringSubmatch(string(data))
	if len(match) < 2 {
		return "", fmt.Errorf("missing project.version in %s", paths.UVPyprojectPath)
	}
	version := strings.TrimSpace(match[1])
	if version == "" {
		return "", fmt.Errorf("empty project.version in %s", paths.UVPyprojectPath)
	}
	return version, nil
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
	binaryName := shared.BinaryName(cfg.Tool.Name, goOS)
	binaryPath := shared.WheelBinaryPath(uvDist.Package, cfg.Tool.Name, goOS)
	launcherPath := path.Join(distName, "_launcher.py")

	wheelMeta := fmt.Sprintf("Wheel-Version: 1.0\nGenerator: omnidist\nRoot-Is-Purelib: false\nTag: py3-none-%s\n", platformTag)
	metadata := fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\nSummary: Binary distribution for %s\n", uvDist.Package, version, cfg.Tool.Name)
	recordPath := path.Join(distInfoDir, "RECORD")
	files := []wheelFile{
		{name: path.Join(distName, "__init__.py"), data: []byte("\"\"\"Generated by omnidist.\"\"\"\n"), mode: 0644},
		{name: launcherPath, data: []byte(pythonLauncher(binaryName)), mode: 0644},
		{name: binaryPath, data: binaryData, mode: 0755},
		{name: path.Join(distInfoDir, "WHEEL"), data: []byte(wheelMeta), mode: 0644},
		{name: path.Join(distInfoDir, "METADATA"), data: []byte(metadata), mode: 0644},
		{name: path.Join(distInfoDir, "entry_points.txt"), data: []byte(consoleEntryPoints(cfg.Tool.Name, distName)), mode: 0644},
	}

	var record strings.Builder
	for _, file := range files {
		if err := addZipFile(zipWriter, file.name, file.data, file.mode); err != nil {
			return err
		}
		record.WriteString(wheelRecordLine(file.name, file.data))
	}
	record.WriteString(recordPath + ",,\n")

	if err := addZipFile(zipWriter, recordPath, []byte(record.String()), 0644); err != nil {
		return err
	}

	return nil
}

type wheelFile struct {
	name string
	data []byte
	mode os.FileMode
}

func addZipFile(zipWriter *zip.Writer, name string, data []byte, mode os.FileMode) error {
	header := &zip.FileHeader{
		Name:   name,
		Method: zip.Store,
	}
	header.UncompressedSize64 = uint64(len(data))
	header.CompressedSize64 = uint64(len(data))
	header.CRC32 = crc32.ChecksumIEEE(data)
	header.SetMode(mode)
	header.Modified = time.Unix(0, 0)

	writer, err := zipWriter.CreateRaw(header)
	if err != nil {
		return fmt.Errorf("create zip entry %s: %w", name, err)
	}
	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("write zip entry %s: %w", name, err)
	}
	return nil
}

func pythonLauncher(binaryName string) string {
	return fmt.Sprintf(`import subprocess
import sys
from pathlib import Path


def main() -> int:
    binary = Path(__file__).resolve().parent / "bin" / %q
    if not binary.exists():
        print(f"Binary not found: {binary}", file=sys.stderr)
        return 1
    return subprocess.call([str(binary), *sys.argv[1:]])
`, binaryName)
}

func consoleEntryPoints(toolName string, distName string) string {
	return fmt.Sprintf("[console_scripts]\n%s=%s._launcher:main\n", toolName, distName)
}

func wheelRecordLine(name string, data []byte) string {
	sum := sha256.Sum256(data)
	digest := base64.RawURLEncoding.EncodeToString(sum[:])
	return fmt.Sprintf("%s,sha256=%s,%d\n", name, digest, len(data))
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
	expectedLauncher := path.Join(distName, "_launcher.py")
	distInfoDir := fmt.Sprintf("%s-%s.dist-info", distName, version)
	expectedMetadata := path.Join(distInfoDir, "METADATA")
	expectedWheelMeta := path.Join(distInfoDir, "WHEEL")
	expectedEntrypoints := path.Join(distInfoDir, "entry_points.txt")
	expectedRecord := path.Join(distInfoDir, "RECORD")

	var (
		foundBinary   bool
		foundLauncher bool
		metadataBytes []byte
		wheelBytes    []byte
		entryBytes    []byte
		recordBytes   []byte
		allFiles      = make(map[string][]byte)
	)

	for _, file := range zipReader.File {
		data, err := readZipFile(file)
		if err != nil {
			return err
		}
		allFiles[file.Name] = data

		switch file.Name {
		case expectedBinary:
			foundBinary = true
		case expectedLauncher:
			foundLauncher = true
		case expectedMetadata:
			metadataBytes = data
		case expectedWheelMeta:
			wheelBytes = data
		case expectedEntrypoints:
			entryBytes = data
		case expectedRecord:
			recordBytes = data
		}
	}

	if !foundBinary {
		return fmt.Errorf("missing binary %s in wheel %s", expectedBinary, wheelPath)
	}
	if !foundLauncher {
		return fmt.Errorf("missing launcher %s in wheel %s", expectedLauncher, wheelPath)
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
	if len(entryBytes) == 0 {
		return fmt.Errorf("missing entry_points.txt in wheel %s", wheelPath)
	}
	expectedEntrypoint := fmt.Sprintf("%s=%s._launcher:main\n", cfg.Tool.Name, distName)
	if !strings.Contains(string(entryBytes), expectedEntrypoint) {
		return fmt.Errorf("console entrypoint mismatch in wheel %s", wheelPath)
	}
	if len(recordBytes) == 0 {
		return fmt.Errorf("missing RECORD in wheel %s", wheelPath)
	}
	if err := verifyRecordEntries(expectedRecord, recordBytes, allFiles); err != nil {
		return fmt.Errorf("invalid RECORD in wheel %s: %w", wheelPath, err)
	}

	return nil
}

func verifyRecordEntries(recordPath string, recordBytes []byte, files map[string][]byte) error {
	lines := strings.Split(strings.TrimSpace(string(recordBytes)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("RECORD is empty")
	}

	seenRecord := false
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) != 3 {
			return fmt.Errorf("invalid RECORD line %q", line)
		}

		name, hashValue, sizeValue := parts[0], parts[1], parts[2]
		if name == recordPath {
			seenRecord = true
			if hashValue != "" || sizeValue != "" {
				return fmt.Errorf("RECORD self-entry must have empty hash and size")
			}
			continue
		}

		data, ok := files[name]
		if !ok {
			return fmt.Errorf("RECORD references missing file %q", name)
		}
		expectedLine := strings.TrimSpace(wheelRecordLine(name, data))
		expectedParts := strings.Split(expectedLine, ",")
		if hashValue != expectedParts[1] || sizeValue != expectedParts[2] {
			return fmt.Errorf("RECORD mismatch for %q", name)
		}
	}

	if !seenRecord {
		return fmt.Errorf("RECORD missing self-entry")
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
