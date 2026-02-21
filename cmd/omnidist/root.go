package main

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/metalagman/omnidist/cmd/omnidist/npm"
	"github.com/metalagman/omnidist/cmd/omnidist/uv"
	godotenv "github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "omnidist",
	Short: "Omni-platform Binary Distribution Toolkit",
	Long:  `A repeatable way to build, package, and publish a Go CLI for npm and uv distributions.`,
}

func Execute() error {
	return rootCmd.Execute()
}

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
