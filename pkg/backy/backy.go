package backy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Host defines a host to which to connect
// If not provided, the values will be looked up in the default ssh config files
type Host struct {
	ConfigFilePath     string
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
	Remote bool `yaml:"remote,omitempty"`

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
}

type BackyGlobalOpts struct {
}

type BackyConfigFile struct {
	/*
		Cmds holds the commands for a list.
		Key is the name of the command,
	*/
	Cmds map[string]Command `yaml:"commands"`

	/*
		CmdLists holds the lists of commands to be run in order.
		Key is the command list name.
	*/
	CmdLists map[string][]string `yaml:"cmd-lists"`

	/*
		Hosts holds the Host config.
		key is the host.
	*/
	Hosts map[string]Host `yaml:"hosts"`

	Logger zerolog.Logger
}

// BackupConfig is a configuration struct that is used to define backups
type BackupConfig struct {
	Name       string
	BackupType string
	ConfigPath string

	Cmd Command
}

/*
* Runs a backup configuration
 */

func (command Command) RunCmd(log *zerolog.Logger) {

	var cmdArgsStr string
	for _, v := range command.CmdArgs {
		cmdArgsStr += fmt.Sprintf(" %s", v)
	}

	fmt.Printf("\n\nRunning command: " + command.Cmd + " " + cmdArgsStr + " on host " + *command.Host + "...\n\n")
	if command.Host != nil {

		command.RemoteHost.Host = *command.Host
		command.RemoteHost.Port = 22
		sshc, err := command.RemoteHost.ConnectToSSHHost(log)
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
		for _, a := range command.CmdArgs {
			cmd += " " + a
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		s.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		s.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = s.Run(cmd)
		log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
		log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()

		if err != nil {
			panic(fmt.Errorf("error when running cmd " + cmd + "\n Error: " + err.Error()))
		}
	} else {
		// shell := "/bin/bash"
		var err error
		if command.Shell != "" {
			cmdArgsStr = fmt.Sprintf("%s %s", command.Cmd, cmdArgsStr)
			localCMD := exec.Command(command.Shell, "-c", cmdArgsStr)
			if command.Dir != nil {
				localCMD.Dir = *command.Dir
			}

			var stdoutBuf, stderrBuf bytes.Buffer
			localCMD.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
			localCMD.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
			err = localCMD.Run()
			log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
			log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()

			if err != nil {
				panic(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err))
			}
			return
		}
		localCMD := exec.Command(command.Cmd, command.CmdArgs...)
		if command.Dir != nil {
			localCMD.Dir = *command.Dir
		}
		var stdoutBuf, stderrBuf bytes.Buffer
		localCMD.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		localCMD.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = localCMD.Run()
		log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
		log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()
		if err != nil {
			panic(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err))
		}
	}
}

func (config *BackyConfigFile) RunBackyConfig() {
	for _, list := range config.CmdLists {
		for _, cmd := range list {
			cmdToRun := config.Cmds[cmd]
			cmdToRun.RunCmd(&config.Logger)
		}
	}
}

type BackyConfigOpts struct {
	// Holds config file
	ConfigFile *BackyConfigFile
	// Holds config file
	ConfigFilePath string

	// Global log level
	BackyLogLvl *string
}

type BackyOptionFunc func(*BackyConfigOpts)

func (c *BackyConfigOpts) LogLvl(level string) BackyOptionFunc {
	return func(bco *BackyConfigOpts) {
		c.BackyLogLvl = &level
	}
}
func (c *BackyConfigOpts) GetConfig() {
	c.ConfigFile = ReadAndParseConfigFile(c.ConfigFilePath)
}

func New() BackupConfig {
	return BackupConfig{}
}

func NewOpts(configFilePath string, opts ...BackyOptionFunc) *BackyConfigOpts {
	b := &BackyConfigOpts{}
	b.ConfigFilePath = configFilePath
	for _, opt := range opts {
		opt(b)
	}
	return b
}

/*
*	NewConfig initializes new config that holds information
* 	from the config file
 */
func NewConfig() *BackyConfigFile {
	return &BackyConfigFile{
		Cmds:     make(map[string]Command),
		CmdLists: make(map[string][]string),
		Hosts:    make(map[string]Host),
	}
}

