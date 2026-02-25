package uv

import "github.com/spf13/cobra"

// Cmd groups uv distribution subcommands.
var Cmd = &cobra.Command{
	Use:   "uv",
	Short: "uv distribution commands",
}
