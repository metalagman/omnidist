package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDistributionMetadataRoundTrip(t *testing.T) {
	for _, profile := range []bool{false, true} {
		for _, backend := range []string{"uv", "gem"} {
			t.Run(backend+"/profile="+strconv.FormatBool(profile), func(t *testing.T) {
				path := writeMetadataConfig(t, backend, "    description: '  Distribute Go binaries  '\n    keywords: [' cli ', '', 'golang', 'cli', '  ']\n", profile)
				cfg, err := Load(path)
				if err != nil {
					t.Fatalf("Load() error = %v", err)
				}
				if profile {
					err = WriteProfiles(path, "default", cfg, true)
				} else {
					err = Save(cfg, path)
				}
				if err != nil {
					t.Fatalf("save config: %v", err)
				}
				cfg, err = Load(path)
				if err != nil {
					t.Fatalf("Load() after Save() error = %v", err)
				}
				if cfg.Runtime.ProfilesMode != profile {
					t.Fatalf("profiles mode = %v, want %v", cfg.Runtime.ProfilesMode, profile)
				}
				data, err := yaml.Marshal(cfg.Distributions)
				if err != nil {
					t.Fatal(err)
				}
				var distributions map[string]struct {
					Description string   `yaml:"description"`
					Keywords    []string `yaml:"keywords"`
				}
				if err := yaml.Unmarshal(data, &distributions); err != nil {
					t.Fatal(err)
				}
				key := backend
				if key == "uv" {
					key = "pypi"
				}
				got := distributions[key]
				if got.Description != "Distribute Go binaries" || !reflect.DeepEqual(got.Keywords, []string{"cli", "golang"}) {
					t.Fatalf("roundtrip metadata = %#v, want trimmed description and unique non-empty keywords", got)
				}
			})
		}
	}
}

func TestDistributionMetadataValidation(t *testing.T) {
	for _, backend := range []string{"uv", "gem"} {
		for _, tc := range []struct {
			name, field, value string
		}{
			{"description newline", "description", "Build\nInjected: header"},
			{"description carriage return", "description", "Build\rInjected: header"},
			{"keyword newline", "keywords", "cli\nInjected: header"},
			{"keyword carriage return", "keywords", "cli\rInjected: header"},
			{"keyword comma", "keywords", "cli,golang"},
		} {
			t.Run(backend+"/"+tc.name, func(t *testing.T) {
				value := strconv.Quote(tc.value)
				if tc.field == "keywords" {
					value = "[" + value + "]"
				}
				path := writeMetadataConfig(t, backend, "    "+tc.field+": "+value+"\n", false)
				_, err := Load(path)
				if err == nil || !strings.Contains(err.Error(), "distributions."+strings.ReplaceAll(backend, "uv", "pypi")+"."+tc.field) {
					t.Fatalf("Load() error = %v, want %s field validation error", err, tc.field)
				}
			})
		}
	}
}

func TestGemKeywordMetadataByteLimit(t *testing.T) {
	for _, count := range []int{512, 513} {
		t.Run(strconv.Itoa(count*2)+" bytes", func(t *testing.T) {
			path := writeMetadataConfig(t, "gem", "    keywords: ["+strconv.Quote(strings.Repeat("é", count))+"]\n", false)
			_, err := Load(path)
			if count == 512 && err != nil {
				t.Fatalf("1024-byte metadata rejected: %v", err)
			}
			if count == 513 && (err == nil || !strings.Contains(err.Error(), "distributions.gem.keywords")) {
				t.Fatalf("Load() error = %v, want metadata byte-limit error", err)
			}
		})
	}
}

func writeMetadataConfig(t *testing.T, backend, metadata string, profile bool) string {
	t.Helper()
	data := "tool:\n  name: mytool\n  main: ./cmd/mytool\ntargets:\n  - os: linux\n    arch: amd64\ndistributions:\n  " + backend + ":\n    package: mytool\n" + metadata
	if profile {
		data = "profiles:\n  default:\n    " + strings.ReplaceAll(strings.TrimSuffix(data, "\n"), "\n", "\n    ") + "\n"
	}
	path := filepath.Join(t.TempDir(), "omnidist.yaml")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
