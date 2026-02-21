package workflow

import "testing"

func TestRenderBuildLDFlags(t *testing.T) {
	t.Setenv("OMNIDIST_VERSION", "1.2.3")
	got := renderBuildLDFlags("-s -w -X github.com/metalagman/appkit/version.version=${OMNIDIST_VERSION}")
	want := "-s -w -X github.com/metalagman/appkit/version.version=1.2.3"
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
