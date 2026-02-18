package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/metalagman/go2npm/internal/config"
	"github.com/spf13/cobra"
)

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Assemble npm packages from built artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		if err := runStage(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error staging:", err)
			os.Exit(1)
		}

		fmt.Println("Staging completed successfully")
	},
}

func runStage(cfg *config.Config) error {
	version := getVersion(cfg)

	for _, target := range cfg.Targets {
		if err := stagePlatformPackage(cfg, target, version); err != nil {
			return fmt.Errorf("failed to stage %s/%s: %w", target.OS, target.CPU, err)
		}
	}

	if err := stageMetaPackage(cfg, version); err != nil {
		return fmt.Errorf("failed to stage meta package: %w", err)
	}

	return nil
}

func stagePlatformPackage(cfg *config.Config, target config.Target, version string) error {
	pkgName := platformPackageName(cfg.NPM.Package, target)
	pkgDir := filepath.Join("npm", pkgName)

	if err := os.MkdirAll(filepath.Join(pkgDir, "bin"), 0755); err != nil {
		return err
	}

	binaryName := cfg.Tool.Name
	if target.OS == "win32" {
		binaryName += ".exe"
	}

	srcPath := filepath.Join("dist", target.OS, target.CPU, binaryName)
	dstPath := filepath.Join(pkgDir, "bin", binaryName)

	if err := copyFile(srcPath, dstPath); err != nil {
		return err
	}

	if target.OS != "win32" {
		os.Chmod(dstPath, 0755)
	}

	pkgJSON := map[string]interface{}{
		"name":        pkgName,
		"version":     version,
		"description": cfg.NPM.Package + " binary for " + target.OS + "/" + target.CPU,
		"os":          []string{target.OS},
		"cpu":         []string{config.MapCPUToNPM(target.CPU)},
		"bin": map[string]string{
			cfg.Tool.Name: "bin/" + binaryName,
		},
		"files": []string{"bin"},
	}

	return writePackageJSON(pkgDir, pkgJSON)
}

func stageMetaPackage(cfg *config.Config, version string) error {
	metaDir := filepath.Join("npm", cfg.NPM.Package)

	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	optionalDeps := make(map[string]string)
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(cfg.NPM.Package, target)
		optionalDeps[pkgName] = version
	}

	pkgJSON := map[string]interface{}{
		"name":                 cfg.NPM.Package,
		"version":              version,
		"description":          "Meta package for " + cfg.Tool.Name,
		"bin":                  cfg.Tool.Name,
		"optionalDependencies": optionalDeps,
		"engines":              map[string]string{"node": ">=16"},
	}

	if err := writePackageJSON(metaDir, pkgJSON); err != nil {
		return err
	}

	shimPath := filepath.Join(metaDir, cfg.Tool.Name+".js")
	if err := writeShim(shimPath, cfg.Tool.Name, cfg.NPM.Package); err != nil {
		return err
	}

	return nil
}

func platformPackageName(meta string, target config.Target) string {
	name := meta + "-" + target.OS + "-" + config.MapCPUToNPM(target.CPU)
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

const packageName = '%s-' + platformKey;
const binDir = path.join(__dirname, '..', packageName, 'bin');
const binaryName = platform === 'win32' ? '%s.exe' : '%s';
const binaryPath = path.join(binDir, binaryName);

try {
	const { execFileSync } = require('child_process');
	process.exit(execFileSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' }));
} catch (e) {
	if (e.code === 'ENOENT') {
		console.error('Binary not found: ' + binaryPath);
		console.error('Expected platform package: ' + packageName);
		console.error('This may be an unsupported platform or installation issue.');
		process.exit(1);
	}
	process.exit(e.status || 1);
}
`, toolName, metaPackage, toolName, toolName)

	return os.WriteFile(path, []byte(shim), 0755)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

func getVersion(cfg *config.Config) string {
	switch cfg.Version.Source {
	case "file":
		if data, err := os.ReadFile("VERSION"); err == nil {
			return string(data)
		}
	case "env":
		if v := os.Getenv("VERSION"); v != "" {
			return v
		}
	}
	return "0.0.0"
}

func loadConfigForStage() (*config.Config, error) {
	return loadConfig()
}

func init() {
	AddCommand(stageCmd)
}
