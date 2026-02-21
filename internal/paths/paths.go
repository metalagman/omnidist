package paths

const (
	ConfigPath = ".omnidist/omnidist.yaml"

	WorkspaceDir    = ".omnidist"
	DistDir         = WorkspaceDir + "/dist"
	DistVersionPath = DistDir + "/VERSION"
	NPMDir          = WorkspaceDir + "/npm"
	NPMRCPath       = WorkspaceDir + "/.npmrc"
	UVDir           = WorkspaceDir + "/uv"
	UVDistDir       = UVDir + "/dist"
)
