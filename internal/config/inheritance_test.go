package config

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type metadataValues struct {
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
	License     string   `yaml:"license"`
}

func TestProjectMetadataInheritanceRoundTrip(t *testing.T) {
	for _, profile := range []bool{false, true} {
		for _, backend := range []string{"npm", "uv", "pypi", "gem"} {
			for _, tc := range []struct {
				name, override string
				want           metadataValues
				present        []string
			}{
				{"inherit", "", metadataValues{"Project description", []string{"golang", "cli"}, "MIT"}, nil},
				{"null inherits", "    description: null\n    keywords: null\n    license: null\n", metadataValues{"Project description", []string{"golang", "cli"}, "MIT"}, nil},
				{"description only", "    description: ' Channel description '\n", metadataValues{"Channel description", []string{"golang", "cli"}, "MIT"}, []string{"description"}},
				{"keywords replace", "    keywords: ['release', ' release ']\n", metadataValues{"Project description", []string{"release"}, "MIT"}, []string{"keywords"}},
				{"license only", "    license: ' Apache-2.0 '\n", metadataValues{"Project description", []string{"golang", "cli"}, "Apache-2.0"}, []string{"license"}},
				{"clear", "    description: ''\n    keywords: []\n    license: ''\n", metadataValues{}, []string{"description", "keywords", "license"}},
				{"blank clears", "    description: '  '\n    keywords: [' ', '']\n    license: '  '\n", metadataValues{}, []string{"description", "keywords", "license"}},
			} {
				t.Run(backend+"/profile="+strconv.FormatBool(profile)+"/"+tc.name, func(t *testing.T) {
					path := writeProjectMetadataConfig(t, backend, tc.override, profile)
					for range 2 {
						cfg, err := Load(path)
						if err != nil {
							t.Fatal(err)
						}
						got := effectiveMetadata(t, cfg, backend)
						if got.Description != tc.want.Description || got.License != tc.want.License || !slices.Equal(got.Keywords, tc.want.Keywords) {
							t.Fatalf("effective metadata = %#v, want %#v", got, tc.want)
						}
						if profile {
							err = WriteProfiles(path, "default", cfg, true)
						} else {
							err = Save(cfg, path)
						}
						if err != nil {
							t.Fatal(err)
						}
						data, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						var source map[string]interface{}
						if err := yaml.Unmarshal(data, &source); err != nil {
							t.Fatal(err)
						}
						if profile {
							source = source["profiles"].(map[string]interface{})["default"].(map[string]interface{})
						}
						if source["description"] != "Project description" || source["license"] != "MIT" {
							t.Fatalf("project declarations lost: %s", data)
						}
						key := backend
						if key == "uv" {
							key = "pypi"
						}
						dist, ok := source["distributions"].(map[string]interface{})[key].(map[string]interface{})
						if !ok {
							t.Fatalf("missing canonical distribution %s: %s", key, data)
						}
						for _, field := range []string{"description", "keywords", "license"} {
							_, present := dist[field]
							wantPresent := false
							for _, explicit := range tc.present {
								wantPresent = wantPresent || explicit == field
							}
							if present != wantPresent {
								t.Errorf("source %s presence = %v, want %v: %s", field, present, wantPresent, data)
							}
						}
					}
				})
			}
		}
	}
}

func TestProjectMetadataResolutionDoesNotMutateSource(t *testing.T) {
	cfg, err := Load(writeProjectMetadataConfig(t, "npm", "", false))
	if err != nil {
		t.Fatal(err)
	}
	first, err := cfg.RequireNPM()
	if err != nil {
		t.Fatal(err)
	}
	first.Keywords[0] = "changed"
	second, err := cfg.RequireNPM()
	if err != nil {
		t.Fatal(err)
	}
	if second.Keywords[0] != "golang" {
		t.Fatalf("resolved keyword mutation leaked: %v", second.Keywords)
	}
	if cfg.Distributions.NPM.Description != "" || cfg.Distributions.NPM.Keywords != nil || cfg.Distributions.NPM.License != "" {
		t.Fatalf("inherited values were written into source: %#v", cfg.Distributions.NPM)
	}
	cfg.Description = "Updated project"
	cfg.Keywords = []string{"updated"}
	cfg.License = "Apache-2.0"
	got := effectiveMetadata(t, cfg, "npm")
	if got.Description != cfg.Description || !slices.Equal(got.Keywords, cfg.Keywords) || got.License != cfg.License {
		t.Fatalf("project edit not inherited: %#v", got)
	}
}

func TestProjectMetadataDirectConfigAndExplicitEmptyKeywords(t *testing.T) {
	cfg := &Config{
		Description: "Project", Keywords: []string{"project"}, License: "MIT",
		Distributions: DistributionConfigs{NPM: &NPMDistributionConfig{
			Package: "mytool", Description: "Override", Keywords: []string{},
		}},
	}
	got := effectiveMetadata(t, cfg, "npm")
	if got.Description != "Override" || len(got.Keywords) != 0 || got.License != "MIT" {
		t.Fatalf("direct configuration metadata = %#v", got)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := effectiveMetadata(t, loaded, "npm"); len(got.Keywords) != 0 {
		t.Fatalf("empty direct keywords inherited after save: %#v", got)
	}
}

func TestProjectMetadataProfileIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "profiles:\n  first:\n    description: First\n    keywords: [first]\n    license: MIT\n    distributions:\n      npm:\n        package: first\n  second:\n    description: Second\n    keywords: [second]\n    license: Apache-2.0\n    distributions:\n      npm:\n        package: second\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second"} {
		cfg, err := LoadWithProfile(path, name)
		if err != nil {
			t.Fatal(err)
		}
		got := effectiveMetadata(t, cfg, "npm")
		if !slices.Equal(got.Keywords, []string{name}) || strings.ToLower(got.Description) != name {
			t.Fatalf("profile %s metadata = %#v", name, got)
		}
	}
}

func TestProjectMetadataRejectsMixedFormat(t *testing.T) {
	for _, field := range []string{"description: text", "keywords: [cli]", "license: MIT"} {
		t.Run(field, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(field+"\nprofiles:\n  default:\n    distributions:\n      npm:\n        package: mytool\n"), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), "mixed format") {
				t.Fatalf("Load() error = %v, want mixed format error", err)
			}
		})
	}
}

func effectiveMetadata(t *testing.T, cfg *Config, backend string) metadataValues {
	t.Helper()
	var dist interface{}
	var err error
	switch backend {
	case "npm":
		dist, err = cfg.RequireNPM()
	case "uv", "pypi":
		dist, err = cfg.RequireUV()
	case "gem":
		dist, err = cfg.RequireGem()
	default:
		t.Fatalf("unknown backend %s", backend)
	}
	if err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(dist)
	if err != nil {
		t.Fatal(err)
	}
	var got metadataValues
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func writeProjectMetadataConfig(t *testing.T, backend, override string, profile bool) string {
	t.Helper()
	data := "description: ' Project description '\nkeywords: [' golang ', 'cli', 'golang']\nlicense: ' MIT '\ntool:\n  name: mytool\n  main: ./cmd/mytool\ntargets:\n  - os: linux\n    arch: amd64\ndistributions:\n  " + backend + ":\n    package: mytool\n" + override
	if profile {
		data = "profiles:\n  default:\n    " + strings.ReplaceAll(strings.TrimSuffix(data, "\n"), "\n", "\n    ") + "\n"
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
