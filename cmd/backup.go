package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/notifications"

	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "backup [--commands==list1,list2]",
		Short: "Runs commands defined in config file.",
		Long: `Backup executes commands defined in config file, 
		use the -cmds flag to execute the specified commands.`,
		Run: Backup,
	}
)
var CmdList []string

func init() {
	// cobra.OnInitialize(initConfig)

	backupCmd.Flags().StringSliceVar(&CmdList, "cmds", nil, "Accepts a comma-separated list of command lists to execute.")

}

func Backup(cmd *cobra.Command, args []string) {
	config := backy.ReadAndParseConfigFile(cfgFile)
	notifications.SetupNotify(*config)
	config.RunBackyConfig()
}
