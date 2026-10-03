package npm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestNPMProjectMetadata(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherit", true: "override"}[override], func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv(shared.EnvVersionName, "1.2.3")
			cfg := testConfig()
			cfg.Description, cfg.Keywords, cfg.License = "Project", []string{"project", "cli"}, "MIT"
			cfg.Distributions.NPM.Package = "omnidist"
			cfg.Distributions.NPM.Aliases = []string{"@omnidist/omnidist"}
			wantDescription, wantKeywords, wantLicense := "Project", []string{"project", "cli"}, "MIT"
			if override {
				cfg.Distributions.NPM.Description = "Channel"
				cfg.Distributions.NPM.Keywords = []string{"channel"}
				cfg.Distributions.NPM.License = "Apache-2.0"
				wantDescription, wantKeywords, wantLicense = "Channel", []string{"channel"}, "Apache-2.0"
			}
			if err := createDistArtifacts(cfg); err != nil {
				t.Fatal(err)
			}
			if err := shared.WriteBuildVersionForConfig(cfg, "1.2.3"); err != nil {
				t.Fatal(err)
			}
			if err := Stage(cfg, StageOptions{}); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"omnidist", "@omnidist/omnidist"} {
				dir := filepath.Join(paths.NPMDir, name)
				pkg, err := readPackageJSON(dir)
				if err != nil {
					t.Fatal(err)
				}
				if pkg["description"] != wantDescription {
					t.Fatalf("%s description = %v", name, pkg["description"])
				}
				assertNPMPackageKeywordsEquals(t, dir, wantKeywords)
				assertNPMPackageLicenseEquals(t, dir, wantLicense)
			}
			for _, target := range cfg.Targets {
				dist, err := cfg.RequireNPM()
				if err != nil {
					t.Fatal(err)
				}
				dir := filepath.Join(paths.NPMDir, platformPackageName(dist.PlatformPackage, target))
				assertNPMPackageLicenseEquals(t, dir, wantLicense)
				pkg, err := readPackageJSON(dir)
				if err != nil {
					t.Fatal(err)
				}
				if pkg["description"] == wantDescription {
					t.Fatal("platform description lost binary context")
				}
			}
			if result := Verify(cfg); !result.Valid {
				t.Fatalf("Verify metadata = %#v", result)
			}
			if !override {
				cfg.Description, cfg.Keywords, cfg.License = "Changed", []string{"changed"}, "Apache-2.0"
				result := Verify(cfg)
				for _, field := range []string{"description mismatch", "keywords mismatch", "license mismatch"} {
					if result.Valid || !strings.Contains(strings.Join(result.Errors, "\n"), field) {
						t.Fatalf("Verify stale %s = %#v", field, result)
					}
				}
			}
		})
	}
}

func TestNPMClearMetadata(t *testing.T) {
	for _, licenseFile := range []bool{false, true} {
		t.Run(map[bool]string{false: "no license file", true: "infer license file"}[licenseFile], func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv(shared.EnvVersionName, "1.2.3")
			if err := os.MkdirAll(".omnidist", 0755); err != nil {
				t.Fatal(err)
			}
			const source = "description: Project\nkeywords: [project]\nlicense: MIT\ntool:\n  name: omnidist\nversion:\n  source: env\ntargets:\n  - os: linux\n    arch: amd64\ndistributions:\n  npm:\n    package: omnidist\n    aliases: ['@omnidist/omnidist']\n    description: ''\n    keywords: []\n    license: ''\n"
			if err := os.WriteFile(paths.ConfigPath, []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			if licenseFile {
				if err := os.WriteFile("LICENSE", []byte("Project license text"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.Load(paths.ConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := createDistArtifacts(cfg); err != nil {
				t.Fatal(err)
			}
			if err := shared.WriteBuildVersionForConfig(cfg, "1.2.3"); err != nil {
				t.Fatal(err)
			}
			if err := Stage(cfg, StageOptions{}); err != nil {
				t.Fatal(err)
			}
			if result := Verify(cfg); !result.Valid {
				t.Fatalf("Verify cleared metadata = %#v", result)
			}
			for _, name := range []string{"omnidist", "@omnidist/omnidist", "omnidist-linux-x64"} {
				dir := filepath.Join(paths.NPMDir, name)
				pkg, err := readPackageJSON(dir)
				if err != nil {
					t.Fatal(err)
				}
				if licenseFile && pkg["license"] != "SEE LICENSE IN LICENSE" || !licenseFile && pkg["license"] != nil {
					t.Fatalf("%s cleared license = %v", name, pkg["license"])
				}
				if name != "omnidist-linux-x64" {
					if pkg["description"] != "Meta package for omnidist" || pkg["keywords"] != nil {
						t.Fatalf("cleared meta package = %#v", pkg)
					}
					pkg["keywords"] = []string{"project"}
				}
				pkg["license"] = "MIT"
				if err := writePackageJSON(dir, pkg); err != nil {
					t.Fatal(err)
				}
			}
			result := Verify(cfg)
			for _, field := range []string{"keywords mismatch", "license mismatch"} {
				if result.Valid || !strings.Contains(strings.Join(result.Errors, "\n"), field) {
					t.Fatalf("Verify stale clear %s = %#v", field, result)
				}
			}
		})
	}
}
