package npm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

// StageOptions controls npm staging behavior.
type StageOptions struct {
	Dev bool
}

// PublishOptions controls npm publish behavior.
type PublishOptions struct {
	DryRun   bool
	Tag      string
	Registry string
	OTP      string
	Stdout   io.Writer
	Stderr   io.Writer
	Progress io.Writer
}

const (
	npmPublishTokenEnv = "NPM_PUBLISH_TOKEN"
)

var projectLicenseCandidates = []string{"LICENSE", "LICENSE.md", "LICENSE.txt"}
var npmVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// VerificationResult summarizes npm staging validation results.
type VerificationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// CheckAuth validates npm authentication for the configured registry.
func CheckAuth(cfg *config.Config, registryOverride string, dryRun bool) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}
	layout := layoutForConfig(cfg)
	token, err := resolvePublishToken(dryRun)
	if err != nil {
		return err
	}

	npmrcPath, err := ensureWorkspaceNPMRC(layout, resolveRegistry(npmDist.Registry, registryOverride))
	if err != nil {
		return fmt.Errorf("prepare npmrc: %w", err)
	}
	workspaceDir, err := ensureWorkingDir(layout.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("resolve npm auth working directory: %w", err)
	}

	cmd := exec.Command("npm", "whoami")
	cmd.Dir = workspaceDir
	cmd.Env = npmCommandEnv(npmrcPath, token)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return fmt.Errorf("npm whoami failed: %w: %s", err, stderrText)
		}
		return fmt.Errorf("npm whoami failed: %w", err)
	}
	return nil
}

// Stage assembles npm platform packages and the meta package from built artifacts.
func Stage(cfg *config.Config, opts StageOptions) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}
	layout := layoutForConfig(cfg)

	version, err := resolveNPMStageVersion(cfg, opts.Dev)
	if err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		if err := stagePlatformPackage(layout, cfg, npmDist, target, version); err != nil {
			return fmt.Errorf("failed to stage %s/%s: %w", target.OS, target.Arch, err)
		}
	}

	if err := stageMetaPackage(layout, cfg, npmDist, version); err != nil {
		return fmt.Errorf("failed to stage meta package: %w", err)
	}

	return nil
}

func resolveNPMStageVersion(cfg *config.Config, dev bool) (string, error) {
	if dev {
		version, err := shared.ResolveVersion(cfg, true)
		if err != nil {
			return "", err
		}
		return validateNPMVersion(version, "resolved dev version")
	}

	version, err := shared.ReadBuildVersionForConfig(cfg)
	if err != nil {
		layout := layoutForConfig(cfg)
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("missing build version file %s; run `omnidist build` before `omnidist npm stage`", layout.DistVersionPath)
		}
		return "", fmt.Errorf("read build version: %w", err)
	}

	return validateNPMVersion(version, "build version")
}

