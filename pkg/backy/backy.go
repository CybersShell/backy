package backy

import (
	"os/exec"

	"github.com/melbahja/goph"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/spf13/viper"
)

type Host struct {
	Empty              bool
	Host               string
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

func ReadConfig(backup string, config viper.Viper) (*viper.Viper, error) {
	err := viper.ReadInConfig()
	if err != nil {
		return &viper.Viper{}, err
	}

	conf := config.Sub("backup." + backup)

	backupConfig := config.Unmarshal(&conf)
	if backupConfig == nil {
		return &viper.Viper{}, backupConfig
	}

	return conf, nil
}

// pass Name-level (i.e. "backups."+configName) to function
func CreateConfig(backup BackupConfig) BackupConfig {
	newBackupConfig := BackupConfig{
		Name:       backup.Name,
		BackupType: backup.BackupType,

		DstDir: backup.DstDir,
		SrcDir: backup.SrcDir,

		ConfigPath: backup.ConfigPath,
	}

	if !backup.Cmds.Before.Empty {
		newBackupConfig.Cmds.Before.Cmd = backup.Cmds.Before.Cmd
		newBackupConfig.Cmds.After.Args = backup.Cmds.Before.Args
		if backup.Cmds.Before.Remote {
			newBackupConfig.Cmds.Before.RemoteHost.Host = backup.Cmds.Before.RemoteHost.Host
			newBackupConfig.Cmds.Before.RemoteHost.Port = backup.Cmds.Before.RemoteHost.Port
			newBackupConfig.Cmds.Before.RemoteHost.PrivateKeyPath = backup.Cmds.Before.RemoteHost.PrivateKeyPath
		} else {
			newBackupConfig.Cmds.Before.RemoteHost.Empty = true
		}
	}
	if !backup.Cmds.Backup.Empty {
		newBackupConfig.Cmds.Backup.Cmd = backup.Cmds.Backup.Cmd
		newBackupConfig.Cmds.Backup.Args = backup.Cmds.Backup.Args
		if backup.Cmds.Backup.Remote {
			newBackupConfig.Cmds.Backup.RemoteHost.Host = backup.Cmds.Backup.RemoteHost.Host
			newBackupConfig.Cmds.Backup.RemoteHost.Port = backup.Cmds.Backup.RemoteHost.Port
			newBackupConfig.Cmds.Backup.RemoteHost.PrivateKeyPath = backup.Cmds.Backup.RemoteHost.PrivateKeyPath
		} else {
			newBackupConfig.Cmds.Backup.RemoteHost.Empty = true
		}
	}
	if !backup.Cmds.After.Empty {
		newBackupConfig.Cmds.After.Cmd = backup.Cmds.After.Cmd
		newBackupConfig.Cmds.After.Args = backup.Cmds.After.Args
		if backup.Cmds.After.Remote {
			newBackupConfig.Cmds.After.RemoteHost.Host = backup.Cmds.After.RemoteHost.Host
			newBackupConfig.Cmds.After.RemoteHost.Port = backup.Cmds.After.RemoteHost.Port
			newBackupConfig.Cmds.After.RemoteHost.PrivateKeyPath = backup.Cmds.After.RemoteHost.PrivateKeyPath
		} else {
			newBackupConfig.Cmds.Before.RemoteHost.Empty = true
		}
	}

	return backup
}

// writes config to file
func WriteConfig(config viper.Viper, backup BackupConfig) error {
	configName := "backup." + backup.Name
	config.Set(configName+".BackupType", backup.BackupType)
	if !backup.Cmds.After.Empty {
		config.Set(configName+".Cmds.After.Cmd", backup.Cmds.After.Cmd)
		config.Set(configName+".Cmds.After.Args", backup.Cmds.After.Args)
		if backup.Cmds.Before.Remote {
			config.Set(configName+".Cmds.After.RemoteHost.Host", backup.Cmds.After.RemoteHost.Host)
		}
	}

	config.Set(configName+"..Cmds.backup.Cmd", backup.Cmds.Backup.Cmd)
	if !backup.Cmds.Before.Empty {
		config.Set(configName+"Cmds.Before.Cmd", backup.Cmds.Before.Cmd)
		config.Set(configName+"Cmds.Before.Args", backup.Cmds.Before.Args)
	}
	err := config.WriteConfig()
	if err != nil {
		return err
	}
	return nil
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
