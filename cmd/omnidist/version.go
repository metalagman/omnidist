package main

import (
	"fmt"

	appversion "github.com/metalagman/appkit/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print omnidist version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), appversion.String())
		return nil
	},
}

func init() {
	AddCommand(versionCmd)
}
