package backy

import (
	"bytes"
	"text/template"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kevinburke/ssh_config"
	"github.com/knadh/koanf/v2"
	"github.com/nikoksr/notify"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

type (

	// Host defines a host to which to connect.
	// If not provided, the values will be looked up in the default ssh config files
	Host struct {
		ConfigFilePath     string `yaml:"config,omitempty"`
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
		Name            string   `yaml:"name,omitempty"`
		Cron            string   `yaml:"cron,omitempty"`
		RunCmdOnFailure string   `yaml:"runCmdOnFailure,omitempty"`
		Order           []string `yaml:"order,omitempty"`
		Notifications   []string `yaml:"notifications,omitempty"`
		GetOutput       bool     `yaml:"getOutput,omitempty"`
		NotifyOnSuccess bool     `yaml:"notifyOnSuccess,omitempty"`

		NotifyConfig *notify.Notify
	}

	ConfigOpts struct {
		// Cmds holds the commands for a list.
		// Key is the name of the command,
		Cmds map[string]*Command `yaml:"commands"`

		// CmdConfigLists holds the lists of commands to be run in order.
		// Key is the command list name.
		CmdConfigLists map[string]*CmdList `yaml:"cmd-lists"`

		// Hosts holds the Host config.
		// key is the host.
		Hosts map[string]*Host `yaml:"hosts"`

		Logger zerolog.Logger

		// Global log level
		BackyLogLvl *string

		// Holds config file
		ConfigFilePath string

		// for command list file
		CmdListFile string

		// use command lists using cron
		useCron bool
		// Holds commands to execute for the exec command
		executeCmds []string
		// Holds lists to execute for the backup command
		executeLists []string

		// Holds env vars from .env file
		backyEnv map[string]string

		vaultClient *vaultapi.Client

		List ListConfig

		VaultKeys []*VaultKey `yaml:"keys"`

		koanf *koanf.Koanf

		NotificationConf *Notifications `yaml:"notifications"`
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

	Notifications struct {
		MailConfig   map[string]MailConfig   `yaml:"mail,omitempty"`
		MatrixConfig map[string]MatrixStruct `yaml:"matrix,omitempty"`
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

	ListConfig struct {
		Lists    []string
		Commands []string
		Hosts    []string
	}
)
