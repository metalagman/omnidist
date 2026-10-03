package config

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestPythonConfigAliases(t *testing.T) {
	for _, profile := range []bool{false, true} {
		for _, key := range []string{"uv", "pypi"} {
			for _, selector := range []string{"", "uv", "pypi", " UV ", " PyPI "} {
				t.Run(key+"/"+selector+"/"+map[bool]string{false: "legacy", true: "profile"}[profile], func(t *testing.T) {
					data := "distributions:\n  " + key + ":\n    package: mytool\n"
					if selector != "" {
						data = "enabled-distributions: [" + strconv.Quote(selector) + "]\n" + data
					}
					path := writePythonConfig(t, data, profile)
					cfg, err := Load(path)
					if err != nil {
						t.Fatal(err)
					}
					got, err := cfg.SelectedDistributionNames()
					if err != nil || !slices.Equal(DistributionNameStrings(got), []string{"pypi"}) {
						t.Fatalf("selection = %v, %v", got, err)
					}
					if profile {
						err = WriteProfiles(path, "default", cfg, true)
					} else {
						err = Save(cfg, path)
					}
					if err != nil {
						t.Fatal(err)
					}
					written, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(written), "pypi:") || strings.Contains(string(written), "uv:") {
						t.Fatalf("noncanonical source: %s", written)
					}
					if _, err := Load(path); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestPythonConfigAliasBoundaries(t *testing.T) {
	for _, profile := range []bool{false, true} {
		for _, tc := range []struct{ name, data, wantErr string }{
			{"dual sections", "distributions:\n  uv: {package: mytool}\n  pypi: {package: mytool}\n", "both"},
			{"canonical null", "distributions:\n  uv: {package: mytool}\n  pypi: null\n", ""},
			{"legacy null", "distributions:\n  pypi: {package: mytool}\n  uv: null\n", ""},
			{"both null", "distributions: {uv: null, pypi: null}\n", "must configure"},
			{"duplicate aliases", "enabled-distributions: [uv, pypi]\ndistributions:\n  uv: {package: mytool}\n", "duplicate"},
			{"absent", "enabled-distributions: [pypi]\ndistributions:\n  npm: {package: mytool}\n", "required"},
			{"canonical unknown field", "distributions:\n  pypi: {package: mytool, unknown: true}\n", "field unknown"},
			{"legacy unknown field", "distributions:\n  uv: {package: mytool, unknown: true}\n", "field unknown"},
		} {
			t.Run(tc.name+"/"+map[bool]string{false: "legacy", true: "profile"}[profile], func(t *testing.T) {
				_, err := Load(writePythonConfig(t, tc.data, profile))
				if tc.wantErr == "" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
			})
		}
	}
}

func writePythonConfig(t *testing.T, data string, profile bool) string {
	t.Helper()
	if profile {
		data = "profiles:\n  default:\n    " + strings.ReplaceAll(strings.TrimSuffix(data, "\n"), "\n", "\n    ") + "\n"
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPythonAliasesAcrossProfilesAndInactiveCanonicalConfig(t *testing.T) {
	path := writePythonConfig(t, "profiles:\n  old:\n    description: Old\n    distributions:\n      uv: {package: old}\n  new:\n    description: New\n    distributions:\n      pypi: {package: new}\n", false)
	for _, profile := range []string{"old", "new"} {
		cfg, err := LoadWithProfile(path, profile)
		if err != nil {
			t.Fatal(err)
		}
		dist, err := cfg.RequireUV()
		if err != nil {
			t.Fatal(err)
		}
		if dist.Package != profile || strings.ToLower(dist.Description) != profile {
			t.Fatalf("profile metadata leaked: %+v", dist)
		}
	}
	cfg, err := Load(writePythonConfig(t, "enabled-distributions: [npm]\ndistributions:\n  npm: {package: mytool}\n  pypi: {package: mytool, linux-tag: invalid}\n", false))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := cfg.SelectedDistributionNames()
	if err != nil || !slices.Equal(DistributionNameStrings(selected), []string{"npm"}) {
		t.Fatalf("inactive backend selected: %v, %v", selected, err)
	}
	if _, err := cfg.RequireUV(); err == nil {
		t.Fatal("direct backend ignored invalid canonical settings")
	}
}
