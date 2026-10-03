package uv

import "github.com/spf13/cobra"

// Cmd groups PyPI distribution subcommands with the legacy uv alias.
var Cmd = &cobra.Command{
	Use:     "pypi",
	Aliases: []string{"uv"},
	Short:   "PyPI distribution commands",
}
