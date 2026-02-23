package workflow

import "testing"

func TestRenderBuildLDFlags(t *testing.T) {
	t.Setenv("OMNIDIST_VERSION", "1.2.3")
	t.Setenv("OMNIDIST_GIT_COMMIT", "deadbeef")
	t.Setenv("OMNIDIST_BUILD_DATE", "2026-02-22T01:02:03Z")
	got := renderBuildLDFlags("-s -w -X github.com/metalagman/appkit/version.version=${OMNIDIST_VERSION}")
	want := "-s -w -X github.com/metalagman/appkit/version.version=1.2.3"
	if got != want {
		t.Fatalf("renderBuildLDFlags() = %q, want %q", got, want)
	}

	got = renderBuildLDFlags("-X github.com/metalagman/appkit/version.gitCommit=${OMNIDIST_GIT_COMMIT} -X github.com/metalagman/appkit/version.buildDate=${OMNIDIST_BUILD_DATE}")
	want = "-X github.com/metalagman/appkit/version.gitCommit=deadbeef -X github.com/metalagman/appkit/version.buildDate=2026-02-22T01:02:03Z"
	if got != want {
		t.Fatalf("renderBuildLDFlags() = %q, want %q", got, want)
	}
}

func TestRenderBuildLDFlagsTrimsWhitespace(t *testing.T) {
	t.Parallel()

	got := renderBuildLDFlags("  -s -w  ")
	if got != "-s -w" {
		t.Fatalf("renderBuildLDFlags() = %q, want %q", got, "-s -w")
	}
}

func TestBuildTagsFlagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "empty",
			tags: nil,
			want: "",
		},
		{
			name: "single",
			tags: []string{"release"},
			want: "release",
		},
		{
			name: "multiple_trimmed",
			tags: []string{"tag1", " tag2 ", ""},
			want: "tag1,tag2",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := buildTagsFlagValue(tc.tags); got != tc.want {
				t.Fatalf("buildTagsFlagValue() = %q, want %q", got, tc.want)
			}
		})
	}
}
