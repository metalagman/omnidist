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
