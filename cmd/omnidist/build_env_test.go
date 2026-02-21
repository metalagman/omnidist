package main

import (
	"os"
	"testing"
	"time"
)

func TestSetBuildMetadataEnvSetsAndRestores(t *testing.T) {
	t.Setenv(envBuildDateName, "old-date")
	t.Setenv(envBuildCommitName, "old-commit")

	restore, err := setBuildMetadataEnv()
	if err != nil {
		t.Fatalf("setBuildMetadataEnv() error = %v", err)
	}

	buildDate := os.Getenv(envBuildDateName)
	if buildDate == "" {
		t.Fatalf("%s is empty", envBuildDateName)
	}
	if _, err := time.Parse(time.RFC3339, buildDate); err != nil {
		t.Fatalf("%s = %q, parse error = %v", envBuildDateName, buildDate, err)
	}

	restore()

	if got := os.Getenv(envBuildDateName); got != "old-date" {
		t.Fatalf("%s after restore = %q, want %q", envBuildDateName, got, "old-date")
	}
	if got := os.Getenv(envBuildCommitName); got != "old-commit" {
		t.Fatalf("%s after restore = %q, want %q", envBuildCommitName, got, "old-commit")
	}
}
