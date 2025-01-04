package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	hostExecCommand = &cobra.Command{
		Use:   "host [--commands=command1,command2, ... | -c command1,command2, ...] [--hosts=host1,hosts2, ... | -m host1,host2, ...] ",
		Short: "Runs command defined in config file on the hosts in order specified.",
		Long:  "Host executes specified commands on the hosts defined in config file.\nUse the --commands or -c flag to choose the commands.",
		Run:   Host,
	}
)

// Holds command list to run
var hostsList []string
var cmdList []string

func init() {

}

// cli input should be hosts and commands. Hosts are defined in config files.
// commands can be passed by the following mutually exclusive options:
//    1. as a list of commands defined in the config file
//    2. stdin (on command line) (TODO)

func Host(cmd *cobra.Command, args []string) {
	backyConfOpts := backy.NewOpts(cfgFile)
	backyConfOpts.InitConfig()

	backy.ReadConfig(backyConfOpts)

	// check CLI input
	if hostsList == nil {
		logging.ExitWithMSG("error: hosts must be specified", 1, &backyConfOpts.Logger)
	}
	// host is only checked when we read the SSH File
	// so a check may not be needed here
	for _, h := range hostsList {
		_, hostFound := backyConfOpts.Hosts[h]
		if !hostFound {
			logging.ExitWithMSG("host "+h+" not found", 1, &backyConfOpts.Logger)
		}
	}
	if cmdList == nil {
		logging.ExitWithMSG("error: commands must be specified", 1, &backyConfOpts.Logger)
	}
	for _, c := range cmdList {
		_, cmdFound := backyConfOpts.Cmds[c]
		if !cmdFound {
			logging.ExitWithMSG("cmd "+c+" not found", 1, &backyConfOpts.Logger)
		}
	}

	backyConfOpts.ExecCmdsSSH(cmdList, hostsList)
}
