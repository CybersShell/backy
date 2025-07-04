// backup.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"

	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list [command]",
		Short: "List commands, lists, or hosts defined in config file.",
		Long:  "List commands, lists, or hosts defined in config file",
	}

	listCmds = &cobra.Command{
		Use:   "cmds [cmd1 cmd2 cmd3...]",
		Short: "List commands defined in config file.",
		Long:  "List commands defined in config file",
		Run:   ListCommands,
	}
	listCmdLists = &cobra.Command{
		Use:   "lists [list1 list2 ...]",
		Short: "List lists defined in config file.",
		Long:  "List lists defined in config file",
		Run:   ListCommandLists,
	}
)

var listsToList []string
var cmdsToList []string

func init() {
	listCmd.AddCommand(listCmds, listCmdLists)

}

func ListCommands(cmd *cobra.Command, args []string) {

	// setup based on whats passed in:
	//   - cmds
	//   - lists
	//   - if none, list all commands
	if len(args) > 0 {
		cmdsToList = args
	} else {
		logging.ExitWithMSG("Error: list cmds subcommand needs commands to list", 1, nil)
	}

	parseS3Config()

	opts := backy.NewConfigOptions(configFile,
		backy.SetLogFile(logFile),
		backy.SetHostsConfigFile(hostsConfigFile))

	opts.InitConfig()
	opts.ParseConfigurationFile()

	for _, v := range cmdsToList {
		opts.ListCommand(v)
	}
}

func ListCommandLists(cmd *cobra.Command, args []string) {

	parseS3Config()

	if len(args) > 0 {
		listsToList = args
	} else {
		logging.ExitWithMSG("Error: lists subcommand needs lists", 1, nil)
	}

	opts := backy.NewConfigOptions(configFile,
		backy.SetLogFile(logFile),
		backy.SetHostsConfigFile(hostsConfigFile))

	opts.InitConfig()
	opts.ParseConfigurationFile()

	for _, v := range listsToList {
		opts.ListCommandList(v)
	}

}
