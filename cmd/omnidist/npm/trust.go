package npm

import (
	"fmt"
	"os/exec"
	"strings"

	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	"github.com/spf13/cobra"
)

const npmTrustCLISpec = "npm@11.16.0"

var (
	flagTrustWorkflowFile string
	flagTrustRepo         string
	flagTrustEnvironment  string
	flagTrustStagePublish bool
	flagTrustApply        bool
)

func init() {
	Cmd.AddCommand(trustCmd)
	trustCmd.Flags().StringVar(&flagTrustWorkflowFile, "workflow-file", "", "Workflow filename configured on npm trusted publishing (default: omnidist-release.yml)")
	trustCmd.Flags().StringVar(&flagTrustRepo, "repo", "", "GitHub repository override in owner/repo form")
	trustCmd.Flags().StringVar(&flagTrustEnvironment, "environment", "", "GitHub Actions environment name configured on npm")
	trustCmd.Flags().BoolVar(&flagTrustStagePublish, "allow-stage-publish", false, "Also allow npm stage publish for the trusted publisher")
	trustCmd.Flags().BoolVar(&flagTrustApply, "apply", false, "Run npm trust commands instead of only printing them")
}

var trustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Print or apply npm trusted publisher commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		plan, err := npmworkflow.TrustedPublishingPlan(cfg, npmworkflow.TrustOptions{
			Repository:        flagTrustRepo,
			WorkflowFile:      flagTrustWorkflowFile,
			Environment:       flagTrustEnvironment,
			AllowStagePublish: flagTrustStagePublish,
		})
		if err != nil {
			return fmt.Errorf("build npm trust plan: %w", err)
		}

		if !flagTrustApply {
			for _, args := range plan.CommandArgs() {
				fmt.Fprintln(cmd.OutOrStdout(), shellQuoteArgs("npx", append([]string{"-y", npmTrustCLISpec}, args...)))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nPrinted %d npm trust command(s)\n", len(plan.Packages))
			return nil
		}

		for _, args := range plan.CommandArgs() {
			npxArgs := append([]string{"-y", npmTrustCLISpec}, args...)
			fmt.Fprintf(cmd.OutOrStdout(), "Running: %s\n", shellQuoteArgs("npx", npxArgs))
			execCmd := exec.Command("npx", npxArgs...)
			execCmd.Stdout = cmd.OutOrStdout()
			execCmd.Stderr = cmd.ErrOrStderr()
			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("npm trust failed for %s: %w", args[2], err)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Configured npm trusted publishing for %d package(s)\n", len(plan.Packages))
		return nil
	},
}

func shellQuoteArgs(command string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(command))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$&;|<>*?()[]{}!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
