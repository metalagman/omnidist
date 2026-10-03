package main

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/paths"
)

func TestPythonCommandsAcrossConfigAliases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell based publisher shim")
	}
	for _, key := range []string{"pypi", "uv"} {
		for _, group := range []string{"pypi", "uv"} {
			t.Run(key+"/"+group, func(t *testing.T) {
				dir := t.TempDir()
				t.Chdir(dir)
				if err := setupCommandFlowProject(); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(paths.ConfigPath)
				if err != nil {
					t.Fatal(err)
				}
				if key == "uv" {
					data = []byte(strings.ReplaceAll(string(data), "pypi:", "uv:"))
				}
				if err := os.WriteFile(paths.ConfigPath, data, 0644); err != nil {
					t.Fatal(err)
				}
				if err := installFakeTool(t, dir, "uv", "#!/bin/sh\nexit 0\n"); err != nil {
					t.Fatal(err)
				}
				other := "pypi"
				if group == "pypi" {
					other = "uv"
				}
				for _, args := range [][]string{
					{group, "stage"},
					{other, "verify"},
					{other, "publish", "--dry-run"},
					{"stage", "--only", "uv,pypi"},
					{group, "verify"},
					{"verify", "--only", "uv,pypi"},
					{group, "publish", "--dry-run"},
					{"publish", "--only", "uv,pypi", "--dry-run"},
				} {
					if _, err := executeCommand(args...); err != nil {
						t.Fatalf("%v: %v", args, err)
					}
				}
				if _, err := os.Stat(paths.UVDistDir); err != nil {
					t.Fatalf("legacy artifact directory: %v", err)
				}
			})
		}
	}
}

func TestPythonLegacyURLFlagPrecedesEnvironmentWhenCanonicalFlagBlank(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell publisher shim")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := setupCommandFlowProject(); err != nil {
		t.Fatal(err)
	}
	logPath := dir + "/publish.log"
	t.Setenv("OMNIDIST_TEST_LOG", logPath)
	t.Setenv("PYPI_PUBLISH_URL", "https://environment.example/legacy/")
	if err := installFakeTool(t, dir, "uv", "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$OMNIDIST_TEST_LOG\"\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand("uv", "stage"); err != nil {
		t.Fatal(err)
	}
	for _, group := range []string{"pypi", "uv"} {
		if _, err := executeCommand(group, "publish", "--dry-run", "--publish-url", "  ", "--repository-url", "https://explicit.example/legacy/"); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "--publish-url\nhttps://explicit.example/legacy/") {
			t.Fatalf("explicit URL lost: %s", data)
		}
	}
}

func TestPythonMixedSelectorsPublishOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell publisher shim")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := setupCommandFlowProject(); err != nil {
		t.Fatal(err)
	}
	logPath := dir + "/publish.log"
	t.Setenv("OMNIDIST_TEST_LOG", logPath)
	if err := installFakeTool(t, dir, "uv", "#!/bin/sh\nif [ \"$1\" = publish ]; then printf 'publish\\n' >> \"$OMNIDIST_TEST_LOG\"; fi\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand("uv", "stage"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand("pypi", "verify"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand("publish", "--only", "uv,pypi, UV , PyPI", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "publish\n" {
		t.Fatalf("alias duplicated publication: %q", data)
	}
}
