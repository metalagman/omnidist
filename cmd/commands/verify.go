package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/spf13/cobra"
)

type VerificationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Enforce correctness before publishing",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		result := runVerify(cfg)

		if len(result.Errors) > 0 {
			fmt.Println("Verification FAILED:")
			for _, e := range result.Errors {
				fmt.Println("  ERROR:", e)
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, w := range result.Warnings {
				fmt.Println("  WARN:", w)
			}
		}

		if result.Valid {
			fmt.Println("Verification PASSED")
		} else {
			os.Exit(1)
		}
	},
}

func runVerify(cfg *config.Config) *VerificationResult {
	result := &VerificationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	version := getVersion(cfg)

	if err := verifyPlatformPackages(cfg, version, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	if err := verifyMetaPackage(cfg, version, result); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	return result
}

func verifyPlatformPackages(cfg *config.Config, version string, result *VerificationResult) error {
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
		pkgDir := filepath.Join("npm", pkgName)

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

func verifyMetaPackage(cfg *config.Config, version string, result *VerificationResult) error {
	metaDir := filepath.Join("npm", cfg.Distributions["npm"].Package)

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
			pkgName := platformPackageName(cfg.Distributions["npm"].Package, target)
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

func init() {
	AddCommandTo(npmCmd, verifyCmd)
}
