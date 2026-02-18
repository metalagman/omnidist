package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "omnidist",
	Short: "Omni-platform Binary Distribution Toolkit",
	Long:  `A repeatable way to build, package, and publish a Go CLI as an npm-installable tool.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
