package workflow

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"gopkg.in/yaml.v3"
)

var getWorkingDir = os.Getwd

// InitOptions controls project identity discovery and config replacement.
type InitOptions struct {
	Force    bool
	ToolName string
	ToolMain string
}

// Init writes a default config and creates initial staging directories.
func Init(configPath string, opts InitOptions) error {
	wd, err := getWorkingDir()
	if err != nil {
		return fmt.Errorf("get current working directory: %w", err)
	}

	if !opts.Force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config %s already exists; rerun with --force to replace it", configPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect config %s: %w", configPath, err)
		}
	}

	cfg, err := defaultInitConfig(wd, opts)
	if err != nil {
		return err
	}

	if err := saveProfilesConfig(cfg, configPath, opts.Force); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	profileCfg, err := config.LoadWithProfile(configPath, config.DefaultProfileName)
	if err != nil {
		return fmt.Errorf("load generated profile config: %w", err)
	}

	if err := CreateNPMStructure(profileCfg); err != nil {
		return err
	}

	if err := CreateUVStructure(profileCfg); err != nil {
		return err
	}

	return nil
}

func defaultInitConfig(wd string, opts InitOptions) (*config.Config, error) {
	name, mainPath, err := resolveProjectIdentity(wd, opts)
	if err != nil {
		return nil, err
	}

	cfg := config.DefaultConfig()
	slug := slugifyName(name)
	cfg.Tool.Name = slug
	cfg.Tool.Main = mainPath

	npmDist := cfg.Distributions["npm"]
	npmDist.Package = fmt.Sprintf("@%s/%s", slug, slug)
	cfg.Distributions["npm"] = npmDist

	uvDist := cfg.Distributions["uv"]
	uvDist.Package = slug
	cfg.Distributions["uv"] = uvDist

	gemDist := cfg.Distributions["gem"]
	gemDist.Package = slug
	cfg.Distributions["gem"] = gemDist

	return cfg, nil
}

func resolveProjectIdentity(wd string, opts InitOptions) (string, string, error) {
	name := strings.TrimSpace(opts.ToolName)
	mainPath := strings.TrimSpace(opts.ToolMain)
	if name != "" {
		name = slugifyName(name)
	}
	if mainPath != "" {
		mainPath = normalizeMainPath(mainPath)
		if name == "" {
			name = slugifyName(filepath.Base(filepath.Clean(mainPath)))
		}
		return name, mainPath, nil
	}

	candidates, err := discoverMainPackages(wd)
	if err != nil {
		return "", "", err
	}
	if name != "" {
		for _, candidate := range candidates {
			if slugifyName(filepath.Base(candidate)) == name {
				return name, normalizeMainPath(candidate), nil
			}
		}
		if len(candidates) == 1 {
			return name, normalizeMainPath(candidates[0]), nil
		}
		return "", "", fmt.Errorf("cannot select a Go main package for tool %q; pass --main explicitly", name)
	}

	cwdSlug := slugifyName(filepath.Base(wd))
	for _, candidate := range candidates {
		if slugifyName(filepath.Base(candidate)) == cwdSlug {
			return cwdSlug, normalizeMainPath(candidate), nil
		}
	}
	if len(candidates) == 1 {
		candidate := candidates[0]
		return slugifyName(filepath.Base(candidate)), normalizeMainPath(candidate), nil
	}
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("no Go main package found under cmd/*; pass --name and --main explicitly")
	}
	return "", "", fmt.Errorf("multiple Go main packages found under cmd/* (%s); pass --name and --main explicitly", strings.Join(candidates, ", "))
}

func discoverMainPackages(wd string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(wd, "cmd"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan Go commands: %w", err)
	}

	candidates := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(wd, "cmd", entry.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("scan Go command %s: %w", entry.Name(), err)
		}
		isMain := false
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, file.Name()), nil, parser.PackageClauseOnly)
			if err != nil {
				return nil, fmt.Errorf("parse Go package %s: %w", filepath.Join("cmd", entry.Name(), file.Name()), err)
			}
			if parsed.Name.Name == "main" {
				isMain = true
				break
			}
		}
		if isMain {
			candidates = append(candidates, filepath.Join("cmd", entry.Name()))
		}
	}
	sort.Strings(candidates)
	return candidates, nil
}

func normalizeMainPath(value string) string {
	clean := filepath.Clean(value)
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, "."+string(filepath.Separator)) {
		return filepath.ToSlash(clean)
	}
	return "./" + filepath.ToSlash(clean)
}

func slugifyName(value string) string {
	input := strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false

	for _, r := range input {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "omnidist"
	}
	return slug
}

// CreateNPMStructure creates the npm workspace directories for configured targets.
func CreateNPMStructure(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	dist, ok := cfg.Distributions["npm"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	layout := paths.NewLayout(cfg.EffectiveWorkspaceDir())
	baseDir := layout.NPMDir

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("create npm base directory %s: %w", baseDir, err)
	}

	metaDir := filepath.Join(baseDir, dist.Package)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("create npm meta directory %s: %w", metaDir, err)
	}

	for _, target := range cfg.Targets {
		pkgDir := fmt.Sprintf("%s-%s-%s", dist.Package, config.MapGoOSToNPM(target.OS), config.MapGoArchToNPM(target.Arch))
		if target.Variant != "" {
			pkgDir = fmt.Sprintf("%s-%s", pkgDir, target.Variant)
		}
		if err := os.MkdirAll(filepath.Join(baseDir, pkgDir, "bin"), 0755); err != nil {
			return fmt.Errorf("create npm platform bin directory for %s: %w", pkgDir, err)
		}
	}

	return nil
}

// CreateUVStructure creates the uv staging directory when uv distribution is configured.
func CreateUVStructure(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	dist, ok := cfg.Distributions["uv"]
	if !ok || strings.TrimSpace(dist.Package) == "" {
		return nil
	}

	layout := paths.NewLayout(cfg.EffectiveWorkspaceDir())
	if err := os.MkdirAll(layout.UVDistDir, 0755); err != nil {
		return fmt.Errorf("create uv dist directory %s: %w", layout.UVDistDir, err)
	}

	return nil
}

// EnsureWorkspaceGitignore creates `.gitignore` for workspace layout control when missing.
func EnsureWorkspaceGitignore(path string) error {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("gitignore path %s is a directory", path)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat gitignore %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create gitignore directory %s: %w", filepath.Dir(path), err)
	}

	content := "*\n!.gitignore\n!omnidist.yaml\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write gitignore %s: %w", path, err)
	}
	return nil
}

func saveProfilesConfig(cfg *config.Config, configPath string, force bool) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	var file struct {
		Profiles map[string]config.Config `yaml:"profiles"`
	}
	file.Profiles = map[string]config.Config{
		config.DefaultProfileName: *cfg,
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal profile config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	if !force {
		file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("create config file %s: %w", configPath, err)
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			return fmt.Errorf("write config file %s: %w", configPath, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close config file %s: %w", configPath, err)
		}
		return nil
	}

	tmp, err := os.CreateTemp(dir, ".omnidist.yaml-*")
	if err != nil {
		return fmt.Errorf("create temporary config in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("replace config file %s: %w", configPath, err)
	}
	return nil
}
