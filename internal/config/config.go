package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultVersionFile is the default version file path when version.source is file.
	DefaultVersionFile = "VERSION"
)

// Config is the root omnidist configuration loaded from omnidist.yaml.
type Config struct {
	Tool          ToolConfig                    `yaml:"tool"`
	Version       VersionConfig                 `yaml:"version"`
	Targets       []Target                      `yaml:"targets"`
	Build         BuildConfig                   `yaml:"build"`
	Distributions map[string]DistributionConfig `yaml:"distributions"`
}

// ToolConfig configures the Go CLI binary to build and package.
type ToolConfig struct {
	Name string `yaml:"name"`
	Main string `yaml:"main"`
}

// VersionConfig defines where omnidist resolves the release version from.
type VersionConfig struct {
	Source string `yaml:"source"`
	File   string `yaml:"file,omitempty"`
	Fixed  string `yaml:"fixed,omitempty"`
}

// Target describes a Go build target and optional packaging variant.
type Target struct {
	OS      string `yaml:"os"`
	Arch    string `yaml:"arch"`
	Variant string `yaml:"variant,omitempty"`
}

// DistributionConfig stores distribution-specific packaging settings.
type DistributionConfig struct {
	Package       string `yaml:"package"`
	Registry      string `yaml:"registry,omitempty"`
	Access        string `yaml:"access,omitempty"`
	License       string `yaml:"license,omitempty"`
	IndexURL      string `yaml:"index-url,omitempty"`
	LinuxTag      string `yaml:"linux-tag,omitempty"`
	IncludeREADME *bool  `yaml:"include-readme,omitempty"`
}

// IncludeREADMEEnabled reports whether README.md should be included in staged artifacts.
func (d DistributionConfig) IncludeREADMEEnabled() bool {
	if d.IncludeREADME == nil {
		return true
	}
	return *d.IncludeREADME
}

// LicenseValue reports the configured package license value after trimming whitespace.
func (d DistributionConfig) LicenseValue() string {
	return strings.TrimSpace(d.License)
}

// MapGoArchToNPM converts a Go GOARCH value to the corresponding npm cpu value.
func MapGoArchToNPM(arch string) string {
	switch arch {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return arch
	}
}

// MapGoOSToNPM converts a Go GOOS value to the corresponding npm os value.
func MapGoOSToNPM(goOS string) string {
	switch goOS {
	case "windows":
		return "win32"
	default:
		return goOS
	}
}

// BuildConfig configures go build flags shared across targets.
type BuildConfig struct {
	Ldflags string   `yaml:"ldflags"`
	Tags    []string `yaml:"tags"`
	CGO     bool     `yaml:"cgo"`
}

// DefaultConfig returns the default omnidist configuration for a new project.
func DefaultConfig() *Config {
	return &Config{
		Tool: ToolConfig{
			Name: "omnidist",
			Main: "./cmd/omnidist",
		},
		Version: VersionConfig{
			Source: "git-tag",
		},
		Targets: []Target{
			{OS: "darwin", Arch: "amd64"},
			{OS: "darwin", Arch: "arm64"},
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
			{OS: "windows", Arch: "amd64"},
		},
		Build: BuildConfig{
			Ldflags: "-s -w",
			Tags:    []string{},
			CGO:     false,
		},
		Distributions: map[string]DistributionConfig{
			"npm": {
				Package:       "@omnidist/omnidist",
				Registry:      "https://registry.npmjs.org",
				Access:        "public",
				IncludeREADME: boolPtr(true),
			},
			"uv": {
				Package:       "omnidist",
				IndexURL:      "https://upload.pypi.org/legacy/",
				LinuxTag:      "manylinux2014",
				IncludeREADME: boolPtr(true),
			},
		},
	}
}

// Load reads and validates an omnidist configuration file from path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	if hasLegacyFixedVersionKey(data) {
		return nil, fmt.Errorf("version.fixed-version is no longer supported; use version.fixed")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	applyVersionDefaults(&cfg)
	applyDistributionDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyVersionDefaults(cfg *Config) {
	cfg.Version.Source = strings.TrimSpace(cfg.Version.Source)
	cfg.Version.File = strings.TrimSpace(cfg.Version.File)
	cfg.Version.Fixed = strings.TrimSpace(cfg.Version.Fixed)
	if cfg.Version.Source == "" {
		cfg.Version.Source = "git-tag"
	}
	if cfg.Version.Source == "file" && cfg.Version.File == "" {
		cfg.Version.File = DefaultVersionFile
	}
}

