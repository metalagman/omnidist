package config

import (
	"strings"
	"testing"
)

func TestUVLicenseValidation(t *testing.T) {
	for _, tc := range []struct {
		license string
		valid   bool
	}{
		{"", true}, {"MIT", true}, {"MIT OR Apache-2.0", true},
		{"GPL-2.0-only WITH Classpath-exception-2.0", true},
		{"LicenseRef-Proprietary", true}, {"LicenseRef-Company.1", true},
		{"not-a-license", false}, {"MIT AND", false}, {"MIT\nInjected: header", false},
		{"DocumentRef-project:LicenseRef-MIT", false},
		{"LicenseRef-bad_name", false}, {"LicenseRef-", false},
	} {
		t.Run(tc.license, func(t *testing.T) {
			for _, inherited := range []bool{false, true} {
				cfg := &Config{Distributions: DistributionConfigs{UV: &UVDistributionConfig{Package: "mytool"}}}
				if inherited {
					cfg.License = tc.license
				} else {
					cfg.Distributions.UV.License = tc.license
				}
				_, err := cfg.RequireUV()
				if tc.valid && err != nil || !tc.valid && (err == nil || !strings.Contains(err.Error(), "distributions.pypi.license")) {
					t.Fatalf("license %q inherited=%v error=%v, want valid=%v", tc.license, inherited, err, tc.valid)
				}
			}
		})
	}
}
