package workflow

import (
	"reflect"
	"testing"
)

func TestSplitVersionMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		wantVersion   string
		wantBuildMeta string
	}{
		{
			name:          "exact_version_without_metadata",
			input:         "1.2.3",
			wantVersion:   "1.2.3",
			wantBuildMeta: "",
		},
		{
			name:          "version_with_metadata",
			input:         "1.2.3+abc123",
			wantVersion:   "1.2.3",
			wantBuildMeta: "abc123",
		},
		{
			name:          "empty_input_defaults_to_dev",
			input:         " \n ",
			wantVersion:   "dev",
			wantBuildMeta: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotVersion, gotMeta := splitVersionMetadata(tc.input)
			if gotVersion != tc.wantVersion || gotMeta != tc.wantBuildMeta {
				t.Fatalf("splitVersionMetadata(%q) = (%q, %q), want (%q, %q)", tc.input, gotVersion, gotMeta, tc.wantVersion, tc.wantBuildMeta)
			}
		})
	}
}

func TestAppkitVersionLDFlags(t *testing.T) {
	t.Parallel()

	metadata := buildMetadata{
		version:   "1.2.3",
		metadata:  "abc123",
		gitCommit: "deadbeef",
		buildDate: "2026-02-21T00:00:00Z",
	}

	got := appkitVersionLDFlags(metadata)
	want := []string{
		"-X github.com/metalagman/appkit/version.version=1.2.3",
		"-X github.com/metalagman/appkit/version.metadata=abc123",
		"-X github.com/metalagman/appkit/version.gitCommit=deadbeef",
		"-X github.com/metalagman/appkit/version.buildDate=2026-02-21T00:00:00Z",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("appkitVersionLDFlags() = %#v, want %#v", got, want)
	}
}

func TestMergeLDFlags(t *testing.T) {
	t.Parallel()

	got := mergeLDFlags("-s -w", []string{"-X a.b.c=1", " ", "-X d.e.f=2"})
	want := "-s -w -X a.b.c=1 -X d.e.f=2"
	if got != want {
		t.Fatalf("mergeLDFlags() = %q, want %q", got, want)
	}
}
