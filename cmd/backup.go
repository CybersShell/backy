// backup.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"

	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "backup [--lists=list1,list2,... | -l list1, list2,...]",
		Short: "Runs commands defined in config file.",
		Long:  "Backup executes commands defined in config file.\nUse the --lists or -l flag to execute the specified lists. If not flag is not given, all lists will be executed.",
		Run:   Backup,
	}
)

// Holds command list to run
var cmdLists []string

func init() {

	backupCmd.Flags().StringSliceVarP(&cmdLists, "lists", "l", nil, "Accepts comma-separated names of command lists to execute.")

}

func Backup(cmd *cobra.Command, args []string) {
	backyConfOpts := backy.NewOpts(cfgFile, backy.AddCommandLists(cmdLists))
	backyConfOpts.InitConfig()

	backy.ReadConfig(backyConfOpts)

	backyConfOpts.RunListConfig("")
	for _, host := range backyConfOpts.Hosts {
		if host.SshClient != nil {
			host.SshClient.Close()
		}
	}
}
