package backy

import (
	"bytes"
	"text/template"

	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman"
	packagemanagercommon "git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/common"
	"git.andrewnw.xyz/CyberShell/backy/pkg/remotefetcher"
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
		KnownHostsFile     string `yaml:"knownHostsFile,omitempty"`
		ClientConfig       *ssh.ClientConfig
		SSHConfigFile      *sshConfigFile
		SshClient          *ssh.Client
		Port               uint16 `yaml:"port,omitempty"`
		ProxyJump          string `yaml:"proxyjump,omitempty"`
		Password           string `yaml:"password,omitempty"`
		PrivateKeyPath     string `yaml:"IdentityFile,omitempty"`
		PrivateKeyPassword string `yaml:"privateKeyPassword,omitempty"`
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

		Cmd string `yaml:"cmd"`

		// See CommandType enum further down the page for acceptable values
		Type CommandType `yaml:"type,omitempty"`

		Host  string   `yaml:"host,omitempty"`
		Hosts []string `yaml:"hosts,omitempty"`

		Hooks *Hooks `yaml:"hooks,omitempty"`

		hookRefs map[string]map[string]*Command

		Shell string `yaml:"shell,omitempty"`

		RemoteHost *Host `yaml:"-"`

		Args []string `yaml:"args,omitempty"`

		Dir *string `yaml:"dir,omitempty"`

		Env string `yaml:"env,omitempty"`

		Environment []string `yaml:"environment,omitempty"`

		ScriptEnvFile string `yaml:"scriptEnvFile"`

		Output struct {
			File   string `yaml:"file,omitempty"`
			ToLog  bool   `yaml:"toLog,omitempty"`
			InList bool   `yaml:"inList,omitempty"`
		} `yaml:"output"`

		// BEGIN PACKAGE COMMAND FIELDS

		PackageManager string `yaml:"packageManager,omitempty"`

		Packages []packagemanagercommon.Package `yaml:"packages,omitempty"`

		PackageVersion string `yaml:"packageVersion,omitempty"`

		PackageOperation PackageOperation `yaml:"packageOperation,omitempty"`

		pkgMan pkgman.PackageManager

		packageCmdSet bool
		// END PACKAGE COMMAND FIELDS

		RemoteSource string `yaml:"remoteSource,omitempty"`

		FetchBeforeExecution bool `yaml:"fetchBeforeExecution,omitempty"`

		Fetcher remotefetcher.RemoteFetcher

		// BEGIN USER COMMAND FIELDS

		Username string `yaml:"userName,omitempty"`

		UserID string `yaml:"userID,omitempty"`

		UserGroups []string `yaml:"userGroups,omitempty"`

		UserHome string `yaml:"userHome,omitempty"`

		UserShell string `yaml:"userShell,omitempty"`

		UserCreateHome bool `yaml:"userCreateHome,omitempty"`

		UserIsSystem bool `yaml:"userIsSystem,omitempty"`

		UserPassword string `yaml:"userPassword,omitempty"`

		UserSshPubKeys []string `yaml:"userSshPubKeys,omitempty"`

		userMan usermanager.UserManager

		// OS for the command, only used when type is user
		OS string `yaml:"OS,omitempty"`

		UserOperation string `yaml:"userOperation,omitempty"`

		userCmdSet bool

		// stdin only for userOperation = password (for now)
		stdin *strings.Reader

		// END USER STRUCommandType FIELDS
	}

	RemoteSource struct {
		URL  string `yaml:"url"`
		Type string `yaml:"type"` // e.g., s3, http
		Auth struct {
			AccessKey string `yaml:"accessKey"`
			SecretKey string `yaml:"secretKey"`
		} `yaml:"auth"`
	}

	BackyOptionFunc func(*ConfigOpts)

	CmdList struct {
		Name                                     string   `yaml:"name,omitempty"`
		Cron                                     string   `yaml:"cron,omitempty"`
		RunCmdOnFailure                          string   `yaml:"runCmdOnFailure,omitempty"`
		Order                                    []string `yaml:"order,omitempty"`
		Notifications                            []string `yaml:"notifications,omitempty"`
		GetCommandOutputInNotificationsOnSuccess bool     `yaml:"sendNotificationOnSuccess,omitempty"`

		Notify struct {
			OnFailure bool `yaml:"onFailure,omitempty"`
			OnSuccess bool `yaml:"onSuccess,omitempty"`
		} `yaml:"notify,omitempty"`

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
		CmdConfigLists map[string]*CmdList `yaml:"cmdLists"`

		// Hosts holds the Host config.
		// key is the host.
		Hosts map[string]*Host `yaml:"hosts"`

		Logger zerolog.Logger

		// Global log level
		BackyLogLvl *string

		CmdStdOut bool

		ConfigFilePath string

		HostsFilePath string

		ConfigDir string

		LogFilePath string

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

		Vars map[string]string `yaml:"variables"`

		VaultKeys []*VaultKey `yaml:"keys"`

		koanf *koanf.Koanf

		NotificationConf *Notifications `yaml:"notifications"`

		Cache      *remotefetcher.Cache
		CachedData []*remotefetcher.CacheData
	}

	outStruct struct {
		CmdName     string
		CmdExecuted string
		Output      []string
	}

	VaultKey struct {
		Name      string `yaml:"name"`
		Key       string `yaml:"key"`
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
		HttpConfig   map[string]HttpConfig   `yaml:"http,omitempty"`
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

	ListMetrics struct {
		Name                 string
		SuccessfulExecutions uint64
		FailedExecutions     uint64
		TotalExecutions      uint64
	}

	CommandMetrics struct {
		Name                 string
		SuccessfulExecutions uint64
		FailedExecutions     uint64
		TotalExecutions      uint64
	}

	// use ints so we can use enums
	CommandType               int
	PackageOperation          int
	AllowedExternalDirectives int
)

//go:generate go run github.com/dmarkham/enumer -linecomment -yaml -text -json -type=CommandType
const (
	DefaultCommandType      CommandType = iota //
	ScriptCommandType                          // script
	ScriptFileCommandType                      // scriptFile
	RemoteScriptCommandType                    // remoteScript
	PackageCommandType                         // package
	UserCommandType                            // user
)

//go:generate go run github.com/dmarkham/enumer -linecomment -yaml -text -json -type=PackageOperation
const (
	DefaultPO                    PackageOperation = iota //
	PackageOperationInstall                              // install
	PackageOperationUpgrade                              // upgrade
	PackageOperationPurge                                // purge
	PackageOperationRemove                               // remove
	PackageOperationCheckVersion                         // checkVersion
	PackageOperationIsInstalled                          // isInstalled
)

//go:generate go run github.com/dmarkham/enumer -linecomment -yaml -text -json -type=AllowedExternalDirectives
const (
	DefaultExternalDir                AllowedExternalDirectives = iota
	AllowedExternalDirectiveVault                               // vault
	AllowedExternalDirectiveVaultEnv                            // vault-env
	AllowedExternalDirectiveVaultFile                           // vault-file
	AllowedExternalDirectiveAll                                 // vault-file-env
	AllowedExternalDirectiveFileEnv                             // file-env
	AllowedExternalDirectiveFile                                // file
	AllowedExternalDirectiveEnv                                 // env
)
