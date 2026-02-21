package shared

import (
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestToPEP440(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "release", input: "1.2.3", want: "1.2.3"},
		{name: "dev", input: "1.2.3-dev.5.gabc123", want: "1.2.3.dev5+abc123"},
		{name: "git_describe", input: "1.2.3-5-gabc123", want: "1.2.3.dev5+abc123"},
		{name: "invalid", input: "1.2.3-rc1", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToPEP440(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ToPEP440(%q) error = nil, want error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ToPEP440(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ToPEP440(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestWheelPlatformTag(t *testing.T) {
	tests := []struct {
		name    string
		target  config.Target
		policy  string
		want    string
		wantErr bool
	}{
		{name: "linux_amd64", target: config.Target{OS: "linux", Arch: "amd64"}, policy: "manylinux2014", want: "manylinux2014_x86_64"},
		{name: "linux_arm64", target: config.Target{OS: "linux", Arch: "arm64"}, policy: "manylinux2014", want: "manylinux2014_aarch64"},
		{name: "darwin_arm64", target: config.Target{OS: "darwin", Arch: "arm64"}, policy: "manylinux2014", want: "macosx_11_0_arm64"},
		{name: "windows_amd64", target: config.Target{OS: "win32", Arch: "amd64"}, policy: "manylinux2014", want: "win_amd64"},
		{name: "invalid", target: config.Target{OS: "linux", Arch: "386"}, policy: "manylinux2014", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := WheelPlatformTag(tc.target, tc.policy)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("WheelPlatformTag(%+v) error = nil, want error", tc.target)
				}
				return
			}
			if err != nil {
				t.Fatalf("WheelPlatformTag(%+v) error = %v", tc.target, err)
			}
			if got != tc.want {
				t.Fatalf("WheelPlatformTag(%+v) = %q, want %q", tc.target, got, tc.want)
			}
		})
	}
}

func TestIsExactSemver(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "plain_semver", input: "1.2.3", want: true},
		{name: "with_prefix_v", input: "v1.2.3", want: false},
		{name: "prerelease", input: "1.2.3-rc.1", want: false},
		{name: "build_meta", input: "1.2.3+build.7", want: false},
		{name: "missing_patch", input: "1.2", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isExactSemver(tc.input)
			if got != tc.want {
				t.Fatalf("isExactSemver(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestResolveReleaseVersion(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		envVer  string
		want    string
		wantErr bool
	}{
		{
			name:   "env_exact_semver",
			cfg:    &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer: "1.2.3",
			want:   "1.2.3",
		},
		{
			name:    "env_non_semver",
			cfg:     &config.Config{Version: config.VersionConfig{Source: "env"}},
			envVer:  "1.2.3-dev.1.gabc",
			wantErr: true,
		},
		{
			name:    "nil_config",
			cfg:     nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VERSION", tc.envVer)
			got, err := ResolveReleaseVersion(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ResolveReleaseVersion() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveReleaseVersion() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("ResolveReleaseVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}
