package cmd

// import (
// 	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"

// 	"github.com/spf13/cobra"
// )

// var (
// 	configCmd = &cobra.Command{
// 		Use:   "config list ...",
// 		Short: "Runs commands defined in config file.",
// 		Long:  `Cron executes commands at the time defined in config file.`,
// 		Run:   config,
// 	}

// 	cmds  []string
// 	lists []string
// )

// func config(cmd *cobra.Command, args []string) {

// 	opts := backy.NewConfigOptions(configFile, backy.cronEnabled())
// 	opts.InitConfig()

// }

// func init() {

// 	configCmd.PersistentFlags().StringArrayVarP(&cmds, "cmds", "c", nil, "Accepts comma-seperated list of commands to list")
// }
