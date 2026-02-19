package main

import (
	"fmt"
	"os"

	"github.com/metalagman/omnidist/internal/workflow"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap npm workspace packaging in existing Go repo",
	Run: func(cmd *cobra.Command, args []string) {
		if err := workflow.Init("omnidist.yaml"); err != nil {
			fmt.Fprintln(os.Stderr, "Error initializing project:", err)
			os.Exit(1)
		}

		fmt.Println("Created omnidist.yaml")
		fmt.Println("Created npm/ directory structure")
	},
}

func init() {
	AddCommand(initCmd)
}
