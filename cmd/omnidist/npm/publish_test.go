package npm

import (
	"reflect"
	"testing"
)

func TestBuildPublishArgs(t *testing.T) {
	t.Parallel()

	origDryRun := flagDryRun
	origTag := flagTag
	origRegistry := flagRegistry
	origOTP := flagOTP
	t.Cleanup(func() {
		flagDryRun = origDryRun
		flagTag = origTag
		flagRegistry = origRegistry
		flagOTP = origOTP
	})

	flagDryRun = false
	flagTag = ""
	flagRegistry = ""
	flagOTP = ""

	got := buildPublishArgs("https://registry.npmjs.org", "public")
	want := []string{
		"publish",
		"--access", "public",
		"--registry", "https://registry.npmjs.org",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildPublishArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildPublishArgsFlagOverrides(t *testing.T) {
	t.Parallel()

	origDryRun := flagDryRun
	origTag := flagTag
	origRegistry := flagRegistry
	origOTP := flagOTP
	t.Cleanup(func() {
		flagDryRun = origDryRun
		flagTag = origTag
		flagRegistry = origRegistry
		flagOTP = origOTP
	})

	flagDryRun = true
	flagTag = "next"
	flagRegistry = "https://npm.example.internal"
	flagOTP = "123456"

	got := buildPublishArgs("https://registry.npmjs.org", "restricted")
	want := []string{
		"publish",
		"--access", "restricted",
		"--dry-run",
		"--tag", "next",
		"--registry", "https://npm.example.internal",
		"--otp", "123456",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildPublishArgs() = %#v, want %#v", got, want)
	}
}