func hasLegacyFixedVersionKey(data []byte) bool {
	var raw struct {
		Version map[string]interface{} `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	if raw.Version == nil {
		return false
	}
	_, found := raw.Version["fixed-version"]
	return found
}

func applyDistributionDefaults(cfg *Config) {
	if cfg.Distributions == nil {
		cfg.Distributions = map[string]DistributionConfig{}
	}

	npmDist := cfg.Distributions["npm"]
	npmDist.Package = strings.TrimSpace(npmDist.Package)
	npmDist.Registry = strings.TrimSpace(npmDist.Registry)
	npmDist.Access = strings.TrimSpace(npmDist.Access)
	npmDist.License = npmDist.LicenseValue()
	if npmDist.Registry == "" {
		npmDist.Registry = "https://registry.npmjs.org"
	}
	if npmDist.Access == "" {
		npmDist.Access = "public"
	}
	if npmDist.Package == "" {
		npmDist.Package = "@omnidist/omnidist"
	}
	if npmDist.IncludeREADME == nil {
		npmDist.IncludeREADME = boolPtr(true)
	}
	cfg.Distributions["npm"] = npmDist

	uvDist := cfg.Distributions["uv"]
	uvDist.Package = strings.TrimSpace(uvDist.Package)
	uvDist.IndexURL = strings.TrimSpace(uvDist.IndexURL)
	uvDist.LinuxTag = strings.TrimSpace(uvDist.LinuxTag)
	if uvDist.Package == "" {
		uvDist.Package = "omnidist"
	}
	if uvDist.IndexURL == "" {
		uvDist.IndexURL = "https://upload.pypi.org/legacy/"
	}
	if uvDist.LinuxTag == "" {
		uvDist.LinuxTag = "manylinux2014"
	}
	if uvDist.IncludeREADME == nil {
		uvDist.IncludeREADME = boolPtr(true)
	}
	cfg.Distributions["uv"] = uvDist
}

func boolPtr(v bool) *bool {
	return &v
}

func validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	for i, target := range cfg.Targets {
		if strings.TrimSpace(target.OS) == "" {
			return fmt.Errorf("targets[%d].os is required", i)
		}
		if strings.TrimSpace(target.Arch) == "" {
			return fmt.Errorf("targets[%d].arch is required", i)
		}
		if target.OS == "win32" {
			return fmt.Errorf("invalid targets[%d].os %q: use Go GOOS value %q", i, target.OS, "windows")
		}
		if target.Arch == "x64" {
			return fmt.Errorf("invalid targets[%d].arch %q: use Go GOARCH value %q", i, target.Arch, "amd64")
		}
	}

	source := strings.TrimSpace(cfg.Version.Source)
	if source == "" {
		source = "git-tag"
	}
	switch source {
	case "git-tag", "file", "env", "fixed":
	default:
		return fmt.Errorf("invalid version.source %q: expected git-tag, file, env, or fixed", cfg.Version.Source)
	}
	if source == "fixed" && strings.TrimSpace(cfg.Version.Fixed) == "" {
		return fmt.Errorf("version.fixed is required when version.source is %q", "fixed")
	}

	if npmDist, ok := cfg.Distributions["npm"]; ok {
		switch npmDist.Access {
		case "", "public", "restricted":
		default:
			return fmt.Errorf("invalid distributions.npm.access %q: expected public or restricted", npmDist.Access)
		}
	}

	if uvDist, ok := cfg.Distributions["uv"]; ok {
		if uvDist.Package == "" {
			return fmt.Errorf("distributions.uv.package is required")
		}
		switch uvDist.LinuxTag {
		case "manylinux2014", "musllinux_1_2":
		default:
			return fmt.Errorf("invalid distributions.uv.linux-tag %q: expected manylinux2014 or musllinux_1_2", uvDist.LinuxTag)
		}
	}

	return nil
}

// Save writes cfg to path in YAML format, creating parent directories as needed.
func Save(cfg *Config, path string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config for %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}
