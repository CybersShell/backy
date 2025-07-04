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

}

func execute(cmd *cobra.Command, args []string) {
	parseS3Config()

	if len(args) < 1 {
		logging.ExitWithMSG("Please provide a command to run. Pass --help to see options.", 1, nil)
	}

	opts := backy.NewConfigOptions(configFile,
		backy.AddCommands(args),
		backy.SetLogFile(logFile),
		backy.EnableCommandStdOut(cmdStdOut),
		backy.SetHostsConfigFile(hostsConfigFile))
	opts.InitConfig()
	opts.ParseConfigurationFile()
	opts.ExecuteCmds()
}
