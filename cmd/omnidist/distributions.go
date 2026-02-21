package main

import (
	"fmt"
	"sort"
	"strings"
)

type distribution string

const (
	distributionNPM distribution = "npm"
	distributionUV  distribution = "uv"
)

var distributionExecutionOrder = []distribution{
	distributionNPM,
	distributionUV,
}

func resolveDistributions(only string) ([]distribution, error) {
	selected := map[distribution]bool{
		distributionNPM: false,
		distributionUV:  false,
	}

	filter := strings.TrimSpace(only)
	if filter == "" {
		return append([]distribution(nil), distributionExecutionOrder...), nil
	}

	parts := strings.Split(filter, ",")
	for _, part := range parts {
		name := distribution(strings.ToLower(strings.TrimSpace(part)))
		switch name {
		case distributionNPM, distributionUV:
			selected[name] = true
		case "":
			return nil, fmt.Errorf("invalid --only value %q: empty distribution name", only)
		default:
			return nil, fmt.Errorf("invalid --only value %q: unsupported distribution %q (allowed: npm,uv)", only, part)
		}
	}

	var resolved []distribution
	for _, dist := range distributionExecutionOrder {
		if selected[dist] {
			resolved = append(resolved, dist)
		}
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("invalid --only value %q: expected at least one of npm,uv", only)
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
