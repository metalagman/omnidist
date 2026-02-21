package main

import (
	"fmt"

	appversion "github.com/metalagman/appkit/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print omnidist version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), appversion.String())
	},
}

func init() {
	AddCommand(versionCmd)
}
