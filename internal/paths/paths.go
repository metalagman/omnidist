package paths

import "strings"

const (
	ConfigPath = ".omnidist/omnidist.yaml"

	WorkspaceDir    = ".omnidist"
	DistDir         = WorkspaceDir + "/dist"
	DistVersionPath = DistDir + "/VERSION"
	NPMDir          = WorkspaceDir + "/npm"
	NPMRCPath       = WorkspaceDir + "/.npmrc"
	UVDir           = WorkspaceDir + "/uv"
	UVPyprojectPath = UVDir + "/pyproject.toml"
	UVDistDir       = UVDir + "/dist"
)

// Layout holds resolved artifact paths for a workspace root.
type Layout struct {
	WorkspaceDir    string
	DistDir         string
	DistVersionPath string
	NPMDir          string
	NPMRCPath       string
	UVDir           string
	UVPyprojectPath string
	UVDistDir       string
}

// NewLayout resolves all path variants for a workspace root.
func NewLayout(workspaceDir string) Layout {
	ws := strings.TrimSpace(workspaceDir)
	if ws == "" {
		ws = WorkspaceDir
	}
	return Layout{
		WorkspaceDir:    ws,
		DistDir:         ws + "/dist",
		DistVersionPath: ws + "/dist/VERSION",
		NPMDir:          ws + "/npm",
		NPMRCPath:       ws + "/.npmrc",
		UVDir:           ws + "/uv",
		UVPyprojectPath: ws + "/uv/pyproject.toml",
		UVDistDir:       ws + "/uv/dist",
	}
}
