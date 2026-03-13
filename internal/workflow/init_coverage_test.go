package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
)

func TestCreateNPMStructureWithVariant(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := &config.Config{
		Distributions: map[string]config.DistributionConfig{
			"npm": {Package: "pkg"},
		},
		Targets: []config.Target{
			{OS: "linux", Arch: "amd64", Variant: "musl"},
		},
	}

	if err := CreateNPMStructure(cfg); err != nil {
		t.Fatalf("CreateNPMStructure() error = %v", err)
	}

	pkgDir := filepath.Join(paths.NPMDir, "pkg-linux-x64-musl", "bin")
	if _, err := os.Stat(pkgDir); err != nil {
		t.Fatalf("expected %s to exist, stat error: %v", pkgDir, err)
	}
}

func TestCreateNPMStructureErrors(t *testing.T) {
	t.Run("mkdir_base_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.MkdirAll(paths.WorkspaceDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.WorkspaceDir, err)
		}
		// Create a file where paths.NPMDir should be
		if err := os.WriteFile(paths.NPMDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", paths.NPMDir, err)
		}
		cfg := &config.Config{
			Distributions: map[string]config.DistributionConfig{
				"npm": {Package: "pkg"},
			},
		}
		err := CreateNPMStructure(cfg)
		if err == nil || !strings.Contains(err.Error(), "create npm base directory") {
			t.Fatalf("CreateNPMStructure(mkdir base fail) error = %v, want mkdir error", err)
		}
	})

	t.Run("mkdir_meta_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.MkdirAll(paths.NPMDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.NPMDir, err)
		}
		// Create a file where meta directory should be
		metaDir := filepath.Join(paths.NPMDir, "pkg")
		if err := os.WriteFile(metaDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", metaDir, err)
		}
		cfg := &config.Config{
			Distributions: map[string]config.DistributionConfig{
				"npm": {Package: "pkg"},
			},
		}
		err := CreateNPMStructure(cfg)
		if err == nil || !strings.Contains(err.Error(), "create npm meta directory") {
			t.Fatalf("CreateNPMStructure(mkdir meta fail) error = %v, want mkdir error", err)
		}
	})

	t.Run("mkdir_platform_bin_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.MkdirAll(paths.NPMDir, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.NPMDir, err)
		}
		// Create a file where platform directory should be
		pkgDir := filepath.Join(paths.NPMDir, "pkg-linux-x64")
		if err := os.WriteFile(pkgDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", pkgDir, err)
		}
		cfg := &config.Config{
			Distributions: map[string]config.DistributionConfig{
				"npm": {Package: "pkg"},
			},
			Targets: []config.Target{
				{OS: "linux", Arch: "amd64"},
			},
		}
		err := CreateNPMStructure(cfg)
		if err == nil || !strings.Contains(err.Error(), "create npm platform bin directory") {
			t.Fatalf("CreateNPMStructure(mkdir platform fail) error = %v, want mkdir error", err)
		}
	})
}

func TestCreateUVStructureErrors(t *testing.T) {
	t.Run("mkdir_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		// Create a file where paths.UVDistDir should be
		if err := os.WriteFile(paths.WorkspaceDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", paths.WorkspaceDir, err)
		}
		cfg := &config.Config{
			Distributions: map[string]config.DistributionConfig{
				"uv": {Package: "pkg"},
			},
		}
		err := CreateUVStructure(cfg)
		if err == nil || !strings.Contains(err.Error(), "create uv dist directory") {
			t.Fatalf("CreateUVStructure(mkdir fail) error = %v, want mkdir error", err)
		}
	})
}

func TestEnsureWorkspaceGitignoreErrors(t *testing.T) {
	t.Run("read_fail", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		// Create a directory where .gitignore should be a file
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", path, err)
		}
		err := EnsureWorkspaceGitignore(path)
		if err == nil || !strings.Contains(err.Error(), "is a directory") {
			t.Fatalf("EnsureWorkspaceGitignore(read fail) error = %v, want directory error", err)
		}
	})

	t.Run("mkdir_dir_fail", func(t *testing.T) {
		dir := t.TempDir()
		// Create a file where the directory should be
		path := filepath.Join(dir, "parent")
		if err := os.WriteFile(path, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
		gitignorePath := filepath.Join(path, ".gitignore")

		err := EnsureWorkspaceGitignore(gitignorePath)
		if err == nil || !strings.Contains(err.Error(), "stat gitignore") {
			t.Fatalf("EnsureWorkspaceGitignore(read fail) error = %v, want stat error", err)
		}
	})

	t.Run("write_fail", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		// Create a directory where .gitignore should be a file
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", path, err)
		}
		// Since we already have a dir, os.ReadFile might fail (covered above)
		// or if we make it unreadable?

		// Actually, let's trigger it by making the directory unwriteable.
		parent := filepath.Join(dir, "readonly")
		if err := os.MkdirAll(parent, 0555); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", parent, err)
		}
		defer os.Chmod(parent, 0755)
		gitignorePath := filepath.Join(parent, ".gitignore")

		err := EnsureWorkspaceGitignore(gitignorePath)
		if err == nil || !strings.Contains(err.Error(), "write gitignore") {
			t.Fatalf("EnsureWorkspaceGitignore(write fail) error = %v, want write error", err)
		}
	})
}

func TestInitErrors(t *testing.T) {
	t.Run("save_config_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		// Create a directory where config file should be
		if err := os.MkdirAll(paths.ConfigPath, 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", paths.ConfigPath, err)
		}
		err := Init(paths.ConfigPath)
		if err == nil || !strings.Contains(err.Error(), "save config") {
			t.Fatalf("Init(save fail) error = %v, want save config error", err)
		}
	})

	t.Run("mkdir_workspace_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		// Create a file where paths.WorkspaceDir should be
		if err := os.WriteFile(paths.WorkspaceDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", paths.WorkspaceDir, err)
		}
		// Init might fail at Save() because it also tries to create the directory
		err := Init(paths.ConfigPath)
		if err == nil {
			t.Fatalf("Init(mkdir fail) error = nil, want error")
		}
	})

	t.Run("npm_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.MkdirAll(filepath.Join(paths.WorkspaceDir, "default"), 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Join(paths.WorkspaceDir, "default"), err)
		}
		npmBase := filepath.Join(paths.WorkspaceDir, "default", "npm")
		if err := os.WriteFile(npmBase, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", npmBase, err)
		}
		err := Init(paths.ConfigPath)
		if err == nil {
			t.Fatalf("Init(npm fail) error = nil, want error")
		}
	})

	t.Run("uv_fail", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.MkdirAll(filepath.Join(paths.WorkspaceDir, "default"), 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Join(paths.WorkspaceDir, "default"), err)
		}
		// Create uv directory as a file to trigger error in CreateUVStructure
		// CreateUVStructure uses profile workspace path .omnidist/default/uv/dist.
		uvDir := filepath.Join(paths.WorkspaceDir, "default", "uv")
		if err := os.WriteFile(uvDir, []byte("file"), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", uvDir, err)
		}
		err := Init(paths.ConfigPath)
		if err == nil {
			t.Fatalf("Init(uv fail) error = nil, want error")
		}
	})
}