func ReadAndParseConfigFile(configFile string) *BackyConfigFile {

	backyConfigFile := NewConfig()

	backyViper := viper.New()

	if configFile != "" {
		backyViper.SetConfigFile(configFile)
	} else {
		backyViper.SetConfigName("backy")               // name of config file (without extension)
		backyViper.SetConfigType("yaml")                // REQUIRED if the config file does not have the extension in the name
		backyViper.AddConfigPath(".")                   // optionally look for config in the working directory
		backyViper.AddConfigPath("$HOME/.config/backy") // call multiple times to add many search paths
	}
	err := backyViper.ReadInConfig() // Find and read the config file
	if err != nil {                  // Handle errors reading the config file
		panic(fmt.Errorf("fatal error finding config file: %w", err))
	}

	backyLoggingOpts := backyViper.Sub("logging")
	verbose := backyLoggingOpts.GetBool("verbose")

	logFile := backyLoggingOpts.GetString("file")
	if verbose {
		zerolog.Level.String(zerolog.DebugLevel)
	}
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC1123}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s: ", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s", i))
	}

	fileLogger := &lumberjack.Logger{
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}
	if strings.Trim(logFile, " ") != "" {
		fileLogger.Filename = logFile
	} else {
		fileLogger.Filename = "./backy.log"
	}

	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// zerolog.TimeFieldFormat = time.RFC1123
	writers := zerolog.MultiLevelWriter(os.Stdout, fileLogger)
	log := zerolog.New(writers).With().Timestamp().Logger()

	backyConfigFile.Logger = log

	commandsMap := backyViper.GetStringMapString("commands")
	commandsMapViper := backyViper.Sub("commands")
	unmarshalErr := commandsMapViper.Unmarshal(&backyConfigFile.Cmds)
	if unmarshalErr != nil {
		panic(fmt.Errorf("error unmarshalling cmds struct: %w", unmarshalErr))
	} else {
		for cmdName, cmdConf := range backyConfigFile.Cmds {
			fmt.Printf("\nCommand Name: %s\n", cmdName)
			fmt.Printf("Shell: %v\n", cmdConf.Shell)
			fmt.Printf("Command: %s\n", cmdConf.Cmd)

			if len(cmdConf.CmdArgs) > 0 {
				fmt.Println("\nCmd Args:")
				for _, args := range cmdConf.CmdArgs {
					fmt.Printf("%s\n", args)
				}
			}
			if cmdConf.Host != nil {
				fmt.Printf("Host: %s\n", *backyConfigFile.Cmds[cmdName].Host)
			}
		}
		os.Exit(0)
	}
	var cmdNames []string
	for k := range commandsMap {
		cmdNames = append(cmdNames, k)
	}
	hostConfigsMap := make(map[string]*viper.Viper)

	for _, cmdName := range cmdNames {
		var backupCmdStruct Command
		subCmd := backyViper.Sub(getNestedConfig("commands", cmdName))

		hostSet := subCmd.IsSet("host")
		host := subCmd.GetString("host")

		if hostSet {
			log.Debug().Timestamp().Str(cmdName, "host is set").Str("host", host).Send()
			backupCmdStruct.Host = &host
			if backyViper.IsSet(getNestedConfig("hosts", host)) {
				hostconfig := backyViper.Sub(getNestedConfig("hosts", host))
				hostConfigsMap[host] = hostconfig
			}
		} else {
			log.Debug().Timestamp().Str(cmdName, "host is not set").Send()
		}

		// backyConfigFile.Cmds[cmdName] = backupCmdStruct

	}

	cmdListCfg := backyViper.GetStringMapStringSlice("cmd-lists")
	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range cmdListCfg {
		for _, cmdInList := range cmdList {
			_, cmdNameFound := backyConfigFile.Cmds[cmdInList]
			if !backyViper.IsSet(getNestedConfig("commands", cmdInList)) && !cmdNameFound {
				cmdNotFoundStr := fmt.Sprintf("command definition %s is not in config file\n", cmdInList)
				cmdNotFoundErr := errors.New(cmdNotFoundStr)
				cmdNotFoundSliceErr = append(cmdNotFoundSliceErr, cmdNotFoundErr)
			} else {
				backyConfigFile.CmdLists[cmdListName] = append(backyConfigFile.CmdLists[cmdListName], cmdInList)
			}
		}
	}
	for _, err := range cmdNotFoundSliceErr {
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	return backyConfigFile
}

func getNestedConfig(nestedConfig, key string) string {
	return fmt.Sprintf("%s.%s", nestedConfig, key)
}

func getNestedSSHConfig(key string) string {
	return fmt.Sprintf("hosts.%s.config", key)
}
