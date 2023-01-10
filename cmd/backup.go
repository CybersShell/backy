package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"

	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "backup [--commands==list1,list2]",
		Short: "Runs commands defined in config file.",
		Long: `Backup executes commands defined in config file, 
		use the -cmds flag to execute the specified commands.`,
	}
)
var CmdList *[]string

func init() {
	cobra.OnInitialize(initConfig)

	backupCmd.Flags().StringSliceVarP(CmdList, "commands", "cmds", nil, "Accepts a comma-separated list of command lists to execute.")
}

func backup() {
	backyConfig := backy.NewOpts(cfgFile)
	backyConfig.GetConfig()
}
