package npm

import (
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
)

const defaultTrustWorkflowFile = "omnidist-release.yml"

// TrustOptions controls npm trusted publisher configuration generation.
type TrustOptions struct {
	Repository        string
	WorkflowFile      string
	Environment       string
	AllowPublish      bool
	AllowStagePublish bool
}

// TrustPlan describes the npm trust commands required for configured packages.
type TrustPlan struct {
	Repository        string
	WorkflowFile      string
	Environment       string
	AllowPublish      bool
	AllowStagePublish bool
	Packages          []string
}

// CommandArgs returns npm CLI arguments for each package in the trust plan.
func (p TrustPlan) CommandArgs() [][]string {
	args := make([][]string, 0, len(p.Packages))
	for _, pkg := range p.Packages {
		cmdArgs := []string{
			"trust", "github", pkg,
			"--repo", p.Repository,
			"--file", p.WorkflowFile,
		}
		if p.Environment != "" {
			cmdArgs = append(cmdArgs, "--env", p.Environment)
		}
		if p.AllowPublish {
			cmdArgs = append(cmdArgs, "--allow-publish")
		}
		if p.AllowStagePublish {
			cmdArgs = append(cmdArgs, "--allow-stage-publish")
		}
		cmdArgs = append(cmdArgs, "--yes")
		args = append(args, cmdArgs)
	}
	return args
}

// TrustedPublishingPlan derives the npm trusted publisher commands for the configured package set.
func TrustedPublishingPlan(cfg *config.Config, opts TrustOptions) (*TrustPlan, error) {
	npmDist, err := npmDistribution(cfg)
	if err != nil {
		return nil, err
	}

	repository := strings.TrimSpace(opts.Repository)
	if repository == "" {
		repository, err = githubRepositoryFromURL(npmDist.RepositoryURLValue())
		if err != nil {
			return nil, err
		}
	}

	workflowFile := strings.TrimSpace(opts.WorkflowFile)
	if workflowFile == "" {
		workflowFile = defaultTrustWorkflowFile
	}
	if strings.Contains(workflowFile, "/") {
		return nil, fmt.Errorf("workflow file must be a filename in .github/workflows, got %q", workflowFile)
	}

	allowStagePublish := opts.AllowStagePublish
	allowPublish := opts.AllowPublish || allowStagePublish
	if !allowPublish && !allowStagePublish {
		allowPublish = true
	}

	packages := trustPackages(cfg, npmDist.Package)
	if len(packages) == 0 {
		return nil, fmt.Errorf("no npm packages configured for trusted publishing")
	}

	return &TrustPlan{
		Repository:        repository,
		WorkflowFile:      workflowFile,
		Environment:       strings.TrimSpace(opts.Environment),
		AllowPublish:      allowPublish,
		AllowStagePublish: allowStagePublish,
		Packages:          packages,
	}, nil
}

func trustPackages(cfg *config.Config, metaPackage string) []string {
	if cfg == nil {
		return nil
	}

	packages := []string{metaPackage}
	seen := map[string]struct{}{metaPackage: {}}
	for _, target := range cfg.Targets {
		pkgName := platformPackageName(metaPackage, target)
		if _, ok := seen[pkgName]; ok {
			continue
		}
		seen[pkgName] = struct{}{}
		packages = append(packages, pkgName)
	}

	platformPackages := append([]string{}, packages[1:]...)
	sort.Strings(platformPackages)
	return append([]string{metaPackage}, platformPackages...)
}

func githubRepositoryFromURL(raw string) (string, error) {
	repositoryURL := strings.TrimSpace(raw)
	if repositoryURL == "" {
		return "", fmt.Errorf("npm trusted publishing requires distributions.npm.repository-url or --repo")
	}

	if strings.HasPrefix(repositoryURL, "git@github.com:") {
		repositoryURL = "ssh://git@github.com/" + strings.TrimPrefix(repositoryURL, "git@github.com:")
	}
	repositoryURL = strings.TrimPrefix(repositoryURL, "git+")

	if strings.HasPrefix(repositoryURL, "github:") {
		repositoryURL = strings.TrimPrefix(repositoryURL, "github:")
		repositoryURL = strings.TrimSuffix(repositoryURL, ".git")
		repositoryURL = strings.Trim(repositoryURL, "/")
		if repositoryURL == "" || !strings.Contains(repositoryURL, "/") {
			return "", fmt.Errorf("invalid GitHub repository URL %q", raw)
		}
		return repositoryURL, nil
	}

	u, err := url.Parse(repositoryURL)
	if err != nil {
		return "", fmt.Errorf("parse repository URL %q: %w", raw, err)
	}
	if host := strings.ToLower(u.Hostname()); host != "github.com" {
		return "", fmt.Errorf("npm trusted publishing currently requires a GitHub repository URL, got %q", raw)
	}

	repository := strings.Trim(path.Clean(u.Path), "/")
	repository = strings.TrimSuffix(repository, ".git")
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid GitHub repository URL %q", raw)
	}
	return repository, nil
}
