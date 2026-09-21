package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestResolveDistributions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		only    string
		want    []distribution
		wantErr bool
	}{
		{
			name: "default_all",
			want: []distribution{distributionNPM, distributionUV, distributionGem},
		},
		{
			name: "only_npm",
			only: "npm",
			want: []distribution{distributionNPM},
		},
		{
			name: "only_uv",
			only: "uv",
			want: []distribution{distributionUV},
		},
		{
			name: "both_preserves_execution_order",
			only: "uv,npm",
			want: []distribution{distributionNPM, distributionUV},
		},
		{
			name: "only_gem",
			only: "gem",
			want: []distribution{distributionGem},
		},
		{
			name:    "invalid_distribution",
			only:    "foo",
			wantErr: true,
		},
		{
			name:    "empty_distribution_token",
			only:    "npm,",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveDistributions(config.DefaultConfig(), tc.only)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveDistributions(%q) error = nil, want error", tc.only)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveDistributions(%q) error = %v", tc.only, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("resolveDistributions(%q) = %#v, want %#v", tc.only, got, tc.want)
			}
		})
	}
}

func TestResolveDistributionsHonorsConfiguredBackends(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Distributions.UV = nil

	got, err := resolveDistributions(cfg, "")
	if err != nil {
		t.Fatalf("resolveDistributions() error = %v", err)
	}
	want := []distribution{distributionNPM, distributionGem}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveDistributions() = %#v, want %#v", got, want)
	}

	_, err = resolveDistributions(cfg, "uv")
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("resolveDistributions(uv) error = %v, want unavailable error", err)
	}
}

func TestResolveDistributionsCoversEveryConfiguredSubset(t *testing.T) {
	t.Parallel()

	subsets := [][]distribution{
		{distributionNPM},
		{distributionUV},
		{distributionGem},
		{distributionNPM, distributionUV},
		{distributionNPM, distributionGem},
		{distributionUV, distributionGem},
		{distributionNPM, distributionUV, distributionGem},
	}
	for _, selected := range subsets {
		selected := selected
		t.Run(distributionList(selected), func(t *testing.T) {
			t.Parallel()
			cfg := config.DefaultConfig()
			configured := cfg.Distributions
			cfg.Distributions = config.DistributionConfigs{}
			for _, name := range selected {
				switch name {
				case distributionNPM:
					cfg.Distributions.NPM = configured.NPM
				case distributionUV:
					cfg.Distributions.UV = configured.UV
				case distributionGem:
					cfg.Distributions.Gem = configured.Gem
				}
			}
			got, err := resolveDistributions(cfg, "")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, selected) {
				t.Fatalf("resolveDistributions() = %#v, want %#v", got, selected)
			}
		})
	}
}

func TestRunDistributionSteps(t *testing.T) {
	t.Parallel()

	t.Run("success_runs_in_order", func(t *testing.T) {
		t.Parallel()
		order := []distribution{distributionNPM, distributionUV, distributionGem}
		calls := make([]distribution, 0, len(order))

		err := runDistributionSteps(order, func(dist distribution) error {
			calls = append(calls, dist)
			return nil
		})
		if err != nil {
			t.Fatalf("runDistributionSteps() error = %v", err)
		}
		if !reflect.DeepEqual(calls, order) {
			t.Fatalf("runDistributionSteps() calls = %#v, want %#v", calls, order)
		}
	})

	t.Run("fail_fast_stops_after_first_error", func(t *testing.T) {
		t.Parallel()
		order := []distribution{distributionNPM, distributionUV, distributionGem}
		calls := make([]distribution, 0, len(order))
		wantErr := errors.New("boom")

		err := runDistributionSteps(order, func(dist distribution) error {
			calls = append(calls, dist)
			if dist == distributionNPM {
				return wantErr
			}
			return nil
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("runDistributionSteps() error = %v, want %v", err, wantErr)
		}
		wantCalls := []distribution{distributionNPM}
		if !reflect.DeepEqual(calls, wantCalls) {
			t.Fatalf("runDistributionSteps() calls = %#v, want %#v", calls, wantCalls)
		}
	})
}

func TestDistributionListSortsNames(t *testing.T) {
	t.Parallel()

	got := distributionList([]distribution{distributionUV, distributionGem, distributionNPM})
	if got != "gem, npm, uv" {
		t.Fatalf("distributionList() = %q, want %q", got, "gem, npm, uv")
	}
}
