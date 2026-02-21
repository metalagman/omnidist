package shared

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
)

const (
	DefaultUVLinuxTag = "manylinux2014"
)

var exactSemverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

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
			return "", fmt.Errorf("read VERSION file: %w", err)
		}
		version = string(data)
	case "env":
		version = os.Getenv("VERSION")
	default:
		return "", fmt.Errorf("unknown version source %q", cfg.Version.Source)
	}

	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("empty version from source %q", cfg.Version.Source)
	}

	return version, nil
}

func resolveGitTagVersion(dev bool) (string, error) {
	if !dev {
		return resolveExactSemverTag()
	}

	args := []string{"describe", "--tags", "--always"}
	if dev {
		args = append(args, "--long")
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("HEAD is not at an exact semver tag; create tag vX.Y.Z before publishing")
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

func ToPEP440(version string) (string, error) {
	v := strings.TrimSpace(version)
	if v == "" {
		return "", fmt.Errorf("version is empty")
	}

	parts := strings.SplitN(v, "-dev.", 2)
	if len(parts) == 2 {
		devParts := strings.Split(parts[1], ".")
		if len(devParts) >= 2 {
			commit := strings.TrimPrefix(devParts[1], "g")
			return fmt.Sprintf("%s.dev%s+%s", parts[0], devParts[0], commit), nil
		}
		return fmt.Sprintf("%s.dev%s", parts[0], strings.ReplaceAll(parts[1], ".", "")), nil
	}

	if strings.Contains(v, "-") {
		return "", fmt.Errorf("version %q is not PEP 440 compatible", v)
	}

	return v, nil
}

func NormalizePythonDistributionName(pkg string) string {
	name := strings.TrimSpace(pkg)
	name = strings.TrimPrefix(name, "@")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

func NormalizeGoTarget(target config.Target) (goOS string, goArch string) {
	goOS = config.MapOSToGo(target.OS)
	goArch = config.MapArchFromNPM(target.Arch)
	return goOS, goArch
}

func BinaryName(toolName string, goOS string) string {
	if goOS == "windows" {
		return toolName + ".exe"
	}
	return toolName
}

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

func WheelFilename(pkg, version string, target config.Target, linuxTag string) (string, error) {
	platformTag, err := WheelPlatformTag(target, linuxTag)
	if err != nil {
		return "", err
	}
	distName := NormalizePythonDistributionName(pkg)
	return fmt.Sprintf("%s-%s-py3-none-%s.whl", distName, version, platformTag), nil
}

func WheelBinaryPath(pkg, toolName string, goOS string) string {
	return path.Join(NormalizePythonDistributionName(pkg), "bin", BinaryName(toolName, goOS))
}
