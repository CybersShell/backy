package backy

import (
	"os/exec"

	"github.com/melbahja/goph"

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

type BackupConfig struct {
	Name       string
	BackupType string
	ConfigPath string

	Cmds Commands

	DstDir string
	SrcDir string
}

/*
* Runs a backup configuration
 */
func Run(backup BackupConfig) logging.Logging {

	beforeConfig := backup.Cmds.Before
	beforeOutput := runCmd(beforeConfig)
	if beforeOutput.Err != nil {
		return logging.Logging{
			Output: beforeOutput.Output,
			Err:    beforeOutput.Err,
		}
	}
	backupConfig := backup.Cmds.Backup
	backupOutput := runCmd(backupConfig)
	if backupOutput.Err != nil {
		return logging.Logging{
			Output: beforeOutput.Output,
			Err:    beforeOutput.Err,
		}
	}
	afterConfig := backup.Cmds.After
	afterOutput := runCmd(afterConfig)
	if afterOutput.Err != nil {
		return logging.Logging{
			Output: beforeOutput.Output,
			Err:    beforeOutput.Err,
		}
	}
	return logging.Logging{
		Output: afterOutput.Output,
		Err:    nil,
	}
}

func runCmd(cmd Command) logging.Logging {
	if !cmd.Empty {
		if cmd.Remote {
			// Start new ssh connection with private key.
			auth, err := goph.Key(cmd.RemoteHost.PrivateKeyPath, cmd.RemoteHost.PrivateKeyPassword)
			if err != nil {
				return logging.Logging{
					Output: err.Error(),
					Err:    err,
				}
			}

			client, err := goph.New(cmd.RemoteHost.User, cmd.RemoteHost.Host, auth)
			if err != nil {
				return logging.Logging{
					Output: err.Error(),
					Err:    err,
				}
			}

			// Defer closing the network connection.
			defer client.Close()

			command := cmd.Cmd
			for _, v := range cmd.Args {
				command += v
			}

			// Execute your command.
			out, err := client.Run(command)
			if err != nil {
				return logging.Logging{
					Output: string(out),
					Err:    err,
				}
			}
		}
		cmdOut := exec.Command(cmd.Cmd, cmd.Args...)
		output, err := cmdOut.Output()
		if err != nil {
			return logging.Logging{
				Output: string(output),
				Err:    err,
			}
		}
	}
	return logging.Logging{
		Output: "",
		Err:    nil,
	}
}
