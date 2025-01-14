package backy

import (
	"bytes"
	"text/template"

	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman"
	"git.andrewnw.xyz/CyberShell/backy/pkg/usermanager"
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
		OS                 string `yaml:"OS,omitempty"`
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
		Name string `yaml:"name,omitempty"`

		// command to run
		Cmd string `yaml:"cmd"`

		// Possible values: script, scriptFile
		// If blank, it is regular command.
		Type string `yaml:"type,omitempty"`

		// host on which to run cmd
		Host *string `yaml:"host,omitempty"`

		// Hooks are for running commands on certain events
		Hooks *Hooks `yaml:"hooks,omitempty"`

		// hook refs are internal references of commands for each hook type
		hookRefs map[string]map[string]*Command

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

		PackageManager string `yaml:"packageManager,omitempty"`

		PackageName string `yaml:"packageName,omitempty"`

		// Version specifies the desired version for package execution
		PackageVersion string `yaml:"packageVersion,omitempty"`

		// PackageOperation specifies the action for package-related commands (e.g., "install" or "remove")
		PackageOperation string `yaml:"packageOperation,omitempty"`

		pkgMan pkgman.PackageManager

		packageCmdSet bool

		// RemoteSource specifies a URL to fetch the command or configuration remotely
		RemoteSource string `yaml:"remoteSource,omitempty"`

		// FetchBeforeExecution determines if the remoteSource should be fetched before running
		FetchBeforeExecution bool `yaml:"fetchBeforeExecution,omitempty"`

		// Username specifies the username for user creation or related operations
		Username string `yaml:"username,omitempty"`

		// UserGroups specifies the groups to add the user to
		UserGroups []string `yaml:"userGroups,omitempty"`

		// UserHome specifies the home directory for the user
		UserHome string `yaml:"userHome,omitempty"`

		// UserShell specifies the shell for the user
		UserShell string `yaml:"userShell,omitempty"`

		// SystemUser specifies whether the user is a system account
		SystemUser bool `yaml:"systemUser,omitempty"`

		// UserPassword specifies the password for the user (can be file: or plain text)
		UserPassword string `yaml:"userPassword,omitempty"`

		userMan usermanager.UserManager

		// OS for the command, only used when type is user
		OS string `yaml:"OS,omitempty"`

		// UserOperation specifies the action for user-related commands (e.g., "create" or "remove")
		UserOperation string `yaml:"userOperation,omitempty"`

		userCmdSet bool

		stdin *strings.Reader
	}

	RemoteSource struct {
		URL  string `yaml:"url"`
		Type string `yaml:"type"` // e.g., yaml
		Auth struct {
			AccessKey string `yaml:"accessKey"`
			SecretKey string `yaml:"secretKey"`
		} `yaml:"auth"`
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
		Source       string `yaml:"source"` // URL to fetch remote commands
		Type         string `yaml:"type"`
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

		// Holds log file
		LogFilePath string

		// for command list file
		CmdListFile string

		// use command lists using cron
		cronEnabled bool
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

	Hooks struct {
		Error   []string `yaml:"error,omitempty"`
		Success []string `yaml:"success,omitempty"`
		Final   []string `yaml:"final,omitempty"`
	}

	CmdResult struct {
		CmdName  string // Name of the command executed
		ListName string // Name of the command list
		Error    error  // Error encountered, if any
	}
)
