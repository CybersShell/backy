package backy

import (
	"bytes"
	"text/template"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kevinburke/ssh_config"
	"github.com/nikoksr/notify"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/ssh"
)

type (
	CmdConfigSchema struct {
		ID      primitive.ObjectID `bson:"_id,omitempty"`
		CmdList []string           `bson:"command-list,omitempty"`
		Name    string             `bson:"name,omitempty"`
	}

	CmdSchema struct {
		ID   primitive.ObjectID `bson:"_id,omitempty"`
		Cmd  string             `bson:"cmd,omitempty"`
		Args []string           `bson:"args,omitempty"`
		Host string             `bson:"host,omitempty"`
		Dir  string             `bson:"dir,omitempty"`
	}

	Schemas struct {
		CmdConfigSchema
		CmdSchema
	}

	// Host defines a host to which to connect.
	// If not provided, the values will be looked up in the default ssh config files
	Host struct {
		ConfigFilePath     string `yaml:"configfilepath,omitempty"`
		Host               string `yaml:"host,omitempty"`
		HostName           string `yaml:"hostname,omitempty"`
		KnownHostsFile     string `yaml:"knownhostsfile,omitempty"`
		ClientConfig       *ssh.ClientConfig
		SSHConfigFile      *sshConfigFile
		SshClient          *ssh.Client
		Port               uint16 `yaml:"port,omitempty"`
		ProxyJump          string `yaml:"proxyjump,omitempty"`
		Password           string `yaml:"password,omitempty"`
		PrivateKeyPath     string `yaml:"privatekeypath,omitempty"`
		PrivateKeyPassword string `yaml:"privatekeypassword,omitempty"`
		useDefaultConfig   bool
		User               string `yaml:"user,omitempty"`
		isProxyHost        bool
		// ProxyHost holds the configuration for a ProxyJump host
		ProxyHost []*Host
		// CertPath           string `yaml:"cert_path,omitempty"`
	}

	sshConfigFile struct {
		SshConfigFile       *ssh_config.Config
		DefaultUserSettings *ssh_config.UserSettings
	}

	Command struct {

		// command to run
		Cmd string `yaml:"cmd"`

		// Possible values: script, scriptFile
		// If blank, it is regualar command.
		Type string `yaml:"type"`

		// host on which to run cmd
		Host *string `yaml:"host,omitempty"`

		/*
			Shell specifies which shell to run the command in, if any.
			Not applicable when host is defined.
		*/
		Shell string `yaml:"shell,omitempty"`

		RemoteHost *Host `yaml:"-"`

		// Args is an array that holds the arguments to cmd
		Args []string `yaml:"args,omitempty"`

		/*
			Dir specifies a directory in which to run the command.
			Ignored if Host is set.
		*/
		Dir *string `yaml:"dir,omitempty"`

		// Env points to a file containing env variables to be used with the command
		Env string `yaml:"env,omitempty"`

		// Environment holds env variables to be used with the command
		Environment []string `yaml:"environment,omitempty"`

		// Output determines if output is requested.
		// Only works if command is in a list.
		GetOutput bool `yaml:"getOutput,omitempty"`

		ScriptEnvFile string `yaml:"scriptEnvFile"`
	}

	BackyOptionFunc func(*ConfigOpts)

	CmdList struct {
		Name          string   `yaml:"name,omitempty"`
		Cron          string   `yaml:"cron,omitempty"`
		Order         []string `yaml:"order,omitempty"`
		Notifications []string `yaml:"notifications,omitempty"`
		GetOutput     bool     `yaml:"getOutput,omitempty"`
		NotifyConfig  *notify.Notify
		// NotificationsConfig map[string]*NotificationsConfig
		// NotifyConfig        map[string]*notify.Notify
	}

	ConfigFile struct {

		// Cmds holds the commands for a list.
		// Key is the name of the command,
		Cmds map[string]*Command `yaml:"commands"`

		// CmdConfigLists holds the lists of commands to be run in order.
		// Key is the command list name.
		CmdConfigLists map[string]*CmdList `yaml:"cmd-configs"`

		// Hosts holds the Host config.
		// key is the host.
		Hosts map[string]*Host `yaml:"hosts"`

		// Notifications holds the config for different notifications.
		Notifications map[string]*NotificationsConfig

		Logger zerolog.Logger
	}

	ConfigOpts struct {
		// Global log level
		BackyLogLvl *string
		// Holds config file
		ConfigFile *ConfigFile
		// Holds config file
		ConfigFilePath string

		Schemas

		DB *mongo.Database
		// use command lists using cron
		useCron bool
		// Holds commands to execute for the exec command
		executeCmds []string
		// Holds lists to execute for the backup command
		executeLists []string

		// Holds env vars from .env file
		backyEnv map[string]string

		vaultClient *vaultapi.Client

		VaultKeys []*VaultKey `yaml:"keys"`

		viper *viper.Viper
	}

	outStruct struct {
		CmdName     string
		CmdExecuted string
		Output      []string
	}

	VaultKey struct {
		Name      string `yaml:"name"`
		Path      string `yaml:"path"`
		ValueType string `yaml:"type"`
		MountPath string `yaml:"mountpath"`
	}

	VaultConfig struct {
		Token   string      `yaml:"token"`
		Address string      `yaml:"address"`
		Enabled string      `yaml:"enabled"`
		Keys    []*VaultKey `yaml:"keys"`
	}

	NotificationsConfig struct {
		Config  *viper.Viper
		Enabled bool
	}

	CmdOutput struct {
		Err    error
		Output bytes.Buffer
	}

	environmentVars struct {
		file string
		env  []string
	}

	msgTemplates struct {
		success *template.Template
		err     *template.Template
	}
)
