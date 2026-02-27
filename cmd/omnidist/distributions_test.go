package main

import (
	"errors"
	"reflect"
	"testing"
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
			want: []distribution{distributionNPM, distributionUV},
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

			got, err := resolveDistributions(tc.only)
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

func TestRunDistributionSteps(t *testing.T) {
	t.Parallel()

	t.Run("success_runs_in_order", func(t *testing.T) {
		t.Parallel()
		order := []distribution{distributionNPM, distributionUV}
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
		order := []distribution{distributionNPM, distributionUV}
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

	got := distributionList([]distribution{distributionUV, distributionNPM})
	if got != "npm, uv" {
		t.Fatalf("distributionList() = %q, want %q", got, "npm, uv")
	}
}
