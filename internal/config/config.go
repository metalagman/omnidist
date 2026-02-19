package config

import (
	"os"
	"path/filepath"

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
}

func (t *Target) NPMName() string {
	return MapArchToNPM(t.Arch)
}

func MapArchToNPM(arch string) string {
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

func MapArchFromNPM(arch string) string {
	switch arch {
	case "x64":
		return "amd64"
	case "x86":
		return "386"
	default:
		return arch
	}
}

func MapOSToGo(os string) string {
	switch os {
	case "win32":
		return "windows"
	default:
		return os
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
			{OS: "win32", Arch: "amd64"},
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

	return &cfg, nil
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
