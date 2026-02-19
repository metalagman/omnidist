package npm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/spf13/viper"
)

func loadConfig() (*config.Config, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = "omnidist.yaml"
	}
	return config.Load(configFile)
}

func npmDistribution(cfg *config.Config) (config.DistributionConfig, error) {
	if cfg == nil {
		return config.DistributionConfig{}, fmt.Errorf("config is nil")
	}
	dist, ok := cfg.Distributions["npm"]
	if !ok {
		return config.DistributionConfig{}, fmt.Errorf("missing required distribution: npm")
	}

	dist.Package = strings.TrimSpace(dist.Package)
	dist.Registry = strings.TrimSpace(dist.Registry)
	dist.Access = strings.TrimSpace(dist.Access)
	if dist.Package == "" {
		return config.DistributionConfig{}, fmt.Errorf("npm distribution package is required")
	}
	if dist.Access != "" && dist.Access != "public" && dist.Access != "restricted" {
		return config.DistributionConfig{}, fmt.Errorf("invalid npm access %q: expected public or restricted", dist.Access)
	}
	return dist, nil
}

func getVersion(cfg *config.Config, dev bool) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	var version string
	switch cfg.Version.Source {
	case "git-tag":
		v, err := getGitTagVersion(dev)
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

func getGitTagVersion(dev bool) (string, error) {
	args := []string{"describe", "--tags", "--always"}
	if dev {
		// Include commit count since tag for dev versions
		args = append(args, "--long")
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version := string(bytes.TrimSpace(out))
	// Remove 'v' prefix if present
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}
	// For dev builds, convert git describe format (0.1.0-5-gabc123) to semver prerelease (0.1.0-dev.5.gabc123)
	if dev && len(version) > 0 {
		// git describe --long produces: tag-commits-gsha
		// We want: tag-dev.commits.gsha
		parts := bytes.Split([]byte(version), []byte("-"))
		if len(parts) >= 3 {
			// parts[2] already includes the 'g' prefix from git describe
			version = string(parts[0]) + "-dev." + string(parts[1]) + "." + string(parts[2])
		}
	}
	return version, nil
}

func platformPackageName(meta string, target config.Target) string {
	name := meta + "-" + target.OS + "-" + config.MapArchToNPM(target.Arch)
	if target.Variant != "" {
		name += "-" + target.Variant
	}
	return name
}

func writePackageJSON(dir string, data map[string]interface{}) error {
	f, err := os.Create(filepath.Join(dir, "package.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func writeShim(path, toolName, metaPackage string) error {
	shim := fmt.Sprintf(`#!/usr/bin/env node
const path = require('path');
const os = require('os');

const platform = os.platform();
const arch = os.arch();

const platformMap = {
	'darwin': { x64: 'darwin-x64', arm64: 'darwin-arm64' },
	'linux': { x64: 'linux-x64', arm64: 'linux-arm64' },
	'win32': { x64: 'win32-x64' }
};

const archMap = { x64: 'x64', arm64: 'arm64', ia32: 'x86' };
const cpu = archMap[arch] || arch;

const osKey = platform === 'win32' ? 'win32' : platform;
const platformKey = platformMap[osKey]?.[cpu];

if (!platformKey) {
	console.error('Unsupported platform: ' + platform + '/' + cpu);
	console.error('Expected package: %s-<os>-<cpu>');
	process.exit(1);
}

const platformPkgName = '%s-' + platformKey;
const binaryName = platform === 'win32' ? '%s.exe' : '%s';

try {
	const { execFileSync } = require('child_process');
	const platformPkgJSON = require.resolve(platformPkgName + '/package.json', { paths: [__dirname] });
	const platformPkgDir = path.dirname(platformPkgJSON);
	const binaryPath = path.join(platformPkgDir, 'bin', binaryName);
	process.exit(execFileSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' }));
} catch (e) {
	if (e.code === 'ENOENT') {
		console.error('Binary not found in package: ' + platformPkgName);
		console.error('Expected platform package: ' + '%s-' + platformKey);
		console.error('This may be an unsupported platform or installation issue.');
		process.exit(1);
	}
	if (e.code === 'MODULE_NOT_FOUND') {
		console.error('Platform package not installed: ' + platformPkgName);
		console.error('Expected platform package: ' + '%s-' + platformKey);
		console.error('Try reinstalling the package, and ensure optional dependencies are enabled.');
		process.exit(1);
	}
	process.exit(e.status || 1);
}
`, toolName, metaPackage, toolName, toolName, metaPackage, metaPackage)

	return os.WriteFile(path, []byte(shim), 0755)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

func readPackageJSON(dir string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return pkg, nil
}

func checkAuth() error {
	cmd := exec.Command("npm", "whoami")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", stderr.String())
	}
	return nil
}
