// backup.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"

	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list [command]",
		Short: "List commands, lists, or hosts defined in config file.",
		Long:  "List commands, lists, or hosts defined in config file. The subcommands take zero or more arguments to print specific commands or lists",
	}

	listCmds = &cobra.Command{
		Use:   "cmds [cmd1 cmd2 cmd3...]",
		Short: "Prints commands defined in config file.",
		Long:  "Prints commands defined in config file. Pass no arguments to print all commands",
		Run:   ListCommands,
	}
	listCmdLists = &cobra.Command{
		Use:   "lists [list1 list2 ...]",
		Short: "Prints lists defined in config file.",
		Long:  "Prints lists defined in config file. Pass no arguments to print all lists",
		Run:   ListCommandLists,
	}
)

func init() {
	listCmd.AddCommand(listCmds, listCmdLists)
	parseS3Config()

}

func ListCommands(cmd *cobra.Command, args []string) {

	// setup based on whats passed in:
	//   - cmds
	//   - lists
	//   - if none, list all commands

	opts := backy.NewConfigOptions(configFile,
		backy.SetLogFile(logFile),
		backy.SetHostsConfigFile(hostsConfigFile))

	opts.InitConfig()
	opts.ParseConfigurationFile()

	if len(args) > 0 {
		for _, v := range args {
			opts.ListCommand(v)
		}

		os.Exit(0)
	}

	if len(opts.Cmds) == 0 {
		fmt.Println("No commands defined in config file")
		os.Exit(1)
	}
	for c := range opts.Cmds {
		println()
		println()
		println("---------------------------------------------------------------------------------")
		opts.ListCommand(c)
	}
}

func ListCommandLists(cmd *cobra.Command, args []string) {

	opts := backy.NewConfigOptions(configFile,
		backy.SetLogFile(logFile),
		backy.SetHostsConfigFile(hostsConfigFile))

	opts.InitConfig()
	opts.ParseConfigurationFile()

	if len(args) > 0 {
		for _, v := range args {
			opts.ListCommandList(v)
		}
		os.Exit(0)
	}

	if len(opts.CmdConfigLists) == 0 {
		fmt.Println("No command lists defined in config file")
		os.Exit(1)
	}
	for c := range opts.CmdConfigLists {
		println()
		println()
		println("---------------------------------------------------------------------------------")
		opts.ListCommandList(c)
	}

}
