package backy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
)

// Host defines a host to which to connect
// If not provided, the values will be looked up in the default ssh config files
type Host struct {
	ConfigFilePath     string
	Empty              bool
	Host               string
	HostName           []string
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

func (command Command) RunCmd() logging.Logging {

	var stdoutBuf, stderrBuf bytes.Buffer
	var err error
	var cmdArgs string
	for _, v := range command.CmdArgs {
		cmdArgs += v
	}

	var remoteHost = &command.RemoteHost
	fmt.Printf("\n\nRunning command: " + command.Cmd + " " + cmdArgs + " on host " + command.RemoteHost.Host + "...\n\n")
	if command.Remote {

		remoteHost.Port = 22
		remoteHost.Host = command.RemoteHost.Host

		sshClient, err := remoteHost.ConnectToSSHHost()
		if err != nil {
			panic(fmt.Errorf("ssh dial: %w", err))
		}
		defer sshClient.Close()
		s, err := sshClient.NewSession()
		if err != nil {
			panic(fmt.Errorf("new ssh session: %w", err))
		}
		defer s.Close()

		cmd := command.Cmd
		for _, a := range command.CmdArgs {
			cmd += " " + a
		}

		s.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		s.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = s.Run(cmd)
		if err != nil {
			return logging.Logging{
				Output: stdoutBuf.String(),
				Err:    fmt.Errorf("error running " + cmd + ": " + stderrBuf.String()),
			}
		}
		// fmt.Printf("Output: %s\n", string(output))
	} else {
		// shell := "/bin/bash"
		localCMD := exec.Command(command.Cmd, command.CmdArgs...)
		localCMD.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		localCMD.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = localCMD.Run()

		if err != nil {
			return logging.Logging{
				Output: stdoutBuf.String(),
				Err:    fmt.Errorf(stderrBuf.String()),
			}
		}
	}
	return logging.Logging{
		Output: stdoutBuf.String(),
		Err:    nil,
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
