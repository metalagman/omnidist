package uv

import (
	"fmt"
	"os"

	"github.com/metalagman/omnidist/internal/workflow/shared"
	uvworkflow "github.com/metalagman/omnidist/internal/workflow/uv"
	"github.com/spf13/cobra"
)

var stageDev bool

func init() {
	Cmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVar(&stageDev, "dev", false, "Generate a dev version for wheel artifacts")
}

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Assemble uv wheel artifacts from built binaries",
	Run: func(cmd *cobra.Command, args []string) {
		if err := uvworkflow.CheckDependency(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		version, err := shared.ResolveStageVersion(cfg, stageDev)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error resolving version:", err)
			os.Exit(1)
		}
		pep440Version, err := shared.ToPEP440(version)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error resolving uv version:", err)
			os.Exit(1)
		}
		fmt.Println("Version:", pep440Version)

		if err := uvworkflow.Stage(cfg, uvworkflow.StageOptions{Dev: stageDev}); err != nil {
			fmt.Fprintln(os.Stderr, "Error staging uv artifacts:", err)
			os.Exit(1)
		}

		fmt.Println("UV staging completed successfully")
	},
}
