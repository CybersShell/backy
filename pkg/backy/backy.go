package backy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Host defines a host to which to connect
// If not provided, the values will be looked up in the default ssh config files
type Host struct {
	ConfigFilePath     string
	Empty              bool
	Host               string
	HostName           string
	Port               uint16
	PrivateKeyPath     string
	PrivateKeyPassword string
	User               string
}

type Command struct {
	Remote bool

	// command to run
	Cmd string

	// host on which to run cmd
	Host string

	/*
		Shell specifies which shell to run the command in, if any
		Not applicable when host is defined
	*/
	Shell string

	RemoteHost Host

	// cmdArgs is an array that holds the arguments to cmd
	CmdArgs []string
}

type BackyConfigFile struct {
	/*
		Cmds holds the commands for a list
		key is the name of the command
	*/
	Cmds map[string]Command

	/*
		CmdLists holds the lists of commands to be run in order
		key is the command list name
	*/
	CmdLists map[string][]string

	/*
		Hosts holds the Host config
		key is the host
	*/
	Hosts map[string]Host
}

// BackupConfig is a configuration struct that is used to define backups
type BackupConfig struct {
	Name       string
	BackupType string
	ConfigPath string

	Cmd Command
}

/*
* Runs a backup configuration
 */

func (command Command) RunCmd() {

	var cmdArgsStr string
	for _, v := range command.CmdArgs {
		cmdArgsStr += fmt.Sprintf(" %s", v)
	}

	fmt.Printf("\n\nRunning command: " + command.Cmd + " " + cmdArgsStr + " on host " + command.Host + "...\n\n")
	if command.Host != "" {

		command.RemoteHost.Host = command.Host
		command.RemoteHost.Port = 22
		sshc, err := command.RemoteHost.ConnectToSSHHost()
		if err != nil {
			panic(fmt.Errorf("ssh dial: %w", err))
		}
		defer sshc.Close()
		s, err := sshc.NewSession()
		if err != nil {
			panic(fmt.Errorf("new ssh session: %w", err))
		}
		defer s.Close()

		cmd := command.Cmd
		for _, a := range command.CmdArgs {
			cmd += " " + a
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		s.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		s.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = s.Run(cmd)
		log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
		log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()

		if err != nil {
			panic(fmt.Errorf("error when running cmd " + cmd + "\n Error: " + err.Error()))
		}
	} else {
		// shell := "/bin/bash"
		var err error
		if command.Shell != "" {
			cmdArgsStr = fmt.Sprintf("%s %s", command.Cmd, cmdArgsStr)
			localCMD := exec.Command(command.Shell, "-c", cmdArgsStr)

			var stdoutBuf, stderrBuf bytes.Buffer
			localCMD.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
			localCMD.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
			err = localCMD.Run()
			log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
			log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()

			if err != nil {
				panic(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err))
			}
			return
		}
		localCMD := exec.Command(command.Cmd, command.CmdArgs...)

		var stdoutBuf, stderrBuf bytes.Buffer
		localCMD.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		localCMD.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = localCMD.Run()
		log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
		log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()
		if err != nil {
			panic(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err))
		}
	}
}

func (config *BackyConfigFile) RunBackyConfig() {
	for _, list := range config.CmdLists {
		for _, cmd := range list {
			cmdToRun := config.Cmds[cmd]
			cmdToRun.RunCmd()
		}
	}
}

func New() BackupConfig {
	return BackupConfig{}
}

// NewConfig initializes new config that holds information
// from the config file
func NewConfig() *BackyConfigFile {
	return &BackyConfigFile{
		Cmds:     make(map[string]Command),
		CmdLists: make(map[string][]string),
		Hosts:    make(map[string]Host),
	}
}

func ReadAndParseConfigFile() *BackyConfigFile {

	backyConfigFile := NewConfig()

	backyViper := viper.New()
	backyViper.SetConfigName("backy")               // name of config file (without extension)
	backyViper.SetConfigType("yaml")                // REQUIRED if the config file does not have the extension in the name
	backyViper.AddConfigPath(".")                   // optionally look for config in the working directory
	backyViper.AddConfigPath("$HOME/.config/backy") // call multiple times to add many search paths
	err := backyViper.ReadInConfig()                // Find and read the config file
	if err != nil {                                 // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	commandsMap := backyViper.GetStringMapString("commands")
	var cmdNames []string
	for k := range commandsMap {
		cmdNames = append(cmdNames, k)
	}
	hostConfigsMap := make(map[string]*viper.Viper)

	for _, cmdName := range cmdNames {
		var backupCmdStruct Command
		println(cmdName)
		subCmd := backyViper.Sub(getNestedConfig("commands", cmdName))

		hostSet := subCmd.IsSet("host")
		host := subCmd.GetString("host")

		cmdSet := subCmd.IsSet("cmd")
		cmd := subCmd.GetString("cmd")
		cmdArgsSet := subCmd.IsSet("cmdargs")
		cmdArgs := subCmd.GetStringSlice("cmdargs")
		shellSet := subCmd.IsSet("shell")
		shell := subCmd.GetString("shell")

		if hostSet {
			println("Host:")
			println(host)
			backupCmdStruct.Host = host
			if backyViper.IsSet(getNestedConfig("hosts", host)) {
				hostconfig := backyViper.Sub(getNestedConfig("hosts", host))
				hostConfigsMap[host] = hostconfig
			}
		} else {
			println("Host is not set")
		}
		if cmdSet {
			println("Cmd:")
			println(cmd)
			backupCmdStruct.Cmd = cmd
		} else {
			println("Cmd is not set")
		}
		if shellSet {
			println("Shell:")
			println(shell)
			backupCmdStruct.Shell = shell
		} else {
			println("Shell is not set")
		}
		if cmdArgsSet {
			println("CmdArgs:")
			for _, arg := range cmdArgs {
				println(arg)
			}
			backupCmdStruct.CmdArgs = cmdArgs
		} else {
			println("CmdArgs are not set")
		}
		backyConfigFile.Cmds[cmdName] = backupCmdStruct

	}

	cmdListCfg := backyViper.GetStringMapStringSlice("cmd-lists")
	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range cmdListCfg {
		println("Cmd list: ", cmdListName)
		for _, cmdInList := range cmdList {
			println("Command in list: " + cmdInList)
			_, cmdNameFound := backyConfigFile.Cmds[cmdInList]
			if !backyViper.IsSet(getNestedConfig("commands", cmdInList)) && !cmdNameFound {
				cmdNotFoundStr := fmt.Sprintf("command definition %s is not in config file\n", cmdInList)
				cmdNotFoundErr := errors.New(cmdNotFoundStr)
				cmdNotFoundSliceErr = append(cmdNotFoundSliceErr, cmdNotFoundErr)
			} else {
				backyConfigFile.CmdLists[cmdListName] = append(backyConfigFile.CmdLists[cmdListName], cmdInList)
			}
		}
	}
	for _, err := range cmdNotFoundSliceErr {
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	return backyConfigFile
}

func getNestedConfig(nestedConfig, key string) string {
	return fmt.Sprintf("%s.%s", nestedConfig, key)
}

func getNestedSSHConfig(key string) string {
	return fmt.Sprintf("hosts.%s.config", key)
}
