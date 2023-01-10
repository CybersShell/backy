package cmd

import (
	"fmt"
	"os"
	"path"

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
		Long: `Backy is a command-line application useful
		for configuring backups, or any commands run in sequence.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file to read from")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Sets verbose level")

}

func initConfig() {
	backyConfig := viper.New()
	if cfgFile != "" {
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
		fmt.Println("Using config file:", backyConfig.ConfigFileUsed())
	}
}
