// backup.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"

	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list [--list=list1,list2,... | -l list1, list2,...] [ -cmd cmd1 cmd2 cmd3...]",
		Short: "Lists commands, lists, or hosts defined in config file.",
		Long:  "Backup lists commands or groups defined in config file.\nUse the --lists or -l flag to list the specified lists. If not flag is not given, all lists will be executed.",
		Run:   List,
	}
)

var listsToList []string
var cmdsToList []string

func init() {

	listCmd.Flags().StringSliceVarP(&listsToList, "lists", "l", nil, "Accepts comma-separated names of command lists to list.")
	listCmd.Flags().StringSliceVarP(&cmdsToList, "cmds", "c", nil, "Accepts comma-separated names of commands to list.")

}

func List(cmd *cobra.Command, args []string) {

	// settup based on whats passed in:
	//   - cmds
	//   - lists
	//   - if none, list all commands
	if cmdLists != nil {

	}

	opts := backy.NewOpts(cfgFile)

	opts.InitConfig()

	opts = backy.ReadConfig(opts)

	opts.ListCommand("rm-sn-db")
}
