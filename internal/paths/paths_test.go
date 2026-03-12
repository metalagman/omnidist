package paths

import (
	"testing"
)

func TestPaths(t *testing.T) {
	// Just touch the constants to ensure they are defined as expected and provide 100% coverage
	// Since these are just strings, we don't need complex logic.
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ConfigPath", ConfigPath, ".omnidist/omnidist.yaml"},
		{"WorkspaceDir", WorkspaceDir, ".omnidist"},
		{"DistDir", DistDir, ".omnidist/dist"},
		{"DistVersionPath", DistVersionPath, ".omnidist/dist/VERSION"},
		{"NPMDir", NPMDir, ".omnidist/npm"},
		{"NPMRCPath", NPMRCPath, ".omnidist/.npmrc"},
		{"UVDir", UVDir, ".omnidist/uv"},
		{"UVPyprojectPath", UVPyprojectPath, ".omnidist/uv/pyproject.toml"},
		{"UVDistDir", UVDistDir, ".omnidist/uv/dist"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestNewLayout(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		want      Layout
	}{
		{
			name:      "empty_workspace_uses_default",
			workspace: "",
			want: Layout{
				WorkspaceDir:    ".omnidist",
				DistDir:         ".omnidist/dist",
				DistVersionPath: ".omnidist/dist/VERSION",
				NPMDir:          ".omnidist/npm",
				NPMRCPath:       ".omnidist/.npmrc",
				UVDir:           ".omnidist/uv",
				UVPyprojectPath: ".omnidist/uv/pyproject.toml",
				UVDistDir:       ".omnidist/uv/dist",
			},
		},
		{
			name:      "custom_workspace",
			workspace: ".omnidist/prod",
			want: Layout{
				WorkspaceDir:    ".omnidist/prod",
				DistDir:         ".omnidist/prod/dist",
				DistVersionPath: ".omnidist/prod/dist/VERSION",
				NPMDir:          ".omnidist/prod/npm",
				NPMRCPath:       ".omnidist/prod/.npmrc",
				UVDir:           ".omnidist/prod/uv",
				UVPyprojectPath: ".omnidist/prod/uv/pyproject.toml",
				UVDistDir:       ".omnidist/prod/uv/dist",
			},
		},
	}

	for _, tt := range tests {
		got := NewLayout(tt.workspace)
		if got != tt.want {
			t.Fatalf("%s: NewLayout(%q) = %#v, want %#v", tt.name, tt.workspace, got, tt.want)
		}
	}
}
