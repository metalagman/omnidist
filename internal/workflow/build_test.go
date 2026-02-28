package workflow

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestRenderBuildLDFlags(t *testing.T) {
	t.Setenv("OMNIDIST_VERSION", "1.2.3")
	t.Setenv("OMNIDIST_GIT_COMMIT", "deadbeef")
	t.Setenv("OMNIDIST_BUILD_DATE", "2026-02-22T01:02:03Z")
	got := renderBuildLDFlags("-s -w -X github.com/metalagman/appkit/version.version=${OMNIDIST_VERSION}")
	want := "-s -w -X github.com/metalagman/appkit/version.version=1.2.3"
	if got != want {
		t.Fatalf("renderBuildLDFlags() = %q, want %q", got, want)
	}

	got = renderBuildLDFlags("-X github.com/metalagman/appkit/version.gitCommit=${OMNIDIST_GIT_COMMIT} -X github.com/metalagman/appkit/version.buildDate=${OMNIDIST_BUILD_DATE}")
	want = "-X github.com/metalagman/appkit/version.gitCommit=deadbeef -X github.com/metalagman/appkit/version.buildDate=2026-02-22T01:02:03Z"
	if got != want {
		t.Fatalf("renderBuildLDFlags() = %q, want %q", got, want)
	}
}

func TestRenderBuildLDFlagsTrimsWhitespace(t *testing.T) {
	t.Parallel()

	got := renderBuildLDFlags("  -s -w  ")
	if got != "-s -w" {
		t.Fatalf("renderBuildLDFlags() = %q, want %q", got, "-s -w")
	}
}

func TestBuildTagsFlagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "empty",
			tags: nil,
			want: "",
		},
		{
			name: "single",
			tags: []string{"release"},
			want: "release",
		},
		{
			name: "multiple_trimmed",
			tags: []string{"tag1", " tag2 ", ""},
			want: "tag1,tag2",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := buildTagsFlagValue(tc.tags); got != tc.want {
				t.Fatalf("buildTagsFlagValue() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildWithOptionsRoutesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", binDir, err)
	}

	goPath := filepath.Join(binDir, "go")
	script := `#!/bin/sh
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
    prev=""
    continue
  fi
  prev="$arg"
done
mkdir -p "$(dirname "$out")"
printf "fake-binary" > "$out"
echo "fake go stdout $GOOS/$GOARCH"
echo "fake go stderr" >&2
`
	if err := os.WriteFile(goPath, []byte(script), 0755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", goPath, err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.DefaultConfig()
	cfg.Tool.Name = "omnitest"
	cfg.Tool.Main = "./cmd/does-not-matter"
	cfg.Targets = []config.Target{{OS: "linux", Arch: "amd64"}}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var progress bytes.Buffer
	if err := BuildWithOptions(cfg, BuildOptions{
		Stdout:         &stdout,
		Stderr:         &stderr,
		ProgressWriter: &progress,
	}); err != nil {
		t.Fatalf("BuildWithOptions() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "fake go stdout linux/amd64") {
		t.Fatalf("stdout = %q, want fake go stdout", stdout.String())
	}
	if !strings.Contains(stderr.String(), "fake go stderr") {
		t.Fatalf("stderr = %q, want fake go stderr", stderr.String())
	}
	if !strings.Contains(progress.String(), "Built: .omnidist/dist/linux/amd64/omnitest") {
		t.Fatalf("progress = %q, want built path", progress.String())
	}
}

func TestBuildUsesDefaultOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", binDir, err)
	}

	goPath := filepath.Join(binDir, "go")
	script := `#!/bin/sh
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
    prev=""
    continue
  fi
  prev="$arg"
done
mkdir -p "$(dirname "$out")"
printf "fake-binary" > "$out"
`
	if err := os.WriteFile(goPath, []byte(script), 0755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", goPath, err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.DefaultConfig()
	cfg.Tool.Name = "omnitest"
	cfg.Tool.Main = "./cmd/does-not-matter"
	cfg.Targets = []config.Target{{OS: "linux", Arch: "amd64"}}

	if err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(".omnidist", "dist", "linux", "amd64", "omnitest")); err != nil {
		t.Fatalf("built artifact missing: %v", err)
	}
}

func TestBuildErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Case 1: MkdirAll failure for dist dir
	os.WriteFile(filepath.Join(dir, ".omnidist"), []byte("file"), 0644)
	cfg := config.DefaultConfig()
	if err := Build(cfg); err == nil {
		t.Fatalf("Build() with blocked .omnidist error = nil, want error")
	}
	os.Remove(filepath.Join(dir, ".omnidist"))

	// Case 2: MkdirAll failure for target dir
	cfg.Targets = []config.Target{{OS: "linux", Arch: "amd64"}}
	os.MkdirAll(".omnidist/dist/linux", 0755)
	os.WriteFile(".omnidist/dist/linux/amd64", []byte("file"), 0644)
	if err := Build(cfg); err == nil {
		t.Fatalf("Build() with blocked target dir error = nil, want error")
	}
}

func TestBuildWindowsTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	goPath := filepath.Join(binDir, "go")
	script := `#!/bin/sh
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
  fi
  prev="$arg"
done
mkdir -p "$(dirname "$out")"
printf "fake-binary" > "$out"
`
	os.WriteFile(goPath, []byte(script), 0755)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.DefaultConfig()
	cfg.Tool.Name = "omnitest"
	cfg.Targets = []config.Target{{OS: "windows", Arch: "amd64"}}

	if err := Build(cfg); err != nil {
		t.Fatalf("Build(windows) error = %v", err)
	}

	if _, err := os.Stat(".omnidist/dist/windows/amd64/omnitest.exe"); err != nil {
		t.Fatalf("built artifact missing: %v", err)
	}
}

func TestBuildCGO(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0755)
	goPath := filepath.Join(binDir, "go")
	script := `#!/bin/sh
if [ "$CGO_ENABLED" != "1" ]; then
  echo "CGO_ENABLED not 1" >&2
  exit 1
fi
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
  fi
  prev="$arg"
done
mkdir -p "$(dirname "$out")"
printf "fake-binary" > "$out"
`
	os.WriteFile(goPath, []byte(script), 0755)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.DefaultConfig()
	cfg.Build.CGO = true
	cfg.Targets = []config.Target{{OS: "linux", Arch: "amd64"}}

	if err := Build(cfg); err != nil {
		t.Fatalf("Build(cgo) error = %v", err)
	}
}
