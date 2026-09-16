package gem

import "github.com/spf13/cobra"

// Cmd groups gem distribution subcommands.
var Cmd = &cobra.Command{
	Use:   "gem",
	Short: "RubyGems distribution commands",
}
