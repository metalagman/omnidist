package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadStrictSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "unknown top-level field",
			yaml:    "mystery: true\ndistributions:\n  npm:\n    package: '@scope/tool'\n",
			wantErr: "field mystery not found",
		},
		{
			name:    "unknown backend",
			yaml:    "distributions:\n  brew:\n    package: tool\n",
			wantErr: "field brew not found",
		},
		{
			name:    "misspelled npm field",
			yaml:    "distributions:\n  npm:\n    pakage: '@scope/tool'\n",
			wantErr: "field pakage not found",
		},
		{
			name:    "uv field under npm",
			yaml:    "distributions:\n  npm:\n    package: '@scope/tool'\n    index-url: https://example.invalid\n",
			wantErr: "field index-url not found",
		},
		{
			name:    "unknown field in profile",
			yaml:    "profiles:\n  default:\n    distributions:\n      uv:\n        package: tool\n        registry: https://example.invalid\n",
			wantErr: "field registry not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := writeResolverConfig(t, tc.yaml)
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Load() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestLoadSelectorPresenceMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		want    []string
		wantErr string
	}{
		{
			name: "absent derives sections",
			yaml: "distributions:\n  gem:\n    package: tool\n  npm:\n    package: '@scope/tool'\n",
			want: []string{"npm", "gem"},
		},
		{
			name:    "absent without sections",
			yaml:    "tool:\n  name: tool\n",
			wantErr: "distributions must configure at least one",
		},
		{
			name:    "null legacy selector",
			yaml:    "enabled-distributions: null\ndistributions:\n  npm:\n    package: '@scope/tool'\n",
			wantErr: "enabled-distributions must not be null",
		},
		{
			name:    "null profile selector",
			yaml:    "profiles:\n  default:\n    enabled-distributions: null\n    distributions:\n      npm:\n        package: '@scope/tool'\n",
			wantErr: "enabled-distributions must not be null",
		},
		{
			name:    "empty legacy selector",
			yaml:    "enabled-distributions: []\ndistributions:\n  npm:\n    package: '@scope/tool'\n",
			wantErr: "enabled-distributions must contain at least one",
		},
		{
			name:    "empty profile selector",
			yaml:    "profiles:\n  default:\n    enabled-distributions: []\n    distributions:\n      npm:\n        package: '@scope/tool'\n",
			wantErr: "enabled-distributions must contain at least one",
		},
		{
			name:    "duplicate selector",
			yaml:    "enabled-distributions: [npm, NPM]\ndistributions:\n  npm:\n    package: '@scope/tool'\n",
			wantErr: "duplicate enabled distribution",
		},
		{
			name:    "unknown selector",
			yaml:    "enabled-distributions: [brew]\ndistributions:\n  npm:\n    package: '@scope/tool'\n",
			wantErr: "invalid enabled distribution",
		},
		{
			name:    "selected section is null",
			yaml:    "enabled-distributions: [uv]\ndistributions:\n  uv: null\n",
			wantErr: "distributions.uv is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(writeResolverConfig(t, tc.yaml))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Load() error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			gotNames, err := cfg.SelectedDistributionNames()
			if err != nil {
				t.Fatalf("SelectedDistributionNames() error = %v", err)
			}
			got := DistributionNameStrings(gotNames)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SelectedDistributionNames() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestLoadCoversEveryDistributionSubsetAndWorkspaceMode(t *testing.T) {
	t.Parallel()

	subsets := [][]DistributionName{
		{DistributionNPM},
		{DistributionUV},
		{DistributionGem},
		{DistributionNPM, DistributionUV},
		{DistributionNPM, DistributionGem},
		{DistributionUV, DistributionGem},
		{DistributionNPM, DistributionUV, DistributionGem},
	}
	for _, profilesMode := range []bool{false, true} {
		profilesMode := profilesMode
		for _, legacySelector := range []bool{false, true} {
			legacySelector := legacySelector
			for _, selected := range subsets {
				selected := append([]DistributionName(nil), selected...)
				mode := "legacy"
				if profilesMode {
					mode = "profiles"
				}
				shape := "presence"
				if legacySelector {
					shape = "selector"
				}
				name := mode + "/" + shape + "/" + strings.Join(DistributionNameStrings(selected), "+")
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					distributions := make(map[string]any, len(selected))
					selector := make([]string, 0, len(selected))
					for i := len(selected) - 1; i >= 0; i-- {
						backend := selected[i]
						packageName := "tool"
						if backend == DistributionNPM {
							packageName = "@scope/tool"
						}
						distributions[string(backend)] = map[string]any{"package": packageName}
						selector = append(selector, string(backend))
					}
					body := map[string]any{"distributions": distributions}
					if legacySelector {
						body["enabled-distributions"] = selector
					}
					root := body
					if profilesMode {
						root = map[string]any{"profiles": map[string]any{DefaultProfileName: body}}
					}
					data, err := yaml.Marshal(root)
					if err != nil {
						t.Fatal(err)
					}
					cfg, err := Load(writeResolverConfig(t, string(data)))
					if err != nil {
						t.Fatalf("Load() error = %v\n%s", err, data)
					}
					got, err := cfg.SelectedDistributionNames()
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, selected) {
						t.Fatalf("SelectedDistributionNames() = %#v, want %#v", got, selected)
					}
					if cfg.Runtime.ProfilesMode != profilesMode {
						t.Fatalf("Runtime.ProfilesMode = %v, want %v", cfg.Runtime.ProfilesMode, profilesMode)
					}
				})
			}
		}
	}
}

