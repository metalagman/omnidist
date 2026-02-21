package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tool          ToolConfig                    `yaml:"tool"`
	Version       VersionConfig                 `yaml:"version"`
	Targets       []Target                      `yaml:"targets"`
	Build         BuildConfig                   `yaml:"build"`
	Distributions map[string]DistributionConfig `yaml:"distributions"`
}

type ToolConfig struct {
	Name string `yaml:"name"`
	Main string `yaml:"main"`
}

type VersionConfig struct {
	Source string `yaml:"source"`
}

type Target struct {
	OS      string `yaml:"os"`
	Arch    string `yaml:"arch"`
	Variant string `yaml:"variant,omitempty"`
}

type DistributionConfig struct {
	Package  string `yaml:"package"`
	Registry string `yaml:"registry,omitempty"`
	Access   string `yaml:"access,omitempty"`
	IndexURL string `yaml:"index-url,omitempty"`
	LinuxTag string `yaml:"linux-tag,omitempty"`
}

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

func MapGoOSToNPM(goOS string) string {
	switch goOS {
	case "windows":
		return "win32"
	default:
		return goOS
	}
}

type BuildConfig struct {
	Ldflags string   `yaml:"ldflags"`
	Tags    []string `yaml:"tags"`
	CGO     bool     `yaml:"cgo"`
}

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
				Package:  "@omnidist/omnidist",
				Registry: "https://registry.npmjs.org",
				Access:   "public",
			},
			"uv": {
				Package:  "omnidist",
				IndexURL: "https://upload.pypi.org/legacy/",
				LinuxTag: "manylinux2014",
			},
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyDistributionDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDistributionDefaults(cfg *Config) {
	if cfg.Distributions == nil {
		cfg.Distributions = map[string]DistributionConfig{}
	}

	npmDist := cfg.Distributions["npm"]
	npmDist.Package = strings.TrimSpace(npmDist.Package)
	npmDist.Registry = strings.TrimSpace(npmDist.Registry)
	npmDist.Access = strings.TrimSpace(npmDist.Access)
	if npmDist.Registry == "" {
		npmDist.Registry = "https://registry.npmjs.org"
	}
	if npmDist.Access == "" {
		npmDist.Access = "public"
	}
	if npmDist.Package == "" {
		npmDist.Package = "@omnidist/omnidist"
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
	cfg.Distributions["uv"] = uvDist
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

func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
