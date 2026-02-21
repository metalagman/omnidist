package npm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

type StageOptions struct {
	Dev bool
}

type PublishOptions struct {
	DryRun   bool
	Tag      string
	Registry string
	OTP      string
}

type VerificationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func CheckAuth(cfg *config.Config, registryOverride string) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}

	npmrcPath, err := ensureWorkspaceNPMRC(resolveRegistry(npmDist.Registry, registryOverride))
	if err != nil {
		return fmt.Errorf("prepare npmrc: %w", err)
	}
	workspaceDir, err := ensureWorkingDir(paths.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("resolve npm auth working directory: %w", err)
	}

	cmd := exec.Command("npm", "whoami")
	cmd.Dir = workspaceDir
	cmd.Env = append(os.Environ(), "NPM_CONFIG_USERCONFIG="+npmrcPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func Stage(cfg *config.Config, opts StageOptions) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}

	version, err := shared.ResolveVersion(cfg, opts.Dev)
	if err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		if err := stagePlatformPackage(cfg, npmDist, target, version); err != nil {
			return fmt.Errorf("failed to stage %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	if err := stageMetaPackage(cfg, npmDist, version); err != nil {
		return fmt.Errorf("failed to stage meta package: %w", err)
	}

	return nil
}

func Verify(cfg *config.Config) *VerificationResult {
	result := &VerificationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	npmDist, err := npmDistribution(cfg)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
		return result
	}

	version, err := shared.ResolveVersion(cfg, false)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
		return result
	}

	if err := verifyPlatformPackages(cfg, npmDist, version, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	if err := verifyMetaPackage(cfg, npmDist, version, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	return result
}

func Publish(cfg *config.Config, opts PublishOptions) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}

	npmrcPath, err := ensureWorkspaceNPMRC(resolveRegistry(npmDist.Registry, opts.Registry))
	if err != nil {
		return fmt.Errorf("prepare npmrc: %w", err)
	}

	platformPackages := []string{}
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(npmDist.Package, target)
		platformPackages = append(platformPackages, pkgName)
	}

	access := npmDist.Access
	if access == "" {
		access = "public"
	}

	fmt.Println("Publishing platform packages first...")
	for _, pkgName := range platformPackages {
		pkgDir := filepath.Join(paths.NPMDir, pkgName)
		if err := publishPackage(pkgDir, npmDist.Registry, access, opts, npmrcPath); err != nil {
			return fmt.Errorf("failed to publish %s: %w", pkgName, err)
		}
		fmt.Printf("Published: %s\n", pkgName)
	}

	fmt.Println("Publishing meta package...")
	metaDir := filepath.Join(paths.NPMDir, npmDist.Package)
	if err := publishPackage(metaDir, npmDist.Registry, access, opts, npmrcPath); err != nil {
		return fmt.Errorf("failed to publish meta package: %w", err)
	}
	fmt.Printf("Published: %s\n", npmDist.Package)

	return nil
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

func stagePlatformPackage(cfg *config.Config, npmDist config.DistributionConfig, target config.Target, version string) error {
	pkgName := platformPackageName(npmDist.Package, target)
	pkgDir := filepath.Join(paths.NPMDir, pkgName)

	if err := os.MkdirAll(filepath.Join(pkgDir, "bin"), 0755); err != nil {
		return err
	}

	binaryName := cfg.Tool.Name
	if target.OS == "win32" {
		binaryName += ".exe"
	}

	srcPath := filepath.Join(paths.DistDir, target.OS, config.MapArchToNPM(target.Arch), binaryName)
	dstPath := filepath.Join(pkgDir, "bin", binaryName)

	if err := copyFile(srcPath, dstPath); err != nil {
		return err
	}

	if target.OS != "win32" {
		if err := os.Chmod(dstPath, 0755); err != nil {
			return err
		}
	}

	pkgJSON := map[string]interface{}{
		"name":        pkgName,
		"version":     version,
		"description": npmDist.Package + " binary for " + target.OS + "/" + target.Arch,
		"os":          []string{target.OS},
		"cpu":         []string{config.MapArchToNPM(target.Arch)},
		"bin": map[string]string{
			cfg.Tool.Name: "bin/" + binaryName,
		},
		"files": []string{"bin"},
	}

	return writePackageJSON(pkgDir, pkgJSON)
}

