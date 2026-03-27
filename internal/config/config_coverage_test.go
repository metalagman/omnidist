package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDistributionConfigIncludeREADMEEnabled(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		d := DistributionConfig{IncludeREADME: nil}
		if !d.IncludeREADMEEnabled() {
			t.Fatalf("IncludeREADMEEnabled(nil) = false, want true")
		}
	})

	t.Run("true", func(t *testing.T) {
		val := true
		d := DistributionConfig{IncludeREADME: &val}
		if !d.IncludeREADMEEnabled() {
			t.Fatalf("IncludeREADMEEnabled(true) = false, want true")
		}
	})

	t.Run("false", func(t *testing.T) {
		val := false
		d := DistributionConfig{IncludeREADME: &val}
		if d.IncludeREADMEEnabled() {
			t.Fatalf("IncludeREADMEEnabled(false) = true, want false")
		}
	})
}

func TestSaveErrors(t *testing.T) {
	t.Run("nil_config", func(t *testing.T) {
		err := Save(nil, "path")
		if err == nil || !strings.Contains(err.Error(), "config is nil") {
			t.Fatalf("Save(nil) error = %v, want config nil error", err)
		}
	})

	t.Run("write_fail", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "readonly")
		if err := os.MkdirAll(path, 0555); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		defer os.Chmod(path, 0755)

		configPath := filepath.Join(path, "omnidist.yaml")
		err := Save(DefaultConfig(), configPath)
		if err == nil || !strings.Contains(err.Error(), "write config file") {
			t.Fatalf("Save(readonly) error = %v, want write error", err)
		}
	})
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(path, []byte("invalid: : yaml"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "parse config file") {
		t.Fatalf("Load(invalid) error = %v, want parse error", err)
	}
}

func TestApplyDistributionDefaultsNilDistributions(t *testing.T) {
	cfg := &Config{Distributions: nil}
	applyDistributionDefaults(cfg)
	if cfg.Distributions == nil {
		t.Fatalf("applyDistributionDefaults() failed to initialize Distributions map")
	}
	if _, ok := cfg.Distributions["npm"]; !ok {
		t.Fatalf("applyDistributionDefaults() missing npm default")
	}
	if _, ok := cfg.Distributions["uv"]; !ok {
		t.Fatalf("applyDistributionDefaults() missing uv default")
	}
}

func TestNormalizeKeywords(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "nil input",
			in:   nil,
			want: nil,
		},
		{
			name: "drops empty and duplicates",
			in:   []string{" ai ", "", "llm", "ai", "  ", "cli"},
			want: []string{"ai", "llm", "cli"},
		},
		{
			name: "all empty",
			in:   []string{"", "  "},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeKeywords(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("normalizeKeywords(%#v) = %#v, want %#v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("normalizeKeywords(%#v) = %#v, want %#v", tt.in, got, tt.want)
				}
			}
		})
	}
}