// Verify validates staged npm packages and returns accumulated findings.
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
	layout := layoutForConfig(cfg)

	metaDir := filepath.Join(layout.NPMDir, npmDist.Package)
	version, err := resolveNPMVersion(cfg, metaDir)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
		return result
	}

	if err := verifyPlatformPackages(layout, cfg, npmDist, version, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	if err := verifyMetaPackage(layout, cfg, npmDist, version, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	return result
}

// Publish publishes staged npm platform and meta packages.
func Publish(cfg *config.Config, opts PublishOptions) error {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return err
	}
	layout := layoutForConfig(cfg)
	token, err := resolvePublishToken(opts.DryRun)
	if err != nil {
		return err
	}

	npmrcPath, err := ensureWorkspaceNPMRC(layout, resolveRegistry(npmDist.Registry, opts.Registry))
	if err != nil {
		return fmt.Errorf("prepare npmrc: %w", err)
	}

	metaDir := filepath.Join(layout.NPMDir, npmDist.Package)
	version, err := resolveNPMVersion(cfg, metaDir)
	if err != nil {
		return fmt.Errorf("resolve npm version: %w", err)
	}

	publishOpts, autoDevTag := withAutoDevTag(opts, version)
	if autoDevTag {
		writeProgressf(opts.Progress, "Detected dev npm version %s, publishing with --tag %q\n", version, publishOpts.Tag)
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

	writeProgressf(opts.Progress, "Publishing platform packages first...\n")
	for _, pkgName := range platformPackages {
		pkgDir := filepath.Join(layout.NPMDir, pkgName)
		if err := publishPackage(pkgDir, npmDist.Registry, access, publishOpts, npmrcPath, token, version); err != nil {
			return fmt.Errorf("failed to publish %s: %w", pkgName, err)
		}
		writeProgressf(opts.Progress, "Published: %s\n", pkgName)
	}

	writeProgressf(opts.Progress, "Publishing meta package...\n")
	if err := publishPackage(metaDir, npmDist.Registry, access, publishOpts, npmrcPath, token, version); err != nil {
		return fmt.Errorf("failed to publish meta package: %w", err)
	}
	writeProgressf(opts.Progress, "Published: %s\n", npmDist.Package)

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
	name := meta + "-" + config.MapGoOSToNPM(target.OS) + "-" + config.MapGoArchToNPM(target.Arch)
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
	return copyFileWithMode(src, dst, 0755)
}

func copyFileWithMode(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

func stageProjectREADME(cfg *config.Config, dist config.DistributionConfig, dstDir string, enabled bool) (bool, error) {
	if !enabled {
		return false, nil
	}

	readmePath := ""
	if cfg != nil {
		readmePath = cfg.ReadmePath
	}
	resolvedPath, required := shared.ResolveProjectREADMEPath(readmePath, dist.ReadmePath)
	data, exists, err := shared.ReadProjectREADME(resolvedPath, required)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if err := os.WriteFile(filepath.Join(dstDir, shared.ProjectREADMEPath), data, 0644); err != nil {
		return false, err
	}
	return true, nil
}

func stageProjectLicense(dstDir string) (string, bool, error) {
	name, data, exists, err := readOptionalProjectLicense()
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}

	if err := os.WriteFile(filepath.Join(dstDir, name), data, 0644); err != nil {
		return "", false, err
	}
	return name, true, nil
}

func readOptionalProjectLicense() (string, []byte, bool, error) {
	for _, candidate := range projectLicenseCandidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return candidate, data, true, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return "", nil, false, fmt.Errorf("read project license %s: %w", candidate, err)
	}
	return "", nil, false, nil
}

