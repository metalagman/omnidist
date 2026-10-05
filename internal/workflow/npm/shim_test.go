package npm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestNPMShimSelectsWindowsPackage(t *testing.T) {
	t.Parallel()

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required to execute the npm launcher")
	}
	for _, tc := range []struct {
		goArch   string
		nodeArch string
		pkg      string
	}{
		{goArch: "amd64", nodeArch: "x64", pkg: "@scope/tool-bin-win32-x64"},
		{goArch: "arm64", nodeArch: "arm64", pkg: "@scope/tool-bin-win32-arm64"},
	} {
		t.Run(tc.goArch, func(t *testing.T) {
			t.Parallel()
			shim, binary := stageWindowsShimFixture(t, tc.goArch, []byte("fixture"))
			preload := filepath.Join(t.TempDir(), "preload.js")
			// Exercise actual package resolution; only the foreign platform and
			// execution of a Windows binary on the host OS need substitutes.
			code := `const os = require('os');
os.platform = () => 'win32';
os.arch = () => process.env.OMNIDIST_TEST_NODE_ARCH;
require('child_process').execFileSync = (file, args, options) => {
  process.stdout.write(JSON.stringify({file, args, stdio: options.stdio}));
};`
			if err := os.WriteFile(preload, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(node, "--require", preload, shim, "version", "argument with spaces")
			cmd.Env = append(os.Environ(), "OMNIDIST_TEST_NODE_ARCH="+tc.nodeArch)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("launcher failed: %v\n%s", err, output)
			}
			var result struct {
				File  string   `json:"file"`
				Args  []string `json:"args"`
				Stdio string   `json:"stdio"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("launcher output %q: %v", output, err)
			}
			if result.File != binary || !strings.Contains(result.File, filepath.FromSlash(tc.pkg)) {
				t.Fatalf("selected %q, want %q from %s", result.File, binary, tc.pkg)
			}
			if len(result.Args) != 2 || result.Args[0] != "version" || result.Args[1] != "argument with spaces" || result.Stdio != "inherit" {
				t.Fatalf("launcher did not forward arguments/stdio: %+v", result)
			}
		})
	}
}

func TestNPMShimWindowsARM64Native(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "arm64" {
		t.Skip("requires native Windows ARM64; exercised by the Windows ARM64 CI job")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("Node.js is required for the native launcher check")
	}
	identity, err := exec.Command(node, "-p", "process.platform + '/' + process.arch").CombinedOutput()
	if err != nil || strings.TrimSpace(string(identity)) != "win32/arm64" {
		t.Fatalf("requires native ARM64 Node.js, got %q: %v", identity, err)
	}
	buildPath := filepath.Join(t.TempDir(), "omnidist.exe")
	build := exec.Command("go", "build", "-o", buildPath, "github.com/metalagman/omnidist/cmd/omnidist")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build native Omnidist: %v\n%s", err, output)
	}
	binary, err := os.ReadFile(buildPath)
	if err != nil {
		t.Fatal(err)
	}
	shim, installedBinary := stageWindowsShimFixture(t, "arm64", binary)
	for _, args := range [][]string{{"version"}, {"--invalid-launcher-test-flag"}} {
		wantOutput, wantErr := exec.Command(installedBinary, args...).CombinedOutput()
		if len(wantOutput) == 0 || commandExitCode(wantErr) < 0 {
			t.Fatalf("native binary did not execute: %v, output %q", wantErr, wantOutput)
		}
		if args[0] == "version" && wantErr != nil {
			t.Fatalf("native version command failed: %v\n%s", wantErr, wantOutput)
		}
		if args[0] != "version" && commandExitCode(wantErr) == 0 {
			t.Fatal("native binary accepted an invalid flag")
		}
		output, err := exec.Command(node, append([]string{shim}, args...)...).CombinedOutput()
		if string(output) != string(wantOutput) || commandExitCode(err) != commandExitCode(wantErr) {
			t.Fatalf("args %v: launcher output %q, error %v; binary output %q, error %v", args, output, err, wantOutput, wantErr)
		}
	}
}

func stageWindowsShimFixture(t *testing.T, arch string, binary []byte) (string, string) {
	t.Helper()
	root := t.TempDir()
	layout := paths.NewLayout(root)
	layout.NPMDir = filepath.Join(root, "node_modules")
	cfg := &config.Config{
		Tool:    config.ToolConfig{Name: "omnidist"},
		Targets: []config.Target{{OS: "windows", Arch: arch}},
	}
	dist := config.NPMDistributionConfig{Package: "@scope/tool", PlatformPackage: "@scope/tool-bin"}
	input := filepath.Join(layout.DistDir, "windows", arch, "omnidist.exe")
	if err := os.MkdirAll(filepath.Dir(input), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, binary, 0755); err != nil {
		t.Fatal(err)
	}
	if err := stagePlatformPackage(layout, cfg, dist, cfg.Targets[0], "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := stageMetaPackage(layout, cfg, dist, dist.Package, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	metaDir := filepath.Join(layout.NPMDir, dist.Package)
	metadata, err := readPackageJSON(metaDir)
	if err != nil {
		t.Fatal(err)
	}
	pkg := "@scope/tool-bin-win32-arm64"
	if arch == "amd64" {
		pkg = "@scope/tool-bin-win32-x64"
	}
	deps, ok := metadata["optionalDependencies"].(map[string]interface{})
	if !ok || deps[pkg] != "0.2.0" {
		t.Fatalf("staged optional dependencies omit %s: %#v", pkg, metadata["optionalDependencies"])
	}
	return filepath.Join(metaDir, "omnidist.js"), filepath.Join(layout.NPMDir, pkg, "bin", "omnidist.exe")
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}
