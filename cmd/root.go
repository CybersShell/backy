// root.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile string
	verbose bool

	rootCmd = &cobra.Command{
		Use:   "backy",
		Short: "An easy-to-configure backup tool.",
		Long:  `Backy is a command-line application useful for configuring backups, or any commands run in sequence.`,
	}
)

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file to read from")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Sets verbose level")

	rootCmd.AddCommand(backupCmd, execCmd, cronCmd, versionCmd)
}
