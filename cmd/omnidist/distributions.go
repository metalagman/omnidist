package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/metalagman/omnidist/internal/config"
)

type distribution string

const (
	distributionNPM distribution = "npm"
	distributionUV  distribution = "uv"
	distributionGem distribution = "gem"
)

var distributionExecutionOrder = []distribution{
	distributionNPM,
	distributionUV,
	distributionGem,
}

func resolveDistributions(cfg *config.Config, only string) ([]distribution, error) {
	enabledNames, err := cfg.EnabledDistributionNames()
	if err != nil {
		return nil, err
	}
	enabled := make(map[distribution]bool, len(enabledNames))
	for _, name := range enabledNames {
		enabled[distribution(name)] = true
	}

	selected := map[distribution]bool{
		distributionNPM: false,
		distributionUV:  false,
		distributionGem: false,
	}

	filter := strings.TrimSpace(only)
	if filter == "" {
		resolved := make([]distribution, 0, len(enabled))
		for _, dist := range distributionExecutionOrder {
			if enabled[dist] {
				resolved = append(resolved, dist)
			}
		}
		return resolved, nil
	}

	parts := strings.Split(filter, ",")
	for _, part := range parts {
		name := distribution(strings.ToLower(strings.TrimSpace(part)))
		switch name {
		case distributionNPM, distributionUV, distributionGem:
			if !enabled[name] {
				return nil, fmt.Errorf("invalid --only value %q: distribution %q is disabled (enabled: %s)", only, name, strings.Join(enabledNames, ","))
			}
			selected[name] = true
		case "":
			return nil, fmt.Errorf("invalid --only value %q: empty distribution name", only)
		default:
			return nil, fmt.Errorf("invalid --only value %q: unsupported distribution %q (allowed: npm,uv,gem)", only, part)
		}
	}

	var resolved []distribution
	for _, dist := range distributionExecutionOrder {
		if selected[dist] {
			resolved = append(resolved, dist)
		}
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("invalid --only value %q: expected at least one of npm,uv,gem", only)
	}

	return resolved, nil
}

func runDistributionSteps(distributions []distribution, run func(distribution) error) error {
	for _, dist := range distributions {
		if err := run(dist); err != nil {
			return err
		}
	}
	return nil
}

func distributionList(distributions []distribution) string {
	names := make([]string, 0, len(distributions))
	for _, dist := range distributions {
		names = append(names, string(dist))
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
