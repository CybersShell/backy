// types.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0
package backy

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Host defines a host to which to connect.
// If not provided, the values will be looked up in the default ssh config files
type Host struct {
	ConfigFilePath     string `yaml:"config-file-path,omitempty"`
	UseConfigFile      bool
	Empty              bool
	Host               string
	HostName           string
	Port               uint16
	PrivateKeyPath     string
	PrivateKeyPassword string
	User               string
}

type Command struct {
	// Remote bool `yaml:"remote,omitempty"`

	Output BackyCommandOutput `yaml:"-"`

	// command to run
	Cmd string `yaml:"cmd"`

	// host on which to run cmd
	Host *string `yaml:"host,omitempty"`

	/*
		Shell specifies which shell to run the command in, if any.
		Not applicable when host is defined.
	*/
	Shell string `yaml:"shell,omitempty"`

	RemoteHost Host `yaml:"-"`

	// cmdArgs is an array that holds the arguments to cmd
	CmdArgs []string `yaml:"cmdArgs,omitempty"`

	/*
		Dir specifies a directory in which to run the command.
		Ignored if Host is set.
	*/
	Dir *string `yaml:"dir,omitempty"`

	// Env points to a file containing env variables to be used with the command
	Env string `yaml:"env,omitempty"`

	// Environment holds env variables to be used with the command
	Environment []string `yaml:"environment,omitempty"`
}

type CmdConfig struct {
	Order               []string `yaml:"order,omitempty"`
	Notifications       []string `yaml:"notifications,omitempty"`
	NotificationsConfig map[string]*NotificationsConfig
}

type BackyConfigFile struct {
	/*
		Cmds holds the commands for a list.
		Key is the name of the command,
	*/
	Cmds map[string]Command `yaml:"commands"`

	/*
		CmdLConfigists holds the lists of commands to be run in order.
		Key is the command list name.
	*/
	CmdConfigLists map[string]*CmdConfig `yaml:"cmd-configs"`

	/*
		Hosts holds the Host config.
		key is the host.
	*/
	Hosts map[string]Host `yaml:"hosts"`

	/*
		Notifications holds the config for different notifications.
	*/
	Notifications map[string]*NotificationsConfig

	Logger zerolog.Logger
}

type BackyConfigOpts struct {
	// Holds config file
	ConfigFile *BackyConfigFile
	// Holds config file
	ConfigFilePath string

	// Global log level
	BackyLogLvl *string
}

type NotificationsConfig struct {
	Config  *viper.Viper
	Enabled bool
}

type CmdOutput struct {
	StdErr []byte
	StdOut []byte
}

type BackyCommandOutput interface {
	Error() error
	GetOutput() CmdOutput
}
