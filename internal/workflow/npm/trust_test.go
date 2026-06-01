package npm

import (
	"reflect"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
)

func TestTrustedPublishingPlan(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Targets = []config.Target{
		{OS: "linux", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
	}

	plan, err := TrustedPublishingPlan(cfg, TrustOptions{})
	if err != nil {
		t.Fatalf("TrustedPublishingPlan() error = %v", err)
	}

	if plan.Repository != "metalagman/omnidist" {
		t.Fatalf("plan.Repository = %q, want %q", plan.Repository, "metalagman/omnidist")
	}
	if plan.WorkflowFile != defaultTrustWorkflowFile {
		t.Fatalf("plan.WorkflowFile = %q, want %q", plan.WorkflowFile, defaultTrustWorkflowFile)
	}
	if !plan.AllowPublish {
		t.Fatalf("plan.AllowPublish = false, want true")
	}
	if plan.AllowStagePublish {
		t.Fatalf("plan.AllowStagePublish = true, want false")
	}

	wantPackages := []string{
		"@omnidist/omnidist",
		"@omnidist/omnidist-darwin-arm64",
		"@omnidist/omnidist-linux-x64",
	}
	if !reflect.DeepEqual(plan.Packages, wantPackages) {
		t.Fatalf("plan.Packages = %v, want %v", plan.Packages, wantPackages)
	}

	wantArgs := [][]string{
		{"trust", "github", "@omnidist/omnidist", "--repo", "metalagman/omnidist", "--file", "omnidist-release.yml", "--allow-publish", "--yes"},
		{"trust", "github", "@omnidist/omnidist-darwin-arm64", "--repo", "metalagman/omnidist", "--file", "omnidist-release.yml", "--allow-publish", "--yes"},
		{"trust", "github", "@omnidist/omnidist-linux-x64", "--repo", "metalagman/omnidist", "--file", "omnidist-release.yml", "--allow-publish", "--yes"},
	}
	if got := plan.CommandArgs(); !reflect.DeepEqual(got, wantArgs) {
		t.Fatalf("plan.CommandArgs() = %v, want %v", got, wantArgs)
	}
}

func TestTrustedPublishingPlanHonorsOverrides(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	plan, err := TrustedPublishingPlan(cfg, TrustOptions{
		Repository:        "acme/custom",
		WorkflowFile:      "publish.yaml",
		Environment:       "release",
		AllowStagePublish: true,
	})
	if err != nil {
		t.Fatalf("TrustedPublishingPlan() error = %v", err)
	}

	if plan.Repository != "acme/custom" {
		t.Fatalf("plan.Repository = %q, want %q", plan.Repository, "acme/custom")
	}
	if !plan.AllowPublish {
		t.Fatalf("plan.AllowPublish = false, want true when stage publish is enabled")
	}
	if !plan.AllowStagePublish {
		t.Fatalf("plan.AllowStagePublish = false, want true")
	}
	args := plan.CommandArgs()[0]
	want := []string{"trust", "github", "@omnidist/omnidist", "--repo", "acme/custom", "--file", "publish.yaml", "--env", "release", "--allow-publish", "--allow-stage-publish", "--yes"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("plan.CommandArgs()[0] = %v, want %v", args, want)
	}
}

func TestTrustedPublishingPlanErrors(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	npmDist := cfg.Distributions["npm"]
	npmDist.RepositoryURL = ""
	cfg.Distributions["npm"] = npmDist

	tests := []struct {
		name    string
		cfg     *config.Config
		opts    TrustOptions
		wantErr string
	}{
		{
			name:    "missing_repo",
			cfg:     cfg,
			opts:    TrustOptions{},
			wantErr: "repository-url",
		},
		{
			name:    "workflow_path_not_filename",
			cfg:     testConfig(),
			opts:    TrustOptions{WorkflowFile: ".github/workflows/publish.yml"},
			wantErr: "workflow file must be a filename",
		},
		{
			name: "non_github_repository",
			cfg: &config.Config{
				Targets: testConfig().Targets,
				Distributions: map[string]config.DistributionConfig{
					"npm": {
						Package:       "@omnidist/omnidist",
						Access:        "public",
						PublishAuth:   "trusted",
						RepositoryURL: "git+https://gitlab.com/acme/tool.git",
					},
				},
			},
			opts:    TrustOptions{},
			wantErr: "GitHub repository URL",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := TrustedPublishingPlan(tc.cfg, tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("TrustedPublishingPlan() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestGithubRepositoryFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "git_plus_https", in: "git+https://github.com/acme/tool.git", want: "acme/tool"},
		{name: "https", in: "https://github.com/acme/tool", want: "acme/tool"},
		{name: "ssh", in: "git@github.com:acme/tool.git", want: "acme/tool"},
		{name: "github_shorthand", in: "github:acme/tool", want: "acme/tool"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := githubRepositoryFromURL(tc.in)
			if err != nil {
				t.Fatalf("githubRepositoryFromURL() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("githubRepositoryFromURL() = %q, want %q", got, tc.want)
			}
		})
	}
}
