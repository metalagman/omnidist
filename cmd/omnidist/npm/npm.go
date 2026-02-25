package npm

import "github.com/spf13/cobra"

// Cmd groups npm distribution subcommands.
var Cmd = &cobra.Command{
	Use:   "npm",
	Short: "NPM distribution commands",
}
