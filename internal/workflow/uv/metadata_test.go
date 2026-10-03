package uv

import (
	"bytes"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestStageConfiguredMetadata(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv(shared.EnvVersionName, "1.2.3")
	cfg := testConfig()
	cfg.Description = `Build Go's "CLI" — binaries`
	cfg.Keywords = []string{"golang", "cli"}
	cfg.License = "MIT"
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatal(err)
	}
	const readme = "# Product docs\nKeep this long description.\n"
	if err := os.WriteFile("README.md", []byte(readme), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, target := range cfg.Targets {
		wheelPath, err := wheelPathForTarget(*cfg.Distributions.UV, target, "1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		data := wheelFileData(t, wheelPath, metadataFilePath(t, cfg, "1.2.3"))
		message, err := mail.ReadMessage(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if got := message.Header.Get("Summary"); got != cfg.Description {
			t.Errorf("Summary = %q, want %q", got, cfg.Description)
		}
		if got := message.Header.Get("Keywords"); got != "golang,cli" {
			t.Errorf("Keywords = %q, want golang,cli", got)
		}
		if message.Header.Get("License-Expression") != "MIT" || message.Header.Get("Metadata-Version") != "2.4" || len(message.Header["License"]) != 0 {
			t.Fatalf("license headers = %#v", message.Header)
		}
		if !bytes.HasSuffix(data, []byte("\n\n"+readme)) {
			t.Errorf("METADATA lost README body: %s", data)
		}
	}
	if result := Verify(cfg); !result.Valid {
		t.Fatalf("Verify() errors = %v", result.Errors)
	}
	cfg.License = "Apache-2.0"
	if result := Verify(cfg); result.Valid || !strings.Contains(strings.Join(result.Errors, "\n"), "License-Expression mismatch") {
		t.Fatalf("Verify stale inherited license = %#v", result)
	}
}

func TestVerifyConfiguredMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, stagedDescription, expectedDescription string
		stagedKeywords, expectedKeywords             []string
		field                                        string
	}{
		{"description changed", "Old description", "Current description", nil, nil, "Summary"},
		{"description only in README", "", "Current description", nil, nil, "Summary"},
		{"keywords changed", "", "", []string{"old"}, []string{"cli"}, "Keywords"},
		{"keywords only in README", "", "", nil, []string{"cli"}, "Keywords"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv(shared.EnvVersionName, "1.2.3")
			cfg := testConfig()
			cfg.Distributions.UV.Description = tc.stagedDescription
			cfg.Distributions.UV.Keywords = tc.stagedKeywords
			if err := createDistArtifacts(cfg); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("README.md", []byte("Summary: Current description\nKeywords: cli\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := Stage(cfg, StageOptions{}); err != nil {
				t.Fatal(err)
			}
			cfg.Distributions.UV.Description = tc.expectedDescription
			cfg.Distributions.UV.Keywords = tc.expectedKeywords
			result := Verify(cfg)
			if result.Valid || !strings.Contains(strings.Join(result.Errors, "\n"), tc.field+" mismatch") {
				t.Fatalf("Verify() = %#v, want %s mismatch", result, tc.field)
			}
		})
	}
}

func TestStageMetadataFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv(shared.EnvVersionName, "1.2.3")
	cfg := testConfig()
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatal(err)
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatal(err)
	}
	wheelPath, err := wheelPathForTarget(*cfg.Distributions.UV, cfg.Targets[0], "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(wheelFileData(t, wheelPath, metadataFilePath(t, cfg, "1.2.3"))))
	if err != nil {
		t.Fatal(err)
	}
	if message.Header.Get("Summary") != "Binary distribution for omnidist" || len(message.Header["Keywords"]) != 0 {
		t.Fatalf("fallback headers = %#v", message.Header)
	}
	if message.Header.Get("Metadata-Version") != "2.1" || len(message.Header["License-Expression"]) != 0 {
		t.Fatalf("unlicensed headers = %#v", message.Header)
	}
}

func TestVerifyUVClearedMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "description: Project\nkeywords: [project]\nlicense: MIT\ntool:\n  name: omnidist\ndistributions:\n  uv:\n    package: omnidist\n    description: ''\n    keywords: []\n    license: ''\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	dist, err := cfg.RequireUV()
	if err != nil {
		t.Fatal(err)
	}
	valid := "Metadata-Version: 2.1\nSummary: Binary distribution for omnidist\n\nREADME\n"
	generated := buildWheelMetadata(dist, "1.2.3", "omnidist", nil)
	message, err := mail.ReadMessage(bytes.NewReader(generated))
	if err != nil {
		t.Fatal(err)
	}
	if message.Header.Get("Summary") != "Binary distribution for omnidist" || message.Header.Get("Metadata-Version") != "2.1" || len(message.Header["Keywords"]) != 0 || len(message.Header["License-Expression"]) != 0 {
		t.Fatalf("cleared headers = %#v", message.Header)
	}
	for _, tc := range []struct{ data, want string }{
		{valid, ""},
		{strings.Replace(valid, "\n\n", "\nKeywords: project\n\n", 1), "Keywords mismatch"},
		{strings.Replace(valid, "\n\n", "\nKeywords: \n\n", 1), "Keywords mismatch"},
		{strings.Replace(valid, "\n\n", "\nLicense-Expression: MIT\n\n", 1), "License-Expression mismatch"},
		{strings.Replace(valid, "\n\n", "\nLicense: MIT\n\n", 1), "License mismatch"},
		{strings.Replace(valid, "Binary distribution for omnidist", "Project", 1), "Summary mismatch"},
	} {
		err := verifyConfiguredMetadata([]byte(tc.data), dist, "omnidist")
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("verify cleared metadata error = %v, want %q", err, tc.want)
		}
	}
}

func TestStageUVLicenseOverride(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv(shared.EnvVersionName, "1.2.3")
	cfg := testConfig()
	cfg.License = "MIT"
	cfg.Distributions.UV.License = "Apache-2.0 OR LicenseRef-Proprietary"
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatal(err)
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatal(err)
	}
	wheelPath, err := wheelPathForTarget(*cfg.Distributions.UV, cfg.Targets[0], "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(wheelFileData(t, wheelPath, metadataFilePath(t, cfg, "1.2.3"))))
	if err != nil {
		t.Fatal(err)
	}
	if message.Header.Get("License-Expression") != cfg.Distributions.UV.License {
		t.Fatalf("override license headers = %#v", message.Header)
	}
	if result := Verify(cfg); !result.Valid {
		t.Fatalf("Verify license override = %#v", result)
	}
}
