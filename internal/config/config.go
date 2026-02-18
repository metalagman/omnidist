package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tool    ToolConfig    `yaml:"tool"`
	NPM     NPMConfig     `yaml:"npm"`
	Version VersionConfig `yaml:"version"`
	Targets []Target      `yaml:"targets"`
	Build   BuildConfig   `yaml:"build"`
}

type ToolConfig struct {
	Name string `yaml:"name"`
	Main string `yaml:"main"`
}

type NPMConfig struct {
	Package  string `yaml:"package"`
	Registry string `yaml:"registry"`
	Access   string `yaml:"access"`
}

type VersionConfig struct {
	Source string `yaml:"source"`
}

type Target struct {
	OS      string `yaml:"os"`
	CPU     string `yaml:"cpu"`
	Variant string `yaml:"variant,omitempty"`
}

func (t *Target) NPMName() string {
	return MapCPUToNPM(t.CPU)
}

func MapCPUToNPM(cpu string) string {
	switch cpu {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return cpu
	}
}

func MapCPUFromNPM(cpu string) string {
	switch cpu {
	case "x64":
		return "amd64"
	case "x86":
		return "386"
	default:
		return cpu
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
			Name: "mytool",
			Main: "./cmd/mytool",
		},
		NPM: NPMConfig{
			Package:  "mytool",
			Registry: "https://registry.npmjs.org",
			Access:   "public",
		},
		Version: VersionConfig{
			Source: "git-tag",
		},
		Targets: []Target{
			{OS: "darwin", CPU: "x64"},
			{OS: "darwin", CPU: "arm64"},
			{OS: "linux", CPU: "x64"},
			{OS: "linux", CPU: "arm64"},
			{OS: "win32", CPU: "x64"},
		},
		Build: BuildConfig{
			Ldflags: "-s -w",
			Tags:    []string{},
			CGO:     false,
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
