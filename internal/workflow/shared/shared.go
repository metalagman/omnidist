package shared

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

const (
	// DefaultUVLinuxTag is the default wheel compatibility policy for Linux targets.
	DefaultUVLinuxTag = "manylinux2014"
	// EnvVersionName is the environment variable used when `version.source` is `env`.
	EnvVersionName = "OMNIDIST_VERSION"
)

var exactSemverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var gitDescribePattern = regexp.MustCompile(`^(\d+\.\d+\.\d+)-(\d+)-g([0-9a-fA-F]+)$`)

// ResolveVersion resolves a version string from the configured version source.
func ResolveVersion(cfg *config.Config, dev bool) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	var version string
	switch strings.TrimSpace(cfg.Version.Source) {
	case "git-tag":
		v, err := resolveGitTagVersion(dev)
		if err != nil {
			return "", fmt.Errorf("resolve git tag version: %w", err)
		}
		version = v
	case "file":
		data, err := os.ReadFile("VERSION")
		if err != nil {
			return "", fmt.Errorf("read VERSION file %s: %w", "VERSION", err)
		}
		version = string(data)
	case "env":
		version = os.Getenv(EnvVersionName)
	default:
		return "", fmt.Errorf("unknown version source %q", cfg.Version.Source)
	}

	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("empty version from source %q", cfg.Version.Source)
	}

	return version, nil
}

// ResolveReleaseVersion resolves an exact publishable semver version.
func ResolveReleaseVersion(cfg *config.Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	var version string
	switch strings.TrimSpace(cfg.Version.Source) {
	case "git-tag":
		v, err := resolveExactSemverTag()
		if err != nil {
			return "", fmt.Errorf("resolve git tag version: %w", err)
		}
		version = v
	case "file":
		data, err := os.ReadFile("VERSION")
		if err != nil {
			return "", fmt.Errorf("read VERSION file %s: %w", "VERSION", err)
		}
		version = strings.TrimSpace(string(data))
	case "env":
		version = strings.TrimSpace(os.Getenv(EnvVersionName))
	default:
		return "", fmt.Errorf("unknown version source %q", cfg.Version.Source)
	}

	if version == "" {
		return "", fmt.Errorf("empty version from source %q", cfg.Version.Source)
	}
	if !isExactSemver(version) {
		return "", fmt.Errorf("release version %q is not exact semver (expected X.Y.Z)", version)
	}

	return version, nil
}

// WriteBuildVersion persists the resolved build version to `dist/version`.
func WriteBuildVersion(version string) error {
	v := strings.TrimSpace(version)
	if v == "" {
		return fmt.Errorf("version is empty")
	}

	if err := os.MkdirAll(paths.DistDir, 0755); err != nil {
		return fmt.Errorf("create dist directory: %w", err)
	}

	if err := os.WriteFile(paths.DistVersionPath, []byte(v+"\n"), 0644); err != nil {
		return fmt.Errorf("write build version file %s: %w", paths.DistVersionPath, err)
	}
	return nil
}

// ReadBuildVersion reads the persisted build version from `dist/version`.
func ReadBuildVersion() (string, error) {
	data, err := os.ReadFile(paths.DistVersionPath)
	if err != nil {
		return "", fmt.Errorf("read build version file %s: %w", paths.DistVersionPath, err)
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("empty build version in %s", paths.DistVersionPath)
	}
	return version, nil
}

// ResolveStageVersion resolves the version used for staging artifacts.
func ResolveStageVersion(cfg *config.Config, dev bool) (string, error) {
	if dev {
		return ResolveVersion(cfg, true)
	}

	version, err := ReadBuildVersion()
	if err == nil {
		return version, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read build version: %w", err)
	}

	return ResolveVersion(cfg, false)
}

func resolveGitTagVersion(dev bool) (string, error) {
	args := []string{"describe", "--tags", "--always"}
	if dev {
		args = append(args, "--long")
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git describe --tags --always failed: %w", err)
	}

	version := string(bytes.TrimSpace(out))
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}

	if dev {
		parts := strings.Split(version, "-")
		if len(parts) >= 3 {
			if !isExactSemver(parts[0]) {
				return "", fmt.Errorf("tag %q is not exact semver (expected vX.Y.Z or X.Y.Z)", parts[0])
			}
			version = fmt.Sprintf("%s-dev.%s.%s", parts[0], parts[1], parts[2])
		}
	}

	return version, nil
}

func resolveExactSemverTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git describe --tags --exact-match failed: %w; HEAD is not at an exact semver tag; create tag vX.Y.Z before publishing", err)
	}

	tag := strings.TrimSpace(string(out))
	version := strings.TrimPrefix(tag, "v")
	if !isExactSemver(version) {
		return "", fmt.Errorf("tag %q is not exact semver (expected vX.Y.Z or X.Y.Z)", tag)
	}

	return version, nil
}

func isExactSemver(v string) bool {
	return exactSemverPattern.MatchString(strings.TrimSpace(v))
}

// ToPEP440 converts an omnidist version to a PEP 440-compatible version when possible.
func ToPEP440(version string) (string, error) {
	v := strings.TrimSpace(version)
	if v == "" {
		return "", fmt.Errorf("version is empty")
	}

	parts := strings.SplitN(v, "-dev.", 2)
	if len(parts) == 2 {
		devParts := strings.Split(parts[1], ".")
		if len(devParts) >= 2 {
			return fmt.Sprintf("%s.dev%s", parts[0], devParts[0]), nil
		}
		return fmt.Sprintf("%s.dev%s", parts[0], strings.ReplaceAll(parts[1], ".", "")), nil
	}

	if matches := gitDescribePattern.FindStringSubmatch(v); len(matches) == 4 {
		base := matches[1]
		devCount := matches[2]
		if _, err := strconv.Atoi(devCount); err != nil {
			return "", fmt.Errorf("invalid git describe dev count %q", devCount)
		}
		return fmt.Sprintf("%s.dev%s", base, devCount), nil
	}

	if strings.Contains(v, "-") {
		return "", fmt.Errorf("version %q is not PEP 440 compatible", v)
	}

	return v, nil
}

// NormalizePythonDistributionName converts npm-style names to Python distribution naming.
func NormalizePythonDistributionName(pkg string) string {
	name := strings.TrimSpace(pkg)
	name = strings.TrimPrefix(name, "@")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

// NormalizeGoTarget returns trimmed GOOS/GOARCH values from a target.
func NormalizeGoTarget(target config.Target) (goOS string, goArch string) {
	goOS = strings.TrimSpace(target.OS)
	goArch = strings.TrimSpace(target.Arch)
	return goOS, goArch
}

// BinaryName returns the platform-specific binary filename for a tool.
func BinaryName(toolName string, goOS string) string {
	if goOS == "windows" {
		return toolName + ".exe"
	}
	return toolName
}

// WheelPlatformTag returns the Python wheel platform tag for a target.
func WheelPlatformTag(target config.Target, linuxTag string) (string, error) {
	goOS, goArch := NormalizeGoTarget(target)

	switch goOS {
	case "linux":
		policy := strings.TrimSpace(linuxTag)
		if policy == "" {
			policy = DefaultUVLinuxTag
		}
		switch goArch {
		case "amd64":
			return policy + "_x86_64", nil
		case "arm64":
			return policy + "_aarch64", nil
		default:
			return "", fmt.Errorf("unsupported linux architecture %q", target.Arch)
		}
	case "darwin":
		switch goArch {
		case "amd64":
			return "macosx_10_13_x86_64", nil
		case "arm64":
			return "macosx_11_0_arm64", nil
		default:
			return "", fmt.Errorf("unsupported darwin architecture %q", target.Arch)
		}
	case "windows":
		switch goArch {
		case "amd64":
			return "win_amd64", nil
		case "arm64":
			return "win_arm64", nil
		default:
			return "", fmt.Errorf("unsupported windows architecture %q", target.Arch)
		}
	default:
		return "", fmt.Errorf("unsupported OS %q", target.OS)
	}
}

// WheelFilename returns the wheel filename for a target artifact.
func WheelFilename(pkg, version string, target config.Target, linuxTag string) (string, error) {
	platformTag, err := WheelPlatformTag(target, linuxTag)
	if err != nil {
		return "", err
	}
	distName := NormalizePythonDistributionName(pkg)
	return fmt.Sprintf("%s-%s-py3-none-%s.whl", distName, version, platformTag), nil
}

// WheelBinaryPath returns the binary path within an unpacked wheel.
func WheelBinaryPath(pkg, toolName string, goOS string) string {
	return path.Join(NormalizePythonDistributionName(pkg), "bin", BinaryName(toolName, goOS))
}
