package commands

import (
	"github.com/spf13/cobra"
)

var npmCmd = &cobra.Command{
	Use:   "npm",
	Short: "NPM distribution commands",
}

func init() {
	AddCommand(npmCmd)
}
