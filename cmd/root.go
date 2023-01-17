// root.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file to read from")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Sets verbose level")

	rootCmd.AddCommand(backupCmd)
}

func initConfig() {
	backyConfig := viper.New()
	if cfgFile != strings.TrimSpace("") {
		// Use config file from the flag.
		backyConfig.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configPath := path.Join(home, ".config", "backy")
		// Search config in config directory with name "backy" (without extension).
		backyConfig.AddConfigPath(configPath)
		backyConfig.SetConfigType("yaml")
		backyConfig.SetConfigName("backy")
	}

	backyConfig.AutomaticEnv()

	if err := backyConfig.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", backyConfig.ConfigFileUsed())
	}
}
