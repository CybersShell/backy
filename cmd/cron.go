package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"

	"github.com/spf13/cobra"
)

var (
	cronCmd = &cobra.Command{
		Use:   "cron [flags]",
		Short: "Starts a scheduler that runs lists defined in config file.",
		Long:  `Cron starts a scheduler that executes command lists at the time defined in config file.`,
		Run:   cron,
	}
)

func cron(cmd *cobra.Command, args []string) {
	parseS3Config()

	opts := backy.NewConfigOptions(configFile,
		backy.EnableCron(),
		backy.SetLogFile(logFile),
		backy.EnableCommandStdOut(cmdStdOut),
		backy.SetHostsConfigFile(hostsConfigFile))

	opts.InitConfig()
	opts.ParseConfigurationFile()

	opts.Cron()
}
