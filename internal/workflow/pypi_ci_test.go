package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestCIResolvesPythonConfigAliases(t *testing.T) {
	for _, profile := range []bool{false, true} {
		var canonical string
		for _, key := range []string{"pypi", "uv"} {
			data := "enabled-distributions: [uv]\ndistributions:\n  " + key + ": {package: mytool}\n"
			if profile {
				data = "profiles:\n  default:\n    " + strings.ReplaceAll(strings.TrimSuffix(data, "\n"), "\n", "\n    ") + "\n"
			}
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(data), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{})
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"publish_pypi:", "pypi publish", "--only 'pypi'", "PYPI_PUBLISH_TOKEN: ${{ secrets.PYPI_PUBLISH_TOKEN }}", "UV_PUBLISH_TOKEN: ${{ secrets.UV_PUBLISH_TOKEN }}", "astral-sh/setup-uv@v6"} {
				if !strings.Contains(got, want) {
					t.Fatalf("%s config generated no %q", key, want)
				}
			}
			if canonical == "" {
				canonical = got
			} else if got != canonical {
				t.Fatal("legacy input changed generated workflow semantics")
			}
		}
	}
}

func TestMaintainedReleaseWorkflowMatchesGenerator(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", ".omnidist", "omnidist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := GenerateGitHubReleaseWorkflow(cfg, CIWorkflowOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "omnidist-release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatal("maintained workflow drifted; regenerate with omnidist ci --force")
	}
}
