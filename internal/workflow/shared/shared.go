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
	// ProjectREADMEPath is the README file included in staged artifacts when enabled.
	ProjectREADMEPath = "README.md"
)

var exactSemverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var gitDescribePattern = regexp.MustCompile(`^(\d+\.\d+\.\d+)-(\d+)-g([0-9a-fA-F]+)$`)

// ResolveVersion resolves a version string from the configured version source.
func ResolveVersion(cfg *config.Config, dev bool) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	return resolveConfiguredVersion(cfg, dev, false)
}

// ResolveReleaseVersion resolves an exact publishable semver version.
func ResolveReleaseVersion(cfg *config.Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	version, err := resolveConfiguredVersion(cfg, false, true)
	if err != nil {
		return "", err
	}
	if !isExactSemver(version) {
		return "", fmt.Errorf("release version %q is not exact semver (expected X.Y.Z)", version)
	}

	return version, nil
}

func resolveConfiguredVersion(cfg *config.Config, dev bool, release bool) (string, error) {
	source := strings.TrimSpace(cfg.Version.Source)
	if source == "" {
		source = "git-tag"
	}

	var version string
	switch source {
	case "git-tag":
		var (
			v   string
			err error
		)
		if release {
			v, err = resolveExactSemverTag()
		} else {
			v, err = resolveGitTagVersion(dev)
		}
		if err != nil {
			return "", fmt.Errorf("resolve git tag version: %w", err)
		}
		version = v
	case "file":
		versionPath := strings.TrimSpace(cfg.Version.File)
		if versionPath == "" {
			versionPath = config.DefaultVersionFile
		}
		data, err := os.ReadFile(versionPath)
		if err != nil {
			return "", fmt.Errorf("read version file %s: %w", versionPath, err)
		}
		version = string(data)
	case "env":
		version = os.Getenv(EnvVersionName)
	case "fixed":
		version = cfg.Version.Fixed
	default:
		return "", fmt.Errorf("unknown version source %q", source)
	}

	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("empty version from source %q", source)
	}

	return version, nil
}

// WriteBuildVersion persists the resolved build version to `dist/version`.
func WriteBuildVersion(version string) error {
	return writeBuildVersionForLayout(paths.NewLayout(config.DefaultWorkspaceDir), version)
}

// WriteBuildVersionForConfig persists the resolved build version for the selected workspace.
func WriteBuildVersionForConfig(cfg *config.Config, version string) error {
	layout := paths.NewLayout(config.DefaultWorkspaceDir)
	if cfg != nil {
		layout = paths.NewLayout(cfg.EffectiveWorkspaceDir())
	}
	return writeBuildVersionForLayout(layout, version)
}

func writeBuildVersionForLayout(layout paths.Layout, version string) error {
	v := strings.TrimSpace(version)
	if v == "" {
		return fmt.Errorf("version is empty")
	}

	if err := os.MkdirAll(layout.DistDir, 0755); err != nil {
		return fmt.Errorf("create dist directory: %w", err)
	}

	if err := os.WriteFile(layout.DistVersionPath, []byte(v+"\n"), 0644); err != nil {
		return fmt.Errorf("write build version file %s: %w", layout.DistVersionPath, err)
	}
	return nil
}

// ReadBuildVersion reads the persisted build version from `dist/version`.
func ReadBuildVersion() (string, error) {
	return readBuildVersionForLayout(paths.NewLayout(config.DefaultWorkspaceDir))
}

// ReadBuildVersionForConfig reads persisted build version for the selected workspace.
func ReadBuildVersionForConfig(cfg *config.Config) (string, error) {
	layout := paths.NewLayout(config.DefaultWorkspaceDir)
	if cfg != nil {
		layout = paths.NewLayout(cfg.EffectiveWorkspaceDir())
	}
	return readBuildVersionForLayout(layout)
}

func readBuildVersionForLayout(layout paths.Layout) (string, error) {
	data, err := os.ReadFile(layout.DistVersionPath)
	if err != nil {
		return "", fmt.Errorf("read build version file %s: %w", layout.DistVersionPath, err)
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("empty build version in %s", layout.DistVersionPath)
	}
	return version, nil
}

// ReadOptionalProjectREADME reads README.md from the project root if it exists.
func ReadOptionalProjectREADME() ([]byte, bool, error) {
	data, err := os.ReadFile(ProjectREADMEPath)
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("read project README %s: %w", ProjectREADMEPath, err)
}

// ResolveStageVersion resolves the version used for staging artifacts.
func ResolveStageVersion(cfg *config.Config, dev bool) (string, error) {
	if dev {
		return ResolveVersion(cfg, true)
	}

	version, err := ReadBuildVersionForConfig(cfg)
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