func stageMetaPackage(cfg *config.Config, npmDist config.DistributionConfig, version string) error {
	metaDir := filepath.Join(paths.NPMDir, npmDist.Package)

	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	optionalDeps := make(map[string]string)
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(npmDist.Package, target)
		optionalDeps[pkgName] = version
	}

	pkgJSON := map[string]interface{}{
		"name":                 npmDist.Package,
		"version":              version,
		"description":          "Meta package for " + cfg.Tool.Name,
		"bin":                  map[string]string{cfg.Tool.Name: cfg.Tool.Name + ".js"},
		"optionalDependencies": optionalDeps,
		"engines":              map[string]string{"node": ">=16"},
		"files":                []string{cfg.Tool.Name + ".js"},
	}

	if err := writePackageJSON(metaDir, pkgJSON); err != nil {
		return err
	}

	shimPath := filepath.Join(metaDir, cfg.Tool.Name+".js")
	if err := writeShim(shimPath, cfg.Tool.Name, npmDist.Package); err != nil {
		return err
	}

	return nil
}

func verifyPlatformPackages(cfg *config.Config, npmDist config.DistributionConfig, version string, result *VerificationResult) error {
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(npmDist.Package, target)
		pkgDir := filepath.Join(paths.NPMDir, pkgName)

		pkgJSON, err := readPackageJSON(pkgDir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Missing package.json for %s", pkgName))
			result.Valid = false
			continue
		}

		if pkgJSON["version"] != version {
			result.Errors = append(result.Errors, fmt.Sprintf("Version mismatch in %s: got %s, expected %s", pkgName, pkgJSON["version"], version))
			result.Valid = false
		}

		osList, ok := pkgJSON["os"].([]interface{})
		if !ok || len(osList) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Missing os field in %s", pkgName))
			result.Valid = false
		} else if osList[0] != target.OS {
			result.Errors = append(result.Errors, fmt.Sprintf("os mismatch in %s: got %v, expected %s", pkgName, osList, target.OS))
			result.Valid = false
		}

		cpuList, ok := pkgJSON["cpu"].([]interface{})
		if !ok || len(cpuList) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Missing cpu field in %s", pkgName))
			result.Valid = false
		} else if cpuList[0] != config.MapArchToNPM(target.Arch) {
			result.Errors = append(result.Errors, fmt.Sprintf("cpu mismatch in %s: got %v, expected %s", pkgName, cpuList, config.MapArchToNPM(target.Arch)))
			result.Valid = false
		}

		binaryName := cfg.Tool.Name
		if target.OS == "win32" {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(pkgDir, "bin", binaryName)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("Missing binary %s in %s", binaryName, pkgName))
			result.Valid = false
		}

		if scripts, ok := pkgJSON["scripts"].(map[string]interface{}); ok {
			if _, hasPostinstall := scripts["postinstall"]; hasPostinstall {
				result.Errors = append(result.Errors, fmt.Sprintf("Scripts.postinstall found in %s (not allowed)", pkgName))
				result.Valid = false
			}
		}
	}

	return nil
}

