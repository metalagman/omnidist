package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultVersionFile is the default version file path when version.source is file.
	DefaultVersionFile = "VERSION"
	// DefaultWorkspaceDir is the root directory for omnidist generated artifacts.
	DefaultWorkspaceDir = ".omnidist"
	// DefaultProfileName is used when no profile is explicitly selected.
	DefaultProfileName = "default"
)

var (
	profileNamePattern    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	npmPackageNamePattern = regexp.MustCompile(`^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)
	gemPackageNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

const maxNPMPackageNameLength = 214

// Config is the root omnidist configuration loaded from omnidist.yaml.
type Config struct {
	Tool          ToolConfig
	Version       VersionConfig
	ReadmePath    string
	Targets       []Target
	Build         BuildConfig
	Distributions DistributionConfigs
	Runtime       RuntimeConfig

	selectedDistributions []DistributionName
}

// DistributionName identifies a supported release backend.
type DistributionName string

const (
	DistributionNPM DistributionName = "npm"
	DistributionUV  DistributionName = "uv"
	DistributionGem DistributionName = "gem"
)

var supportedDistributionNames = []DistributionName{
	DistributionNPM,
	DistributionUV,
	DistributionGem,
}

// SupportedDistributionNames returns the canonical backend execution order.
func SupportedDistributionNames() []DistributionName {
	return append([]DistributionName(nil), supportedDistributionNames...)
}

// ParseDistributionName parses a backend name.
func ParseDistributionName(raw string) (DistributionName, error) {
	name := DistributionName(strings.ToLower(strings.TrimSpace(raw)))
	switch name {
	case DistributionNPM, DistributionUV, DistributionGem:
		return name, nil
	default:
		return "", fmt.Errorf("invalid distribution %q: expected npm, uv, or gem", raw)
	}
}

// DistributionNameStrings converts typed backend names to YAML/CLI strings.
func DistributionNameStrings(names []DistributionName) []string {
	values := make([]string, len(names))
	for i, name := range names {
		values[i] = string(name)
	}
	return values
}

// SelectedDistributionNames returns selected distributions in deterministic execution order.
func (cfg *Config) SelectedDistributionNames() ([]DistributionName, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if cfg.selectedDistributions != nil {
		return append([]DistributionName(nil), cfg.selectedDistributions...), nil
	}
	names := cfg.Distributions.Names()
	if len(names) == 0 {
		return nil, fmt.Errorf("distributions must configure at least one of npm, uv, or gem")
	}
	return names, nil
}

// IsSelected reports whether a backend belongs to the aggregate selection.
func (cfg *Config) IsSelected(name DistributionName) bool {
	names, err := cfg.SelectedDistributionNames()
	if err != nil {
		return false
	}
	for _, selected := range names {
		if selected == name {
			return true
		}
	}
	return false
}

// RuntimeConfig stores resolved runtime metadata not persisted in YAML.
type RuntimeConfig struct {
	Profile      string `yaml:"-"`
	ProfilesMode bool   `yaml:"-"`
	WorkspaceDir string `yaml:"-"`
}

// ToolConfig configures the Go CLI binary to build and package.
type ToolConfig struct {
	Name string `yaml:"name"`
	Main string `yaml:"main"`
}

// VersionConfig defines where omnidist resolves the release version from.
type VersionConfig struct {
	Source string `yaml:"source"`
	File   string `yaml:"file,omitempty"`
	Fixed  string `yaml:"fixed,omitempty"`
}

// Target describes a Go build target and optional packaging variant.
type Target struct {
	OS      string `yaml:"os"`
	Arch    string `yaml:"arch"`
	Variant string `yaml:"variant,omitempty"`
}

// DistributionConfigs stores explicitly configured release backends.
type DistributionConfigs struct {
	NPM *NPMDistributionConfig `yaml:"npm,omitempty"`
	UV  *UVDistributionConfig  `yaml:"uv,omitempty"`
	Gem *GemDistributionConfig `yaml:"gem,omitempty"`
}

// Names returns configured backend names in canonical execution order.
func (d DistributionConfigs) Names() []DistributionName {
	names := make([]DistributionName, 0, 3)
	if d.NPM != nil {
		names = append(names, DistributionNPM)
	}
	if d.UV != nil {
		names = append(names, DistributionUV)
	}
	if d.Gem != nil {
		names = append(names, DistributionGem)
	}
	return names
}

// Has reports whether a backend has an explicit configuration section.
func (d DistributionConfigs) Has(name DistributionName) bool {
	switch name {
	case DistributionNPM:
		return d.NPM != nil
	case DistributionUV:
		return d.UV != nil
	case DistributionGem:
		return d.Gem != nil
	default:
		return false
	}
}

// NPMDistributionConfig stores npm packaging settings.
type NPMDistributionConfig struct {
	Package         string   `yaml:"package"`
	Aliases         []string `yaml:"aliases,omitempty"`
	PlatformPackage string   `yaml:"platform-package,omitempty"`
	Registry        string   `yaml:"registry,omitempty"`
	Access          string   `yaml:"access,omitempty"`
	PublishAuth     string   `yaml:"publish-auth,omitempty"`
	RepositoryURL   string   `yaml:"repository-url,omitempty"`
	License         string   `yaml:"license,omitempty"`
	Keywords        []string `yaml:"keywords,omitempty"`
	ReadmePath      string   `yaml:"readme-path,omitempty"`
	IncludeREADME   *bool    `yaml:"include-readme,omitempty"`
}

// UVDistributionConfig stores uv/PyPI packaging settings.
type UVDistributionConfig struct {
	Package       string `yaml:"package"`
	IndexURL      string `yaml:"index-url,omitempty"`
	LinuxTag      string `yaml:"linux-tag,omitempty"`
	ReadmePath    string `yaml:"readme-path,omitempty"`
	IncludeREADME *bool  `yaml:"include-readme,omitempty"`
}

// GemDistributionConfig stores RubyGems packaging settings.
type GemDistributionConfig struct {
	Package       string `yaml:"package"`
	Registry      string `yaml:"registry,omitempty"`
	PublishAuth   string `yaml:"publish-auth,omitempty"`
	RepositoryURL string `yaml:"repository-url,omitempty"`
	License       string `yaml:"license,omitempty"`
	ReadmePath    string `yaml:"readme-path,omitempty"`
	IncludeREADME *bool  `yaml:"include-readme,omitempty"`
}

type rawNPMDistributionConfig NPMDistributionConfig
type rawUVDistributionConfig UVDistributionConfig
type rawGemDistributionConfig GemDistributionConfig

type rawDistributionConfigs struct {
	NPM *rawNPMDistributionConfig `yaml:"npm,omitempty"`
	UV  *rawUVDistributionConfig  `yaml:"uv,omitempty"`
	Gem *rawGemDistributionConfig `yaml:"gem,omitempty"`
}

type rawDistributionSelector struct {
	present bool
	null    bool
	values  []string
}

type rawConfig struct {
	Tool                 ToolConfig             `yaml:"tool"`
	Version              VersionConfig          `yaml:"version"`
	ReadmePath           string                 `yaml:"readme-path,omitempty"`
	Targets              []Target               `yaml:"targets"`
	Build                BuildConfig            `yaml:"build"`
	EnabledDistributions yaml.Node              `yaml:"enabled-distributions,omitempty"`
	Distributions        rawDistributionConfigs `yaml:"distributions"`
}

type rawProfilesDocument struct {
	Profiles map[string]rawConfig `yaml:"profiles"`
}

type persistedConfig struct {
	Tool          ToolConfig          `yaml:"tool"`
	Version       VersionConfig       `yaml:"version"`
	ReadmePath    string              `yaml:"readme-path,omitempty"`
	Targets       []Target            `yaml:"targets"`
	Build         BuildConfig         `yaml:"build"`
	Distributions DistributionConfigs `yaml:"distributions"`
}

// MarshalYAML prevents resolved runtime configuration from being serialized as source YAML.
func (Config) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("resolved Config cannot be marshaled directly; use a configuration writer")
}

func persistedConfigFromResolved(cfg *Config) (persistedConfig, error) {
	if cfg == nil {
		return persistedConfig{}, fmt.Errorf("config is nil")
	}
	selected, err := cfg.SelectedDistributionNames()
	if err != nil {
		return persistedConfig{}, err
	}
	configured := cfg.Distributions.Names()
	if !sameDistributionNames(selected, configured) {
		return persistedConfig{}, fmt.Errorf("cannot write canonical config with inactive legacy distributions (selected: %s; configured: %s)", strings.Join(DistributionNameStrings(selected), ","), strings.Join(DistributionNameStrings(configured), ","))
	}
	return persistedConfig{
		Tool:          cfg.Tool,
		Version:       cfg.Version,
		ReadmePath:    cfg.ReadmePath,
		Targets:       append([]Target(nil), cfg.Targets...),
		Build:         cfg.Build,
		Distributions: cfg.Distributions,
	}, nil
}

func sameDistributionNames(a, b []DistributionName) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MarshalProfiles serializes one resolved configuration in profiles format.
func MarshalProfiles(profile string, cfg *Config) ([]byte, error) {
	if err := validateProfileName(normalizeProfile(profile)); err != nil {
		return nil, err
	}
	persisted, err := persistedConfigFromResolved(cfg)
	if err != nil {
		return nil, err
	}
	file := struct {
		Profiles map[string]persistedConfig `yaml:"profiles"`
	}{
		Profiles: map[string]persistedConfig{normalizeProfile(profile): persisted},
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("marshal profile config: %w", err)
	}
	return data, nil
}

// WriteProfiles writes a canonical profiles configuration without a legacy selector.
func WriteProfiles(path string, profile string, cfg *Config, force bool) error {
	data, err := MarshalProfiles(profile, cfg)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}
	if !force {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("create config file %s: %w", path, err)
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			return fmt.Errorf("write config file %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close config file %s: %w", path, err)
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
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config file %s: %w", path, err)
	}
	return nil
}

// ValidateNPMPackageName checks the npm package-name grammar used by Omnidist.
func ValidateNPMPackageName(name string) error {
	if len(name) > maxNPMPackageNameLength {
		return fmt.Errorf("package name exceeds %d characters", maxNPMPackageNameLength)
	}
	if !npmPackageNamePattern.MatchString(name) {
		return fmt.Errorf("expected a lowercase package name with an optional @scope/ prefix")
	}
	return nil
}

// NPMPlatformPackageName returns the npm package name for a target-specific binary.
func NPMPlatformPackageName(base string, target Target) string {
	name := base + "-" + MapGoOSToNPM(target.OS) + "-" + MapGoArchToNPM(target.Arch)
	if target.Variant != "" {
		name += "-" + target.Variant
	}
	return name
}

// ValidateNPMPlatformPackage checks a configured base and every target name it produces.
func ValidateNPMPlatformPackage(base string, targets []Target) error {
	if err := ValidateNPMPackageName(base); err != nil {
		return err
	}
	for _, target := range targets {
		name := NPMPlatformPackageName(base, target)
		if err := ValidateNPMPackageName(name); err != nil {
			return fmt.Errorf("generated package %q for target %s/%s: %w", name, target.OS, target.Arch, err)
		}
	}
	return nil
}

// IncludeREADMEEnabled reports whether README.md should be included in staged artifacts.
func includeREADMEEnabled(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

// IncludeREADMEEnabled reports whether README.md should be included in npm artifacts.
func (d NPMDistributionConfig) IncludeREADMEEnabled() bool {
	return includeREADMEEnabled(d.IncludeREADME)
}

// MetaPackages returns the primary npm package followed by configured aliases.
func (d NPMDistributionConfig) MetaPackages() []string {
	packages := make([]string, 1, len(d.Aliases)+1)
	packages[0] = d.Package
	return append(packages, d.Aliases...)
}

// IncludeREADMEEnabled reports whether README.md should be included in uv artifacts.
func (d UVDistributionConfig) IncludeREADMEEnabled() bool {
	return includeREADMEEnabled(d.IncludeREADME)
}

// IncludeREADMEEnabled reports whether README.md should be included in gem artifacts.
func (d GemDistributionConfig) IncludeREADMEEnabled() bool {
	return includeREADMEEnabled(d.IncludeREADME)
}

// LicenseValue reports the configured package license value after trimming whitespace.
func (d NPMDistributionConfig) LicenseValue() string {
	return strings.TrimSpace(d.License)
}

// LicenseValue reports the configured gem license value after trimming whitespace.
func (d GemDistributionConfig) LicenseValue() string { return strings.TrimSpace(d.License) }

// RepositoryURLValue reports the configured repository URL after trimming whitespace.
func (d NPMDistributionConfig) RepositoryURLValue() string {
	return strings.TrimSpace(d.RepositoryURL)
}

// RepositoryURLValue reports the configured gem repository URL after trimming whitespace.
func (d GemDistributionConfig) RepositoryURLValue() string {
	return strings.TrimSpace(d.RepositoryURL)
}

// MapGoArchToNPM converts a Go GOARCH value to the corresponding npm cpu value.
func MapGoArchToNPM(arch string) string {
	switch arch {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return arch
	}
}

// MapGoOSToNPM converts a Go GOOS value to the corresponding npm os value.
func MapGoOSToNPM(goOS string) string {
	switch goOS {
	case "windows":
		return "win32"
	default:
		return goOS
	}
}

// BuildConfig configures go build flags shared across targets.
type BuildConfig struct {
	Ldflags string   `yaml:"ldflags"`
	Tags    []string `yaml:"tags"`
	CGO     bool     `yaml:"cgo"`
}

// DefaultConfig returns the default omnidist configuration for a new project.
func DefaultConfig() *Config {
	cfg := &Config{
		Tool: ToolConfig{
			Name: "omnidist",
			Main: "./cmd/omnidist",
		},
		Version: VersionConfig{
			Source: "git-tag",
		},
		Targets: []Target{
			{OS: "darwin", Arch: "amd64"},
			{OS: "darwin", Arch: "arm64"},
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
			{OS: "windows", Arch: "amd64"},
		},
		Build: BuildConfig{
			Ldflags: "-s -w",
			Tags:    []string{},
			CGO:     false,
		},
		Distributions: DistributionConfigs{
			NPM: &NPMDistributionConfig{
				Package:       "@omnidist/omnidist",
				Registry:      "https://registry.npmjs.org",
				Access:        "public",
				PublishAuth:   "token",
				IncludeREADME: boolPtr(true),
			},
			UV: &UVDistributionConfig{
				Package:       "omnidist",
				IndexURL:      "https://upload.pypi.org/legacy/",
				LinuxTag:      "manylinux2014",
				IncludeREADME: boolPtr(true),
			},
			Gem: &GemDistributionConfig{
				Package:       "omnidist",
				Registry:      "https://rubygems.org",
				PublishAuth:   "token",
				IncludeREADME: boolPtr(true),
			},
		},
	}
	applyRuntimeDefaults(cfg, DefaultProfileName, false)
	return cfg
}

// Load reads and validates an omnidist configuration file from path.
func Load(path string) (*Config, error) {
	return LoadWithProfile(path, "")
}

// LoadWithProfile reads and validates a config file and resolves a selected profile.
func LoadWithProfile(path string, profile string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	if hasLegacyFixedVersionKey(data) {
		return nil, fmt.Errorf("version.fixed-version is no longer supported; use version.fixed")
	}

	root, err := parseRootMap(data, path)
	if err != nil {
		return nil, err
	}

	hasProfiles := hasRootKey(root, "profiles")
	hasLegacyFields := hasTopLevelLegacyFields(root)
	if hasProfiles && hasLegacyFields {
		return nil, fmt.Errorf("invalid config file %s: mixed format is not supported; use either top-level config or profiles map", path)
	}

	if hasProfiles {
		return loadProfileConfig(path, data, profile)
	}
	return loadLegacyConfig(path, data)
}

func loadLegacyConfig(path string, data []byte) (*Config, error) {
	var raw rawConfig
	if err := decodeStrict(path, data, &raw); err != nil {
		return nil, err
	}
	cfg, err := resolveRawConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", path, err)
	}
	applyRuntimeDefaults(cfg, DefaultProfileName, false)

	return cfg, nil
}

func loadProfileConfig(path string, data []byte, selected string) (*Config, error) {
	var file rawProfilesDocument
	if err := decodeStrict(path, data, &file); err != nil {
		return nil, err
	}
	if len(file.Profiles) == 0 {
		return nil, fmt.Errorf("invalid config file %s: profiles map is empty", path)
	}

	selectedProfile := normalizeProfile(selected)
	if err := validateProfileName(selectedProfile); err != nil {
		return nil, err
	}

	raw, ok := file.Profiles[selectedProfile]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in %s; available profiles: %s", selectedProfile, path, strings.Join(sortedProfileNames(file.Profiles), ", "))
	}

	for profileName := range file.Profiles {
		if err := validateProfileName(profileName); err != nil {
			return nil, err
		}
	}

	cfg, err := resolveRawConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid config file %s: profiles.%s: %w", path, selectedProfile, err)
	}
	applyRuntimeDefaults(cfg, selectedProfile, true)
	return cfg, nil
}

func decodeStrict(path string, data []byte, out interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("parse config file %s: multiple YAML documents are not supported", path)
		}
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

func resolveRawConfig(raw rawConfig) (*Config, error) {
	cfg := &Config{
		Tool:          raw.Tool,
		Version:       raw.Version,
		ReadmePath:    raw.ReadmePath,
		Targets:       append([]Target(nil), raw.Targets...),
		Build:         raw.Build,
		Distributions: resolveRawDistributions(raw.Distributions),
	}
	selector, err := decodeRawSelector(raw.EnabledDistributions)
	if err != nil {
		return nil, err
	}
	selected, err := resolveRawSelection(selector, cfg.Distributions)
	if err != nil {
		return nil, err
	}
	cfg.selectedDistributions = selected
	applyVersionDefaults(cfg)
	applyDistributionDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func decodeRawSelector(node yaml.Node) (rawDistributionSelector, error) {
	if node.Kind == 0 {
		return rawDistributionSelector{}, nil
	}
	selector := rawDistributionSelector{present: true}
	if node.Tag == "!!null" {
		selector.null = true
		return selector, nil
	}
	if node.Kind != yaml.SequenceNode {
		return rawDistributionSelector{}, fmt.Errorf("enabled-distributions must be a sequence")
	}
	if err := node.Decode(&selector.values); err != nil {
		return rawDistributionSelector{}, fmt.Errorf("decode enabled-distributions: %w", err)
	}
	return selector, nil
}

func resolveRawDistributions(raw rawDistributionConfigs) DistributionConfigs {
	var resolved DistributionConfigs
	if raw.NPM != nil {
		value := NPMDistributionConfig(*raw.NPM)
		resolved.NPM = &value
	}
	if raw.UV != nil {
		value := UVDistributionConfig(*raw.UV)
		resolved.UV = &value
	}
	if raw.Gem != nil {
		value := GemDistributionConfig(*raw.Gem)
		resolved.Gem = &value
	}
	return resolved
}

func resolveRawSelection(selector rawDistributionSelector, distributions DistributionConfigs) ([]DistributionName, error) {
	if !selector.present {
		names := distributions.Names()
		if len(names) == 0 {
			return nil, fmt.Errorf("distributions must configure at least one of npm, uv, or gem")
		}
		return names, nil
	}
	if selector.null {
		return nil, fmt.Errorf("enabled-distributions must not be null")
	}
	if len(selector.values) == 0 {
		return nil, fmt.Errorf("enabled-distributions must contain at least one of npm, uv, or gem")
	}

	seen := make(map[DistributionName]bool, len(selector.values))
	for _, rawName := range selector.values {
		name, err := ParseDistributionName(rawName)
		if err != nil {
			return nil, fmt.Errorf("invalid enabled distribution %q: expected npm, uv, or gem", rawName)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate enabled distribution %q", name)
		}
		if !distributions.Has(name) {
			return nil, fmt.Errorf("distributions.%s is required because enabled-distributions selects %q", name, name)
		}
		seen[name] = true
	}

	selected := make([]DistributionName, 0, len(seen))
	for _, name := range supportedDistributionNames {
		if seen[name] {
			selected = append(selected, name)
		}
	}
	return selected, nil
}

func applyVersionDefaults(cfg *Config) {
	cfg.Version.Source = strings.TrimSpace(cfg.Version.Source)
	cfg.Version.File = strings.TrimSpace(cfg.Version.File)
	cfg.Version.Fixed = strings.TrimSpace(cfg.Version.Fixed)
	if cfg.Version.Source == "" {
		cfg.Version.Source = "git-tag"
	}
	if cfg.Version.Source == "file" && cfg.Version.File == "" {
		cfg.Version.File = DefaultVersionFile
	}
}

func hasLegacyFixedVersionKey(data []byte) bool {
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	return containsLegacyFixedVersionKey(raw)
}

func containsLegacyFixedVersionKey(raw interface{}) bool {
	switch typed := raw.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			if key == "fixed-version" {
				return true
			}
			if containsLegacyFixedVersionKey(value) {
				return true
			}
		}
	case map[interface{}]interface{}:
		for key, value := range typed {
			if keyStr, ok := key.(string); ok && keyStr == "fixed-version" {
				return true
			}
			if containsLegacyFixedVersionKey(value) {
				return true
			}
		}
	case []interface{}:
		for _, value := range typed {
			if containsLegacyFixedVersionKey(value) {
				return true
			}
		}
	}
	return false
}

func applyDistributionDefaults(cfg *Config) {
	cfg.ReadmePath = strings.TrimSpace(cfg.ReadmePath)

	if cfg.Distributions.NPM != nil {
		npmDist := *cfg.Distributions.NPM
		normalizeNPMDistribution(&npmDist)
		cfg.Distributions.NPM = &npmDist
	}

	if cfg.Distributions.UV != nil {
		uvDist := *cfg.Distributions.UV
		normalizeUVDistribution(&uvDist)
		cfg.Distributions.UV = &uvDist
	}

	if cfg.Distributions.Gem != nil {
		gemDist := *cfg.Distributions.Gem
		normalizeGemDistribution(&gemDist)
		cfg.Distributions.Gem = &gemDist
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func normalizeKeywords(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	keywords := make([]string, 0, len(values))
	for _, raw := range values {
		keyword := strings.TrimSpace(raw)
		if keyword == "" {
			continue
		}
		if _, ok := seen[keyword]; ok {
			continue
		}
		seen[keyword] = struct{}{}
		keywords = append(keywords, keyword)
	}
	if len(keywords) == 0 {
		return nil
	}
	return keywords
}

func parseRootMap(data []byte, path string) (map[string]interface{}, error) {
	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}
	if root == nil {
		return map[string]interface{}{}, nil
	}
	return root, nil
}

func hasRootKey(root map[string]interface{}, key string) bool {
	_, ok := root[key]
	return ok
}

func hasTopLevelLegacyFields(root map[string]interface{}) bool {
	for _, key := range []string{"tool", "version", "readme-path", "targets", "build", "enabled-distributions", "distributions"} {
		if hasRootKey(root, key) {
			return true
		}
	}
	return false
}

func normalizeProfile(selected string) string {
	v := strings.TrimSpace(selected)
	if v == "" {
		return DefaultProfileName
	}
	return v
}

func validateProfileName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("profile name is empty")
	}
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: allowed characters are letters, digits, dot, underscore, and hyphen", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid profile name %q", name)
	}
	return nil
}

func sortedProfileNames(profiles map[string]rawConfig) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func applyRuntimeDefaults(cfg *Config, profile string, profilesMode bool) {
	if cfg == nil {
		return
	}
	selected := normalizeProfile(profile)
	workspace := DefaultWorkspaceDir
	if profilesMode {
		workspace = path.Join(DefaultWorkspaceDir, selected)
	}

	cfg.Runtime.Profile = selected
	cfg.Runtime.ProfilesMode = profilesMode
	cfg.Runtime.WorkspaceDir = workspace
}

// EffectiveWorkspaceDir returns the artifact workspace root for this config.
func (cfg *Config) EffectiveWorkspaceDir() string {
	if cfg == nil {
		return DefaultWorkspaceDir
	}
	workspace := strings.TrimSpace(cfg.Runtime.WorkspaceDir)
	if workspace == "" {
		return DefaultWorkspaceDir
	}
	return workspace
}

// SelectedProfile returns the resolved profile name for this config.
func (cfg *Config) SelectedProfile() string {
	if cfg == nil {
		return DefaultProfileName
	}
	return normalizeProfile(cfg.Runtime.Profile)
}

// IsProfilesMode reports whether this config was loaded from `profiles`.
func (cfg *Config) IsProfilesMode() bool {
	return cfg != nil && cfg.Runtime.ProfilesMode
}

// RequireNPM returns validated npm configuration or reports that it is absent.
func (cfg *Config) RequireNPM() (NPMDistributionConfig, error) {
	if cfg == nil {
		return NPMDistributionConfig{}, fmt.Errorf("config is nil")
	}
	if cfg.Distributions.NPM == nil {
		return NPMDistributionConfig{}, fmt.Errorf("distributions.npm is required")
	}
	dist := *cfg.Distributions.NPM
	normalizeNPMDistribution(&dist)
	if err := validateNPMDistribution(dist, cfg.Targets); err != nil {
		return NPMDistributionConfig{}, err
	}
	if dist.PlatformPackage == "" {
		dist.PlatformPackage = dist.Package
	}
	return dist, nil
}

// RequireUV returns validated uv configuration or reports that it is absent.
func (cfg *Config) RequireUV() (UVDistributionConfig, error) {
	if cfg == nil {
		return UVDistributionConfig{}, fmt.Errorf("config is nil")
	}
	if cfg.Distributions.UV == nil {
		return UVDistributionConfig{}, fmt.Errorf("distributions.uv is required")
	}
	dist := *cfg.Distributions.UV
	normalizeUVDistribution(&dist)
	if err := validateUVDistribution(dist); err != nil {
		return UVDistributionConfig{}, err
	}
	return dist, nil
}

// RequireGem returns validated RubyGems configuration or reports that it is absent.
func (cfg *Config) RequireGem() (GemDistributionConfig, error) {
	if cfg == nil {
		return GemDistributionConfig{}, fmt.Errorf("config is nil")
	}
	if cfg.Distributions.Gem == nil {
		return GemDistributionConfig{}, fmt.Errorf("distributions.gem is required")
	}
	dist := *cfg.Distributions.Gem
	normalizeGemDistribution(&dist)
	if err := validateGemDistribution(dist); err != nil {
		return GemDistributionConfig{}, err
	}
	return dist, nil
}

func normalizeNPMDistribution(dist *NPMDistributionConfig) {
	dist.Package = strings.TrimSpace(dist.Package)
	dist.Aliases = append([]string(nil), dist.Aliases...)
	for i := range dist.Aliases {
		dist.Aliases[i] = strings.TrimSpace(dist.Aliases[i])
	}
	dist.PlatformPackage = strings.TrimSpace(dist.PlatformPackage)
	dist.Registry = strings.TrimSpace(dist.Registry)
	dist.Access = strings.TrimSpace(dist.Access)
	dist.PublishAuth = strings.TrimSpace(dist.PublishAuth)
	dist.RepositoryURL = dist.RepositoryURLValue()
	dist.License = dist.LicenseValue()
	dist.Keywords = normalizeKeywords(dist.Keywords)
	dist.ReadmePath = strings.TrimSpace(dist.ReadmePath)
	if dist.Registry == "" {
		dist.Registry = "https://registry.npmjs.org"
	}
	if dist.Access == "" {
		dist.Access = "public"
	}
	if dist.PublishAuth == "" {
		dist.PublishAuth = "token"
	}
	if dist.IncludeREADME == nil {
		dist.IncludeREADME = boolPtr(true)
	}
}

func normalizeUVDistribution(dist *UVDistributionConfig) {
	dist.Package = strings.TrimSpace(dist.Package)
	dist.ReadmePath = strings.TrimSpace(dist.ReadmePath)
	dist.IndexURL = strings.TrimSpace(dist.IndexURL)
	dist.LinuxTag = strings.TrimSpace(dist.LinuxTag)
	if dist.IndexURL == "" {
		dist.IndexURL = "https://upload.pypi.org/legacy/"
	}
	if dist.LinuxTag == "" {
		dist.LinuxTag = "manylinux2014"
	}
	if dist.IncludeREADME == nil {
		dist.IncludeREADME = boolPtr(true)
	}
}

func normalizeGemDistribution(dist *GemDistributionConfig) {
	dist.Package = strings.TrimSpace(dist.Package)
	dist.Registry = strings.TrimSpace(dist.Registry)
	dist.PublishAuth = strings.TrimSpace(dist.PublishAuth)
	dist.RepositoryURL = dist.RepositoryURLValue()
	dist.License = dist.LicenseValue()
	dist.ReadmePath = strings.TrimSpace(dist.ReadmePath)
	if dist.Registry == "" {
		dist.Registry = "https://rubygems.org"
	}
	if dist.PublishAuth == "" {
		dist.PublishAuth = "token"
	}
	if dist.IncludeREADME == nil {
		dist.IncludeREADME = boolPtr(true)
	}
}

func validateNPMDistribution(dist NPMDistributionConfig, targets []Target) error {
	if dist.Package == "" {
		return fmt.Errorf("distributions.npm.package is required")
	}
	if err := ValidateNPMPackageName(dist.Package); err != nil {
		return fmt.Errorf("invalid distributions.npm.package %q: %w", dist.Package, err)
	}
	seenAliases := map[string]int{dist.Package: -1}
	for i, alias := range dist.Aliases {
		if err := ValidateNPMPackageName(alias); err != nil {
			return fmt.Errorf("invalid distributions.npm.aliases[%d] %q: %w", i, alias, err)
		}
		if previous, ok := seenAliases[alias]; ok {
			if previous == -1 {
				return fmt.Errorf("distributions.npm.aliases[%d] duplicates distributions.npm.package %q", i, alias)
			}
			return fmt.Errorf("distributions.npm.aliases[%d] duplicates distributions.npm.aliases[%d] %q", i, previous, alias)
		}
		seenAliases[alias] = i
	}
	if dist.PlatformPackage != "" {
		if err := ValidateNPMPlatformPackage(dist.PlatformPackage, targets); err != nil {
			return fmt.Errorf("invalid distributions.npm.platform-package %q: %w", dist.PlatformPackage, err)
		}
	}
	switch dist.Access {
	case "", "public", "restricted":
	default:
		return fmt.Errorf("invalid distributions.npm.access %q: expected public or restricted", dist.Access)
	}
	switch dist.PublishAuth {
	case "", "token", "trusted":
	default:
		return fmt.Errorf("invalid distributions.npm.publish-auth %q: expected token or trusted", dist.PublishAuth)
	}
	if dist.PublishAuth == "trusted" && dist.RepositoryURLValue() == "" {
		return fmt.Errorf("distributions.npm.repository-url is required when distributions.npm.publish-auth is %q", "trusted")
	}
	return nil
}

func validateUVDistribution(dist UVDistributionConfig) error {
	if dist.Package == "" {
		return fmt.Errorf("distributions.uv.package is required")
	}
	switch dist.LinuxTag {
	case "manylinux2014", "musllinux_1_2":
	default:
		return fmt.Errorf("invalid distributions.uv.linux-tag %q: expected manylinux2014 or musllinux_1_2", dist.LinuxTag)
	}
	return nil
}

func validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := validateTargets(cfg.Targets); err != nil {
		return err
	}

	source := strings.TrimSpace(cfg.Version.Source)
	if source == "" {
		source = "git-tag"
	}
	switch source {
	case "git-tag", "file", "env", "fixed":
	default:
		return fmt.Errorf("invalid version.source %q: expected git-tag, file, env, or fixed", cfg.Version.Source)
	}
	if source == "fixed" && strings.TrimSpace(cfg.Version.Fixed) == "" {
		return fmt.Errorf("version.fixed is required when version.source is %q", "fixed")
	}

	selectedNames, err := cfg.SelectedDistributionNames()
	if err != nil {
		return err
	}
	for _, name := range selectedNames {
		if !cfg.Distributions.Has(name) {
			return fmt.Errorf("distributions.%s is required because it is selected", name)
		}
		switch name {
		case DistributionNPM:
			if _, err := cfg.RequireNPM(); err != nil {
				return err
			}
		case DistributionUV:
			if _, err := cfg.RequireUV(); err != nil {
				return err
			}
		case DistributionGem:
			if _, err := cfg.RequireGem(); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateTargets(targets []Target) error {
	for i, target := range targets {
		if strings.TrimSpace(target.OS) == "" {
			return fmt.Errorf("targets[%d].os is required", i)
		}
		if strings.TrimSpace(target.Arch) == "" {
			return fmt.Errorf("targets[%d].arch is required", i)
		}
		if target.OS == "win32" {
			return fmt.Errorf("invalid targets[%d].os %q: use Go GOOS value %q", i, target.OS, "windows")
		}
		if target.Arch == "x64" {
			return fmt.Errorf("invalid targets[%d].arch %q: use Go GOARCH value %q", i, target.Arch, "amd64")
		}
	}
	return nil
}

func validateGemDistribution(dist GemDistributionConfig) error {
	if dist.Package == "" {
		return fmt.Errorf("distributions.gem.package is required")
	}
	if !gemPackageNamePattern.MatchString(dist.Package) {
		return fmt.Errorf("invalid distributions.gem.package %q", dist.Package)
	}
	switch dist.PublishAuth {
	case "", "token", "trusted":
	default:
		return fmt.Errorf("invalid distributions.gem.publish-auth %q: expected token or trusted", dist.PublishAuth)
	}
	if dist.PublishAuth == "trusted" && dist.Registry != "" && dist.Registry != "https://rubygems.org" {
		return fmt.Errorf("distributions.gem.publish-auth %q requires distributions.gem.registry %q", "trusted", "https://rubygems.org")
	}
	return nil
}

// Save writes cfg to path in YAML format, creating parent directories as needed.
func Save(cfg *Config, path string) error {
	persisted, err := persistedConfigFromResolved(cfg)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(persisted)
	if err != nil {
		return fmt.Errorf("marshal config for %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}
