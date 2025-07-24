package cmd

import (
	"maps"
	"slices"

	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	hostsExecCommand = &cobra.Command{
		Use:   "hosts [--command=command1 --command=command2 ... | -c command1 -c command2 ...]",
		Short: "Runs command defined in config file on the hosts in order specified.",
		Long:  "Hosts executes specified commands on all the hosts defined in config file.\nUse the --commands or -c flag to choose the commands.",
		Run:   Hosts,
	}

	hostsListExecCommand = &cobra.Command{
		Use:   "list list1 list2 ...",
		Short: "Runs lists in order specified defined in config file on all hosts.",
		Long:  "Lists executes specified lists on all the hosts defined in hosts config.\nPass the names of lists as arguments after command.",
		Run:   HostsList,
	}
)

func init() {
	hostsExecCommand.AddCommand(hostsListExecCommand)
	parseS3Config()

}

// cli input should be hosts and commands. Hosts are defined in config files.
// commands can be passed by the following mutually exclusive options:
//    1. as a list of commands defined in the config file
//    2. stdin (on command line) (TODO)

func Hosts(cmd *cobra.Command, args []string) {
	backyConfOpts := backy.NewConfigOptions(configFile,
		backy.SetLogFile(logFile),
		backy.EnableCommandStdOut(cmdStdOut),
		backy.SetHostsConfigFile(hostsConfigFile))
	backyConfOpts.InitConfig()

	backyConfOpts.ParseConfigurationFile()

	for _, h := range backyConfOpts.Hosts {

		hostsList = append(hostsList, h.Host)
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

	backyConfOpts.ExecCmdsOnHosts(cmdList, hostsList)
}

func HostsList(cmd *cobra.Command, args []string) {
	backyConfOpts := backy.NewConfigOptions(configFile,
		backy.SetLogFile(logFile),
		backy.EnableCommandStdOut(cmdStdOut),
		backy.SetHostsConfigFile(hostsConfigFile))
	backyConfOpts.InitConfig()

	backyConfOpts.ParseConfigurationFile()

	if len(args) == 0 {
		logging.ExitWithMSG("error: no lists specified", 1, &backyConfOpts.Logger)
	}

	for _, l := range args {
		_, listFound := backyConfOpts.CmdConfigLists[l]
		if !listFound {
			logging.ExitWithMSG("list "+l+" not found", 1, &backyConfOpts.Logger)
		}
	}

	maps.DeleteFunc(backyConfOpts.CmdConfigLists, func(k string, v *backy.CmdList) bool {
		return !slices.Contains(args, k)
	})

	backyConfOpts.ExecuteListOnHosts(args)
}
