package backy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
)

type Host struct {
	Empty              bool
	Host               string
	UseSSHAgent        bool
	HostName           string
	Port               uint16
	PrivateKeyPath     string
	PrivateKeyPassword string
	User               string
}

type Command struct {
	Empty      bool
	Remote     bool
	RemoteHost Host
	Cmd        string
	Args       []string
}
type Commands struct {
	Before Command
	Backup Command
	After  Command
}

// BackupConfig is a configuration struct that is used to define backups
type BackupConfig struct {
	Name       string
	BackupType string
	ConfigPath string

	Cmds Commands
}

/*
* Runs a backup configuration
 */
func (backup BackupConfig) Run() logging.Logging {

	beforeConfig := backup.Cmds.Before
	beforeOutput := beforeConfig.runCmd()
	if beforeOutput.Err != nil {
		return logging.Logging{
			Output: beforeOutput.Output,
			Err:    beforeOutput.Err,
		}
	}
	backupConfig := backup.Cmds.Backup
	backupOutput := backupConfig.runCmd()
	if backupOutput.Err != nil {
		return logging.Logging{
			Output: beforeOutput.Output,
			Err:    beforeOutput.Err,
		}
	}
	afterConfig := backup.Cmds.After
	afterOutput := afterConfig.runCmd()
	if afterOutput.Err != nil {
		return afterOutput
	}
	return logging.Logging{
		Output: afterOutput.Output,
		Err:    nil,
	}
}

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

		sshc, err := remoteHost.connectToSSHHost()
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
