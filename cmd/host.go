package cmd

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	hostExecCommand = &cobra.Command{
		Use:   "host [--command=command1 --command=command2 ... | -c command1 -c command2 ...] [--hosts=host1 --hosts=hosts2 ... | -m host1 -m host2 ...] ",
		Short: "Runs command defined in config file on the hosts in order specified.",
		Long:  "Host executes specified commands on the hosts defined in config file.\nUse the --commands or -c flag to choose the commands.",
		Run:   Host,
	}
)

// Holds list of hosts to run commands on
var hostsList []string

// Holds command list to run
var cmdList []string

func init() {

	hostExecCommand.Flags().StringArrayVarP(&hostsList, "hosts", "m", nil, "Accepts space-separated names of hosts. Specify multiple times for multiple hosts.")
	hostExecCommand.Flags().StringArrayVarP(&cmdList, "command", "c", nil, "Accepts space-separated names of commands. Specify multiple times for multiple commands.")
	parseS3Config()

}

// cli input should be hosts and commands. Hosts are defined in config files.
// commands can be passed by the following mutually exclusive options:
//    1. as a list of commands defined in the config file
//    2. stdin (on command line) (TODO)

func Host(cmd *cobra.Command, args []string) {
	backyConfOpts := backy.NewOpts(cfgFile, backy.SetLogFile(logFile))
	backyConfOpts.InitConfig()

	backyConfOpts.ReadConfig()

	// check CLI input
	if hostsList == nil {
		logging.ExitWithMSG("error: hosts must be specified", 1, &backyConfOpts.Logger)
	}
	// host is only checked when we read the SSH File
	// so a check may not be needed here
	// but we can check if the host is in the config file
	for _, h := range hostsList {
		_, hostFound := backyConfOpts.Hosts[h]
		if !hostFound {
			// check if h exists in the config file
			hostFoundInConfig, s := backy.CheckIfHostHasHostName(h)
			if !hostFoundInConfig {
				logging.ExitWithMSG("host "+h+" not found", 1, &backyConfOpts.Logger)
			}
			// create host with hostname and host
			backyConfOpts.Hosts[h] = &backy.Host{Host: h, HostName: s}
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
