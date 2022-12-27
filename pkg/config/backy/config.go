package config

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
	"github.com/spf13/viper"
)

func ReadConfig(Config *viper.Viper) (*viper.Viper, error) {

	backyViper := viper.New()

	// Check for existing config, if none exists, return new one
	if Config == nil {

		backyViper.AddConfigPath(".")
		// name of config file (without extension)
		backyViper.SetConfigName("backy")
		// REQUIRED if the config file does not have the extension in the name
		backyViper.SetConfigType("yaml")

		if err := backyViper.ReadInConfig(); err != nil {
			if configFileNotFound, ok := err.(viper.ConfigFileNotFoundError); ok {
				return nil, configFileNotFound
				// Config file not found; ignore error if desired
			} else {
				// Config file was found but another error was produced
			}
		}
	} else {
		// Config exists, try to read config file
		if err := Config.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {

				backyViper.AddConfigPath(".")
				// name of config file (without extension)
				backyViper.SetConfigName("backy")
				// REQUIRED if the config file does not have the extension in the name
				backyViper.SetConfigType("yaml")

				if err := backyViper.ReadInConfig(); err != nil {
					if configFileNotFound, ok := err.(viper.ConfigFileNotFoundError); ok {
						return nil, configFileNotFound
					} else {
						// Config file was found but another error was produced
					}
				}

			} else {
				// Config file was found but another error was produced
			}
		}
	}

	return backyViper, nil
}

func unmarshallConfig(backup string, config *viper.Viper) (*viper.Viper, error) {
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	yamlConfigPath := "backup." + backup
	conf := config.Sub(yamlConfigPath)

	backupConfig := config.Unmarshal(&conf)
	if backupConfig == nil {
		return nil, backupConfig
	}

	return conf, nil
}

// CreateConfig creates a configuration
// pass Name-level (i.e. "backups."+configName) to function
func CreateConfig(backup backy.BackupConfig) backy.BackupConfig {
	newBackupConfig := backy.BackupConfig{
		Name:       backup.Name,
		BackupType: backup.BackupType,

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
