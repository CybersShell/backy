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
		Short: "Lists commands, lists, or hosts defined in config file.",
		Long:  "Backup lists commands or groups defined in config file.\nUse the --lists or -l flag to list the specified lists. If not flag is not given, all lists will be executed.",
	}

	listCmds = &cobra.Command{
		Use:   "cmds [cmd1 cmd2 cmd3...]",
		Short: "Lists commands, lists, or hosts defined in config file.",
		Long:  "Backup lists commands or groups defined in config file.\nUse the --lists or -l flag to list the specified lists. If not flag is not given, all lists will be executed.",
		Run:   ListCmds,
	}
	listCmdLists = &cobra.Command{
		Use:   "lists [list1 list2 ...]",
		Short: "Lists commands, lists, or hosts defined in config file.",
		Long:  "Backup lists commands or groups defined in config file.\nUse the --lists or -l flag to list the specified lists. If not flag is not given, all lists will be executed.",
		Run:   ListCmdLists,
	}
)

var listsToList []string
var cmdsToList []string

func init() {
	listCmd.AddCommand(listCmds, listCmdLists)

}

func ListCmds(cmd *cobra.Command, args []string) {

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

	opts := backy.NewOpts(cfgFile, backy.SetLogFile(logFile))

	opts.InitConfig()
	opts.ReadConfig()

	for _, v := range cmdsToList {
		opts.ListCommand(v)
	}
}

func ListCmdLists(cmd *cobra.Command, args []string) {

	parseS3Config()

	if len(args) > 0 {
		listsToList = args
	} else {
		logging.ExitWithMSG("Error: lists subcommand needs lists", 1, nil)
	}

	opts := backy.NewOpts(cfgFile, backy.SetLogFile(logFile))

	opts.InitConfig()
	opts.ReadConfig()

	for _, v := range listsToList {
		opts.ListCommandList(v)
	}

}
