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

	return newBackupConfig
}
