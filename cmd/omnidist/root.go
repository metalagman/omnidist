package main

import (
	"errors"
	"fmt"
	"io/fs"

	godotenv "github.com/joho/godotenv"
	"github.com/metalagman/omnidist/cmd/omnidist/npm"
	"github.com/metalagman/omnidist/cmd/omnidist/uv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "omnidist",
	Short:         "Omni-platform Binary Distribution Toolkit",
	Long:          `A repeatable way to build, package, and publish a Go CLI for npm and uv distributions.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root omnidist command tree.
func Execute() error {
	return rootCmd.Execute()
}

// AddCommand registers a top-level subcommand on the root command.
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

func init() {
	cobra.OnInitialize(initDotEnv)
	rootCmd.AddCommand(npm.Cmd)
	rootCmd.AddCommand(uv.Cmd)
}

func initDotEnv() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		cobra.CheckErr(fmt.Errorf(".env load: %w", err))
	}
}
