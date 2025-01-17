// exec.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"

	"github.com/spf13/cobra"
)

var (
	execCmd = &cobra.Command{
		Use:   "exec command ...",
		Short: "Runs commands defined in config file in order given.",
		Long:  `Exec executes commands defined in config file in order given.`,
		Run:   execute,
	}
)

func init() {
	execCmd.AddCommand(hostExecCommand)

	hostExecCommand.Flags().StringSliceVarP(&hostsList, "hosts", "m", nil, "Accepts comma-separated names of hosts.")
	hostExecCommand.Flags().StringSliceVarP(&cmdList, "commands", "c", nil, "Accepts comma-separated names of commands.")

}

func execute(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		logging.ExitWithMSG("Please provide a command to run. Pass --help to see options.", 1, nil)
	}

	opts := backy.NewOpts(cfgFile, backy.AddCommands(args), backy.SetLogFile(logFile))
	opts.InitConfig()
	opts.ReadConfig()
	opts.ExecuteCmds()
}
