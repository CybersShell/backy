package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/notification"

	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "backup [--lists==list1,list2]",
		Short: "Runs commands defined in config file.",
		Long: `Backup executes commands defined in config file.
		Use the --lists flag to execute the specified commands.`,
		Run: Backup,
	}
)

// Holds command list to run
var cmdList []string

func init() {

	backupCmd.Flags().StringSliceVarP(&cmdList, "lists", "l", nil, "Accepts a comma-separated names of command lists to execute.")

}

func Backup(cmd *cobra.Command, args []string) {

	config := backy.ReadAndParseConfigFile(cfgFile, cmdList)
	notification.SetupNotify(*config)
	config.RunBackyConfig()
}
