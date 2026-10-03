package uv

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestPublishEnvironmentPrecedence(t *testing.T) {
	for _, tc := range []struct{ name, option, canonical, legacy, want string }{
		{"option", " flag ", "canonical", "legacy", "flag"},
		{"canonical", "", " canonical ", "legacy", "canonical"},
		{"legacy", "", "", " legacy ", "legacy"},
		{"blank canonical", " ", " ", "legacy", "legacy"},
		{"missing", "", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PYPI_PUBLISH_TOKEN", tc.canonical)
			t.Setenv("UV_PUBLISH_TOKEN", tc.legacy)
			t.Setenv("PYPI_PUBLISH_URL", tc.canonical)
			t.Setenv("UV_PUBLISH_URL", tc.legacy)
			got, err := resolvePublishToken(PublishOptions{Token: tc.option, DryRun: true})
			if err != nil || got != tc.want {
				t.Fatalf("token = %q, %v; want %q", got, err, tc.want)
			}
			_, err = resolvePublishToken(PublishOptions{Token: tc.option})
			if (err != nil) != (tc.want == "") {
				t.Fatalf("credential requirement: %v", err)
			}
			wantURL := tc.want
			if wantURL == "" {
				wantURL = "configured"
			}
			args := buildPublishArgs("configured", PublishOptions{PublishURL: tc.option}, []string{"artifact.whl"})
			if !reflect.DeepEqual(args, []string{"publish", "--publish-url", wantURL, "artifact.whl"}) {
				t.Fatalf("args = %v", args)
			}
		})
	}
}

func TestPublishCanonicalEnvironmentForwardingAndDestinationPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell publisher shim")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(shared.EnvVersionName, "1.2.3+local")
	cfg := testConfig()
	cfg.Distributions.UV.IndexURL = "https://private.example/legacy/"
	if err := createDistArtifacts(cfg); err != nil {
		t.Fatal(err)
	}
	if err := shared.WriteBuildVersion("1.2.3+local"); err != nil {
		t.Fatal(err)
	}
	if err := Stage(cfg, StageOptions{}); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "publish.log")
	t.Setenv("OMNIDIST_TEST_LOG", logPath)
	tool := filepath.Join(dir, "uv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$UV_PUBLISH_TOKEN\" \"$@\" > \"$OMNIDIST_TEST_LOG\"\n"
	if err := os.WriteFile(tool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PYPI_PUBLISH_TOKEN", "canonical-token")
	t.Setenv("UV_PUBLISH_TOKEN", "legacy-token")
	t.Setenv("PYPI_PUBLISH_URL", "https://upload.pypi.org/legacy/")
	t.Setenv("UV_PUBLISH_URL", "https://legacy.example/legacy/")
	for _, run := range []func() error{
		func() error { return PreflightPublish(cfg, PublishOptions{}) },
		func() error { return Publish(cfg, PublishOptions{}) },
	} {
		if err := run(); err == nil || !strings.Contains(err.Error(), "local version") {
			t.Fatalf("public destination policy: %v", err)
		}
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("publication attempted before policy check: %v", err)
	}
	t.Setenv("PYPI_PUBLISH_URL", "https://canonical.example/legacy/")
	var stdout, stderr bytes.Buffer
	opts := PublishOptions{Stdout: &stdout, Stderr: &stderr}
	if err := PreflightPublish(cfg, opts); err != nil {
		t.Fatal(err)
	}
	if err := Publish(cfg, opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	if lines[0] != "canonical-token" || !strings.Contains(string(data), "--publish-url\nhttps://canonical.example/legacy/") {
		t.Fatalf("wrong publisher options: %s", data)
	}
	if strings.Contains(strings.Join(lines[1:], "\n"), "token") || strings.Contains(stdout.String()+stderr.String(), "token") {
		t.Fatal("credential exposed in argv/output")
	}
	if cfg.Distributions.UV.IndexURL != "https://private.example/legacy/" {
		t.Fatal("environment resolution mutated source")
	}
	// A private override permits local versions even when config defaults to PyPI.
	cfg.Distributions.UV.IndexURL = "https://upload.pypi.org/legacy/"
	if err := PreflightPublish(cfg, opts); err != nil {
		t.Fatal(err)
	}
	if err := Publish(cfg, opts); err != nil {
		t.Fatal(err)
	}
}