func verifyMetaPackage(cfg *config.Config, npmDist config.DistributionConfig, version string, result *VerificationResult) error {
	metaDir := filepath.Join(paths.NPMDir, npmDist.Package)

	pkgJSON, err := readPackageJSON(metaDir)
	if err != nil {
		result.Errors = append(result.Errors, "Missing meta package.json")
		result.Valid = false
		return err
	}

	if pkgJSON["version"] != version {
		result.Errors = append(result.Errors, fmt.Sprintf("Meta package version mismatch: got %s, expected %s", pkgJSON["version"], version))
		result.Valid = false
	}

	if scripts, ok := pkgJSON["scripts"].(map[string]interface{}); ok {
		if _, hasPostinstall := scripts["postinstall"]; hasPostinstall {
			result.Errors = append(result.Errors, "Scripts.postinstall found in meta package (not allowed)")
			result.Valid = false
		}
	}

	optionalDeps, ok := pkgJSON["optionalDependencies"].(map[string]interface{})
	if !ok {
		result.Errors = append(result.Errors, "Missing optionalDependencies in meta package")
		result.Valid = false
	} else {
		for _, target := range cfg.Targets {
			pkgName := platformPackageName(npmDist.Package, target)
			if _, exists := optionalDeps[pkgName]; !exists {
				result.Errors = append(result.Errors, fmt.Sprintf("Missing %s in optionalDependencies", pkgName))
				result.Valid = false
			} else if optionalDeps[pkgName] != version {
				result.Errors = append(result.Errors, fmt.Sprintf("Version mismatch for %s in optionalDependencies: got %s, expected %s", pkgName, optionalDeps[pkgName], version))
				result.Valid = false
			}
		}
	}

	shimPath := filepath.Join(metaDir, cfg.Tool.Name+".js")
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		result.Errors = append(result.Errors, "Missing shim in meta package")
		result.Valid = false
	}

	return nil
}

func resolveRegistry(defaultRegistry, overrideRegistry string) string {
	registry := strings.TrimSpace(overrideRegistry)
	if registry != "" {
		return registry
	}
	registry = strings.TrimSpace(defaultRegistry)
	if registry != "" {
		return registry
	}
	return "https://registry.npmjs.org"
}

func ensureWorkspaceNPMRC(registry string) (string, error) {
	tokenKey, err := npmTokenConfigKey(registry)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(paths.WorkspaceDir, 0755); err != nil {
		return "", err
	}

	content := fmt.Sprintf(
		"# omnidist npm auth (uses npm env substitution)\nregistry=%s\n%s=${NPM_TOKEN}\n",
		registry,
		tokenKey,
	)
	if err := os.WriteFile(paths.NPMRCPath, []byte(content), 0644); err != nil {
		return "", err
	}

	npmrcPath, err := filepath.Abs(paths.NPMRCPath)
	if err != nil {
		return "", err
	}
	return npmrcPath, nil
}

func ensureWorkingDir(dir string) (string, error) {
	cleaned := strings.TrimSpace(dir)
	if cleaned == "" {
		return "", fmt.Errorf("working directory is empty")
	}

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}

	return abs, nil
}

func npmTokenConfigKey(registry string) (string, error) {
	raw := strings.TrimSpace(registry)
	if raw == "" {
		return "", fmt.Errorf("npm registry is empty")
	}

	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	} else if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse npm registry %q: %w", registry, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("parse npm registry %q: missing host", registry)
	}

	pathPart := strings.Trim(strings.TrimSpace(u.Path), "/")
	if pathPart == "" {
		return fmt.Sprintf("//%s/:_authToken", u.Host), nil
	}

	return fmt.Sprintf("//%s/%s/:_authToken", u.Host, pathPart), nil
}

func buildPublishArgs(defaultRegistry, defaultAccess string, opts PublishOptions) []string {
	args := []string{"publish"}

	access := strings.TrimSpace(defaultAccess)
	if access == "" {
		access = "public"
	}
	args = append(args, "--access", access)

	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.Tag != "" {
		args = append(args, "--tag", opts.Tag)
	}

	registry := strings.TrimSpace(defaultRegistry)
	if opts.Registry != "" {
		registry = strings.TrimSpace(opts.Registry)
	}
	if registry != "" {
		args = append(args, "--registry", registry)
	}
	if opts.OTP != "" {
		args = append(args, "--otp", opts.OTP)
	}

	return args
}

func publishPackage(dir, defaultRegistry, defaultAccess string, opts PublishOptions, npmrcPath string) error {
	args := buildPublishArgs(defaultRegistry, defaultAccess, opts)
	packageDir, err := ensureWorkingDir(dir)
	if err != nil {
		return fmt.Errorf("resolve package working directory %q: %w", dir, err)
	}

	execCmd := exec.Command("npm", args...)
	execCmd.Dir = packageDir
	execCmd.Env = append(os.Environ(), "NPM_CONFIG_USERCONFIG="+npmrcPath)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}
