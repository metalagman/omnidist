package npm

import (
	"fmt"
	"os"

	"github.com/metalagman/omnidist/internal/config"
	npmworkflow "github.com/metalagman/omnidist/internal/workflow/npm"
	"github.com/metalagman/omnidist/internal/workflow/shared"
	"github.com/spf13/cobra"
)

var flagDev bool

func init() {
	Cmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVar(&flagDev, "dev", false, "Generate dev version (appends -dev.<commits> to version)")
}

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Assemble npm packages from built artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading config:", err)
			os.Exit(1)
		}

		version, err := shared.ResolveVersion(cfg, flagDev)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error resolving version:", err)
			os.Exit(1)
		}
		fmt.Println("Version:", version)

		if err := runStage(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "Error staging:", err)
			os.Exit(1)
		}

		fmt.Println("Staging completed successfully")
	},
}

func runStage(cfg *config.Config) error {
	return npmworkflow.Stage(cfg, npmworkflow.StageOptions{
		Dev: flagDev,
	})
}
