package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	godotenv "github.com/joho/godotenv"
	"github.com/metalagman/omnidist/cmd/omnidist/npm"
	"github.com/metalagman/omnidist/cmd/omnidist/uv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var omnidistRoot string
var initRootErr error

var rootCmd = &cobra.Command{
	Use:           "omnidist",
	Short:         "Omni-platform Binary Distribution Toolkit",
	Long:          `A repeatable way to build, package, and publish a Go CLI for npm and uv distributions.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initRootErr
	},
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
	cobra.OnInitialize(initOmnidistRoot, initDotEnv, initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .omnidist/omnidist.yaml)")
	rootCmd.PersistentFlags().StringVar(&omnidistRoot, "omnidist-root", "", "project root directory used as working directory")
	rootCmd.AddCommand(npm.Cmd)
	rootCmd.AddCommand(uv.Cmd)
}

func initOmnidistRoot() {
	initRootErr = nil
	root := strings.TrimSpace(omnidistRoot)
	if root == "" {
		return
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		initRootErr = fmt.Errorf("resolve --omnidist-root %q: %w", root, err)
		return
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		initRootErr = fmt.Errorf("stat --omnidist-root %q: %w", absRoot, err)
		return
	}
	if !info.IsDir() {
		initRootErr = fmt.Errorf("--omnidist-root %q is not a directory", absRoot)
		return
	}

	if err := os.Chdir(absRoot); err != nil {
		initRootErr = fmt.Errorf("chdir --omnidist-root %q: %w", absRoot, err)
		return
	}
	omnidistRoot = absRoot
}

func initDotEnv() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		cobra.CheckErr(fmt.Errorf(".env load: %w", err))
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
}
