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
	Remote     bool
	RemoteHost Host
	Cmd        string
	Args       []string
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

func (command Command) runCmd() logging.Logging {

	var stdoutBuf, stderrBuf bytes.Buffer
	var err error
	var cmdArgs string
	for _, v := range command.Args {
		cmdArgs += v
	}

	var remoteHost = &command.RemoteHost
	fmt.Printf("\n\nRunning command: " + command.Cmd + " " + cmdArgs + " on host " + command.RemoteHost.Host + "...\n\n")
	if command.Remote {

		remoteHost.Port = 22
		remoteHost.Host = command.RemoteHost.Host

		sshc, err := remoteHost.ConnectToSSHHost()
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
		for _, a := range command.Args {
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
		localCMD := exec.Command(command.Cmd, command.Args...)
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