func TestLoadDefersInactiveBackendSemanticValidation(t *testing.T) {
	t.Parallel()

	path := writeResolverConfig(t, `enabled-distributions: [npm]
distributions:
  npm:
    package: "@scope/tool"
  uv:
    package: tool
    linux-tag: invalid
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Distributions.UV == nil {
		t.Fatal("Load() discarded inactive distributions.uv")
	}
	if _, err := cfg.RequireUV(); err == nil || !strings.Contains(err.Error(), "distributions.uv.linux-tag") {
		t.Fatalf("RequireUV() error = %v, want inactive uv validation error", err)
	}
}

func TestResolvedConfigCannotMarshalAsSourceYAML(t *testing.T) {
	t.Parallel()

	_, err := yaml.Marshal(DefaultConfig())
	if err == nil || !strings.Contains(err.Error(), "cannot be marshaled directly") {
		t.Fatalf("yaml.Marshal(Config) error = %v, want resolved-config error", err)
	}
}

func TestMarshalProfilesWritesCanonicalPresenceBasedConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Tool.Name = "tool"
	cfg.Tool.Main = "./cmd/tool"
	cfg.Distributions.NPM.Package = "@scope/tool"
	cfg.Distributions.UV = nil
	cfg.Distributions.Gem = nil
	cfg.Runtime = RuntimeConfig{
		Profile:      "private-runtime-profile",
		ProfilesMode: true,
		WorkspaceDir: "private-runtime-workspace",
	}

	data, err := MarshalProfiles(DefaultProfileName, cfg)
	if err != nil {
		t.Fatalf("MarshalProfiles() error = %v", err)
	}
	content := string(data)
	for _, unwanted := range []string{"enabled-distributions", "private-runtime-profile", "private-runtime-workspace", "\n      uv:", "\n      gem:"} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("MarshalProfiles() contains %q:\n%s", unwanted, content)
		}
	}
	for _, want := range []string{"profiles:", "default:", "npm:", "package: '@scope/tool'"} {
		if !strings.Contains(content, want) {
			t.Fatalf("MarshalProfiles() missing %q:\n%s", want, content)
		}
	}

	loadedPath := writeResolverConfig(t, content)
	loaded, err := Load(loadedPath)
	if err != nil {
		t.Fatalf("Load(canonical config) error = %v", err)
	}
	gotNames, err := loaded.SelectedDistributionNames()
	if err != nil {
		t.Fatal(err)
	}
	got := DistributionNameStrings(gotNames)
	if !reflect.DeepEqual(got, []string{"npm"}) {
		t.Fatalf("canonical selection = %#v, want npm", got)
	}
}

func TestMarshalProfilesRejectsInactiveLegacySections(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeResolverConfig(t, `enabled-distributions: [npm]
distributions:
  npm:
    package: "@scope/tool"
  uv:
    package: tool
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = MarshalProfiles(DefaultProfileName, cfg)
	if err == nil || !strings.Contains(err.Error(), "inactive legacy distributions") {
		t.Fatalf("MarshalProfiles() error = %v, want inactive legacy error", err)
	}
}

func writeResolverConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "omnidist.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
