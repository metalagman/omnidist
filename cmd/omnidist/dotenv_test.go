package main

import (
	"os"
	"testing"
)

func TestInitDotEnvLoadsEnvFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	const key = "OMNIDIST_DOTENV_TEST"
	const value = "loaded-value"

	if err := os.WriteFile(".env", []byte(key+"="+value+"\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile(.env) error = %v", err)
	}
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
	}

	initDotEnv()

	if got := os.Getenv(key); got != value {
		t.Fatalf("%s = %q, want %q", key, got, value)
	}
}

func TestInitDotEnvMissingFileIsNoop(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	const key = "OMNIDIST_DOTENV_MISSING"
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
	}

	initDotEnv()

	if _, ok := os.LookupEnv(key); ok {
		t.Fatalf("%s unexpectedly set", key)
	}
}
