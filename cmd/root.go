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
	configFile      string
	hostsConfigFile string
	verbose         bool
	cmdStdOut       bool
	logFile         string
	s3Endpoint      string

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
	rootCmd.PersistentFlags().StringVar(&logFile, "logFile", "", "log file to write to")
	rootCmd.PersistentFlags().BoolVar(&cmdStdOut, "cmdStdOut", false, "Pass to print command output to stdout")

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "", "config file to read from")
	rootCmd.PersistentFlags().StringVar(&hostsConfigFile, "hostsConfig", "", "yaml hosts file to read from")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Sets verbose level")
	rootCmd.PersistentFlags().StringVar(&s3Endpoint, "s3Endpoint", "", "Sets the S3 endpoint used for config file fetching. Overrides S3_ENDPOINT env variable.")
	rootCmd.AddCommand(backupCmd, execCmd, cronCmd, versionCmd, listCmd)
}

func parseS3Config() {
	if s3Endpoint != "" {
		os.Setenv("S3_ENDPOINT", s3Endpoint)
	}
}