func packageLicenseValue(dist config.DistributionConfig, licenseName string, licenseIncluded bool) (string, bool) {
	if license := dist.LicenseValue(); license != "" {
		return license, true
	}
	if licenseIncluded {
		return "SEE LICENSE IN " + licenseName, true
	}
	return "", false
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

func stagePlatformPackage(layout paths.Layout, cfg *config.Config, npmDist config.DistributionConfig, target config.Target, version string) error {
	pkgName := platformPackageName(npmDist.Package, target)
	pkgDir := filepath.Join(layout.NPMDir, pkgName)

	if err := os.MkdirAll(filepath.Join(pkgDir, "bin"), 0755); err != nil {
		return err
	}

	binaryName := cfg.Tool.Name
	if target.OS == "windows" {
		binaryName += ".exe"
	}

	srcPath := filepath.Join(layout.DistDir, target.OS, target.Arch, binaryName)
	dstPath := filepath.Join(pkgDir, "bin", binaryName)

	if err := copyFile(srcPath, dstPath); err != nil {
		return err
	}

	if target.OS != "windows" {
		if err := os.Chmod(dstPath, 0755); err != nil {
			return err
		}
	}

	files := []string{"bin"}
	if included, err := stageProjectREADME(cfg, npmDist, pkgDir, npmDist.IncludeREADMEEnabled()); err != nil {
		return err
	} else if included {
		files = append(files, shared.ProjectREADMEPath)
	}
	licenseName, licenseIncluded, err := stageProjectLicense(pkgDir)
	if err != nil {
		return err
	}
	if licenseIncluded {
		files = append(files, licenseName)
	}

	pkgJSON := map[string]interface{}{
		"name":        pkgName,
		"version":     version,
		"description": npmDist.Package + " binary for " + target.OS + "/" + target.Arch,
		"os":          []string{config.MapGoOSToNPM(target.OS)},
		"cpu":         []string{config.MapGoArchToNPM(target.Arch)},
		"bin": map[string]string{
			cfg.Tool.Name: "bin/" + binaryName,
		},
		"files": files,
	}
	if license, ok := packageLicenseValue(npmDist, licenseName, licenseIncluded); ok {
		pkgJSON["license"] = license
	}

	return writePackageJSON(pkgDir, pkgJSON)
}

func stageMetaPackage(layout paths.Layout, cfg *config.Config, npmDist config.DistributionConfig, version string) error {
	metaDir := filepath.Join(layout.NPMDir, npmDist.Package)

	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	files := []string{cfg.Tool.Name + ".js"}
	if included, err := stageProjectREADME(cfg, npmDist, metaDir, npmDist.IncludeREADMEEnabled()); err != nil {
		return err
	} else if included {
		files = append(files, shared.ProjectREADMEPath)
	}
	licenseName, licenseIncluded, err := stageProjectLicense(metaDir)
	if err != nil {
		return err
	}
	if licenseIncluded {
		files = append(files, licenseName)
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
		"files":                files,
	}
	if license, ok := packageLicenseValue(npmDist, licenseName, licenseIncluded); ok {
		pkgJSON["license"] = license
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

func verifyPlatformPackages(layout paths.Layout, cfg *config.Config, npmDist config.DistributionConfig, version string, result *VerificationResult) error {
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(npmDist.Package, target)
		pkgDir := filepath.Join(layout.NPMDir, pkgName)

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
		} else if osList[0] != config.MapGoOSToNPM(target.OS) {
			result.Errors = append(result.Errors, fmt.Sprintf("os mismatch in %s: got %v, expected %s", pkgName, osList, config.MapGoOSToNPM(target.OS)))
			result.Valid = false
		}

		cpuList, ok := pkgJSON["cpu"].([]interface{})
		if !ok || len(cpuList) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Missing cpu field in %s", pkgName))
			result.Valid = false
		} else if cpuList[0] != config.MapGoArchToNPM(target.Arch) {
			result.Errors = append(result.Errors, fmt.Sprintf("cpu mismatch in %s: got %v, expected %s", pkgName, cpuList, config.MapGoArchToNPM(target.Arch)))
			result.Valid = false
		}

		binaryName := cfg.Tool.Name
		if target.OS == "windows" {
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

		if expectedLicense := npmDist.LicenseValue(); expectedLicense != "" {
			if pkgJSON["license"] != expectedLicense {
				result.Errors = append(result.Errors, fmt.Sprintf("license mismatch in %s: got %v, expected %s", pkgName, pkgJSON["license"], expectedLicense))
				result.Valid = false
			}
		}
	}

	return nil
}

func verifyMetaPackage(layout paths.Layout, cfg *config.Config, npmDist config.DistributionConfig, version string, result *VerificationResult) error {
	metaDir := filepath.Join(layout.NPMDir, npmDist.Package)

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

	if expectedLicense := npmDist.LicenseValue(); expectedLicense != "" {
		if pkgJSON["license"] != expectedLicense {
			result.Errors = append(result.Errors, fmt.Sprintf("Meta package license mismatch: got %v, expected %s", pkgJSON["license"], expectedLicense))
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

func ensureWorkspaceNPMRC(layout paths.Layout, registry string) (string, error) {
	tokenKey, err := npmTokenConfigKey(registry)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(layout.WorkspaceDir, 0755); err != nil {
		return "", err
	}

	content := fmt.Sprintf(
		"# omnidist npm auth (uses npm env substitution)\nregistry=%s\n%s=${NPM_PUBLISH_TOKEN}\n",
		registry,
		tokenKey,
	)
	if err := os.WriteFile(layout.NPMRCPath, []byte(content), 0644); err != nil {
		return "", err
	}

	npmrcPath, err := filepath.Abs(layout.NPMRCPath)
	if err != nil {
		return "", err
	}
	return npmrcPath, nil
}

func layoutForConfig(cfg *config.Config) paths.Layout {
	if cfg == nil {
		return paths.NewLayout(config.DefaultWorkspaceDir)
	}
	return paths.NewLayout(cfg.EffectiveWorkspaceDir())
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

func isDevVersion(version string) bool {
	v := strings.ToLower(strings.TrimSpace(version))
	return strings.Contains(v, "-dev.") || strings.HasSuffix(v, "-dev")
}

func withAutoDevTag(opts PublishOptions, version string) (PublishOptions, bool) {
	publishOpts := opts
	if publishOpts.Tag == "" && isDevVersion(version) {
		publishOpts.Tag = "dev"
		return publishOpts, true
	}
	return publishOpts, false
}

func stagedPackageVersion(dir string) (string, error) {
	pkgJSON, err := readPackageJSON(dir)
	if err != nil {
		return "", err
	}

	value, ok := pkgJSON["version"]
	if !ok {
		return "", fmt.Errorf("missing version in package.json")
	}

	version, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid version in package.json: expected string")
	}

	return validateNPMVersion(version, "staged package.json version")
}

func resolveNPMVersion(cfg *config.Config, metaDir string) (string, error) {
	version, err := stagedPackageVersion(metaDir)
	if err == nil {
		return version, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read staged npm package version: %w", err)
	}

	version, err = shared.ResolveStageVersion(cfg, false)
	if err != nil {
		return "", fmt.Errorf("resolve build/source version: %w", err)
	}
	version, err = validateNPMVersion(version, "build/source version")
	if err != nil {
		return "", err
	}
	return version, nil
}

func resolvePublishToken(dryRun bool) (string, error) {
	token := strings.TrimSpace(os.Getenv(npmPublishTokenEnv))
	if token == "" && !dryRun {
		return "", fmt.Errorf("npm publish requires token auth: set %s", npmPublishTokenEnv)
	}
	return token, nil
}

func npmCommandEnv(npmrcPath string, token string) []string {
	env := append(os.Environ(), "NPM_CONFIG_USERCONFIG="+npmrcPath)
	if token != "" {
		env = append(env, npmPublishTokenEnv+"="+token)
	}
	return env
}

func publishPackage(dir, defaultRegistry, defaultAccess string, opts PublishOptions, npmrcPath string, token string, version string) error {
	publishOpts, _ := withAutoDevTag(opts, version)
	args := buildPublishArgs(defaultRegistry, defaultAccess, publishOpts)
	packageDir, err := ensureWorkingDir(dir)
	if err != nil {
		return fmt.Errorf("resolve package working directory %q: %w", dir, err)
	}

	execCmd := exec.Command("npm", args...)
	execCmd.Dir = packageDir
	execCmd.Env = npmCommandEnv(npmrcPath, token)
	execCmd.Stdout = commandOutputWriter(opts.Stdout)
	execCmd.Stderr = commandOutputWriter(opts.Stderr)

	return execCmd.Run()
}

func commandOutputWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func writeProgressf(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format, args...)
}

func validateNPMVersion(version string, source string) (string, error) {
	v := strings.TrimSpace(version)
	if v == "" {
		return "", fmt.Errorf("empty %s", source)
	}
	if !npmVersionPattern.MatchString(v) {
		return "", fmt.Errorf("invalid npm version %q from %s: expected semver (e.g. 1.2.3 or 1.2.3-dev.4.gabc123)", v, source)
	}
	return v, nil
}
