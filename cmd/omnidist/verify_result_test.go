package main

import (
	"strings"
	"testing"
)

func TestVerifyResult(t *testing.T) {
	t.Run("valid_with_warning", func(t *testing.T) {
		err := verifyResult("npm", nil, []string{"heads-up"}, true)
		if err != nil {
			t.Fatalf("verifyResult(valid=true) error = %v, want nil", err)
		}
	})

	t.Run("invalid_without_errors", func(t *testing.T) {
		err := verifyResult("uv", nil, nil, false)
		if err == nil || !strings.Contains(err.Error(), "uv verify failed") {
			t.Fatalf("verifyResult(valid=false, no errors) = %v, want generic failure", err)
		}
	})

	t.Run("invalid_with_errors", func(t *testing.T) {
		err := verifyResult("npm", []string{"bad one", "bad two"}, nil, false)
		if err == nil || !strings.Contains(err.Error(), "2 error(s)") {
			t.Fatalf("verifyResult(valid=false, errors) = %v, want count message", err)
		}
	})
}
