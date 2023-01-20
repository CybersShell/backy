// backy.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0
package backy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"gopkg.in/natefinch/lumberjack.v2"
)

var requiredKeys = []string{"commands", "cmd-configs"}

var Sprintf = fmt.Sprintf

func (c *BackyConfigOpts) LogLvl(level string) BackyOptionFunc {

	return func(bco *BackyConfigOpts) {
		c.BackyLogLvl = &level
	}
}

func AddCommands(commands []string) BackyOptionFunc {
	return func(bco *BackyConfigOpts) {
		bco.executeCmds = append(bco.executeCmds, commands...)
	}
}

func NewOpts(configFilePath string, opts ...BackyOptionFunc) *BackyConfigOpts {
	b := &BackyConfigOpts{}
	b.ConfigFilePath = configFilePath
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

/*
NewConfig initializes new config that holds information	from the config file
*/
func NewConfig() *BackyConfigFile {
	return &BackyConfigFile{
		Cmds:           make(map[string]*Command),
		CmdConfigLists: make(map[string]*CmdConfig),
		Hosts:          make(map[string]Host),
		Notifications:  make(map[string]*NotificationsConfig),
	}
}

type environmentVars struct {
	file string
	env  []string
}

// RunCmd runs a Command.
// The environment of local commands will be the machine's environment plus any extra
// variables specified in the Env file or Environment.
//
// If host is specifed, the command will call ConnectToSSHHost,
// returning a client that is used to run the command.
func (command *Command) RunCmd(log *zerolog.Logger) {

	var envVars = environmentVars{
		file: command.Env,
		env:  command.Environment,
	}
	envVars.env = append(envVars.env, os.Environ()...)

	var cmdArgsStr string
	for _, v := range command.CmdArgs {
		cmdArgsStr += fmt.Sprintf(" %s", v)
	}
	var hostStr string
	if command.Host != nil {
		hostStr = *command.Host
	}

	log.Info().Str("Command", fmt.Sprintf("Running command: %s %s on host %s", command.Cmd, cmdArgsStr, hostStr)).Send()
	if command.Host != nil {
		command.RemoteHost.Host = *command.Host
		command.RemoteHost.Port = 22
		sshc, err := command.RemoteHost.ConnectToSSHHost(log)
		if err != nil {
			log.Err(fmt.Errorf("ssh dial: %w", err)).Send()
		}
		defer sshc.Close()
		commandSession, err := sshc.NewSession()
		if err != nil {
			log.Err(fmt.Errorf("new ssh session: %w", err)).Send()
		}
		defer commandSession.Close()

		injectEnvIntoSSH(envVars, commandSession, log)
		cmd := command.Cmd
		for _, a := range command.CmdArgs {
			cmd += " " + a
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		commandSession.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		commandSession.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err = commandSession.Run(cmd)
		log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
		log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()

		if err != nil {
			log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
		}
	} else {
		cmdExists := command.checkCmdExists()
		if !cmdExists {
			log.Error().Str(command.Cmd, "not found").Send()
		}
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
			injectEnvIntoLocalCMD(envVars, localCMD, log)
			err = localCMD.Run()
			log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
			log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()

			if err != nil {
				log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
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
		injectEnvIntoLocalCMD(envVars, localCMD, log)
		err = localCMD.Run()
		log.Info().Bytes(fmt.Sprintf("%s stdout", command.Cmd), stdoutBuf.Bytes()).Send()
		log.Info().Bytes(fmt.Sprintf("%s stderr", command.Cmd), stderrBuf.Bytes()).Send()
		if err != nil {
			log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
		}
	}
}

func cmdListWorker(id int, jobs <-chan *CmdConfig, config *BackyConfigFile, results chan<- string) {
	for j := range jobs {
		for _, cmd := range j.Order {
			cmdToRun := config.Cmds[cmd]
			cmdToRun.RunCmd(&config.Logger)
		}
		results <- "done"
	}
}

// RunBackyConfig runs a command list from the BackyConfigFile.
func (config *BackyConfigFile) RunBackyConfig() {
	configListsLen := len(config.CmdConfigLists)
	jobs := make(chan *CmdConfig, configListsLen)
	results := make(chan string)
	// configChan := make(chan map[string]Command)

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= 3; w++ {
		go cmdListWorker(w, jobs, config, results)

	}

	// Here we send 5 `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	// configChan <- config.Cmds
	for _, cmdConfig := range config.CmdConfigLists {
		jobs <- cmdConfig
		// fmt.Println("sent job", config.Order)
	}
	close(jobs)

	for a := 1; a <= configListsLen; a++ {
		<-results
	}

}

func (config *BackyConfigFile) ExecuteCmds() {
	for _, cmd := range config.Cmds {
		cmd.RunCmd(&config.Logger)
	}
}

// ReadAndParseConfigFile validates and reads the config file.
func ReadAndParseConfigFile(configFile string, lists []string) *BackyConfigFile {

	backyConfigFile := NewConfig()

	backyViper := viper.New()

	if configFile != "" {
		backyViper.SetConfigFile(configFile)
	} else {
		backyViper.SetConfigName("backy.yaml")          // name of config file (with extension)
		backyViper.SetConfigType("yaml")                // REQUIRED if the config file does not have the extension in the name
		backyViper.AddConfigPath(".")                   // optionally look for config in the working directory
		backyViper.AddConfigPath("$HOME/.config/backy") // call multiple times to add many search paths
	}
	err := backyViper.ReadInConfig() // Find and read the config file
	if err != nil {                  // Handle errors reading the config file
		panic(fmt.Errorf("fatal error reading config file %s: %w", backyViper.ConfigFileUsed(), err))
	}

	CheckConfigValues(backyViper)

	for _, l := range lists {
		if !backyViper.IsSet(getCmdListFromConfig(l)) {
			logging.ExitWithMSG(Sprintf("list %s not found", l), 1, nil)
		}
	}

	var backyLoggingOpts *viper.Viper
	backyLoggingOptsSet := backyViper.IsSet("logging")
	if backyLoggingOptsSet {
		backyLoggingOpts = backyViper.Sub("logging")
	}
	verbose := backyLoggingOpts.GetBool("verbose")

	logFile := backyLoggingOpts.GetString("file")
	if verbose {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		globalLvl := zerolog.GlobalLevel().String()
		os.Setenv("BACKY_LOGLEVEL", globalLvl)
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
	if strings.TrimSpace(logFile) != "" {
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
	}

	var cmdNames []string
	for k := range commandsMap {
		cmdNames = append(cmdNames, k)
	}
	hostConfigsMap := make(map[string]*viper.Viper)

	for _, cmdName := range cmdNames {
		subCmd := backyViper.Sub(getNestedConfig("commands", cmdName))

		hostSet := subCmd.IsSet("host")
		host := subCmd.GetString("host")

		if hostSet {
			log.Debug().Timestamp().Str(cmdName, "host is set").Str("host", host).Send()
			if backyViper.IsSet(getNestedConfig("hosts", host)) {
				hostconfig := backyViper.Sub(getNestedConfig("hosts", host))
				hostConfigsMap[host] = hostconfig
			}
		} else {
			log.Debug().Timestamp().Str(cmdName, "host is not set").Send()
		}

	}

	cmdListCfg := backyViper.Sub("cmd-configs")
	unmarshalErr = cmdListCfg.Unmarshal(&backyConfigFile.CmdConfigLists)
	if unmarshalErr != nil {
		panic(fmt.Errorf("error unmarshalling cmd list struct: %w", unmarshalErr))
	}
	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range backyConfigFile.CmdConfigLists {
		for _, cmdInList := range cmdList.Order {
			_, cmdNameFound := backyConfigFile.Cmds[cmdInList]
			if !cmdNameFound {
				cmdNotFoundStr := fmt.Sprintf("command %s is not defined in config file", cmdInList)
				cmdNotFoundErr := errors.New(cmdNotFoundStr)
				cmdNotFoundSliceErr = append(cmdNotFoundSliceErr, cmdNotFoundErr)
			} else {
				log.Info().Str(cmdInList, "found in "+cmdListName).Send()
			}
		}
		for _, notificationID := range cmdList.Notifications {

			cmdList.NotificationsConfig = make(map[string]*NotificationsConfig)
			notifConfig := backyViper.Sub(getNestedConfig("notifications", notificationID))
			config := &NotificationsConfig{
				Config:  notifConfig,
				Enabled: true,
			}
			cmdList.NotificationsConfig[notificationID] = config
			// First we get a "copy" of the entry
			if entry, ok := cmdList.NotificationsConfig[notificationID]; ok {

				// Then we modify the copy
				entry.Config = notifConfig
				entry.Enabled = true

				// Then we reassign the copy
				cmdList.NotificationsConfig[notificationID] = entry
			}
			backyConfigFile.CmdConfigLists[cmdListName].NotificationsConfig[notificationID] = config

		}
	}

	if len(lists) > 0 {
		for l := range backyConfigFile.CmdConfigLists {
			if !contains(lists, l) {
				delete(backyConfigFile.CmdConfigLists, l)
			}
		}
	}

	if len(cmdNotFoundSliceErr) > 0 {
		var cmdNotFoundErrorLog = log.Fatal()
		for _, err := range cmdNotFoundSliceErr {
			if err != nil {
				cmdNotFoundErrorLog.Err(err)
			}
		}
		cmdNotFoundErrorLog.Send()
	}

	var notificationsMap = make(map[string]interface{})
	if backyViper.IsSet("notifications") {
		notificationsMap = backyViper.GetStringMap("notifications")
		for id := range notificationsMap {
			notifConfig := backyViper.Sub(getNestedConfig("notifications", id))
			config := &NotificationsConfig{
				Config:  notifConfig,
				Enabled: true,
			}
			backyConfigFile.Notifications[id] = config
		}
	}

	return backyConfigFile
}

// GetCmdsInConfigFile validates and reads the config file for commands.
func (opts *BackyConfigOpts) GetCmdsInConfigFile() *BackyConfigFile {

	backyConfigFile := NewConfig()

	backyViper := viper.New()

	if opts.ConfigFilePath != strings.TrimSpace("") {
		backyViper.SetConfigFile(opts.ConfigFilePath)
	} else {
		backyViper.SetConfigName("backy.yaml")          // name of config file (with extension)
		backyViper.SetConfigType("yaml")                // REQUIRED if the config file does not have the extension in the name
		backyViper.AddConfigPath(".")                   // optionally look for config in the working directory
		backyViper.AddConfigPath("$HOME/.config/backy") // call multiple times to add many search paths
	}
	err := backyViper.ReadInConfig() // Find and read the config file
	if err != nil {                  // Handle errors reading the config file
		panic(fmt.Errorf("fatal error reading config file %s: %w", backyViper.ConfigFileUsed(), err))
	}

	CheckConfigValues(backyViper)
	for _, c := range opts.executeCmds {
		if !backyViper.IsSet(getCmdFromConfig(c)) {
			logging.ExitWithMSG(Sprintf("command %s is not in config file %s", c, backyViper.ConfigFileUsed()), 1, nil)
		}
	}
	var backyLoggingOpts *viper.Viper
	backyLoggingOptsSet := backyViper.IsSet("logging")
	if backyLoggingOptsSet {
		backyLoggingOpts = backyViper.Sub("logging")
	}
	verbose := backyLoggingOpts.GetBool("verbose")

	logFile := backyLoggingOpts.GetString("file")
	if verbose {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		globalLvl := zerolog.GlobalLevel().String()
		os.Setenv("BACKY_LOGLEVEL", globalLvl)
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
	if strings.TrimSpace(logFile) != "" {
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
	}

	var cmdNames []string
	for c := range commandsMap {
		if contains(opts.executeCmds, c) {
			cmdNames = append(cmdNames, c)
		}
		if !contains(opts.executeCmds, c) {
			delete(backyConfigFile.Cmds, c)
		}
	}

	hostConfigsMap := make(map[string]*viper.Viper)

	for _, cmdName := range cmdNames {
		subCmd := backyViper.Sub(getNestedConfig("commands", cmdName))

		hostSet := subCmd.IsSet("host")
		host := subCmd.GetString("host")

		if hostSet {
			log.Debug().Timestamp().Str(cmdName, "host is set").Str("host", host).Send()
			if backyViper.IsSet(getNestedConfig("hosts", host)) {
				hostconfig := backyViper.Sub(getNestedConfig("hosts", host))
				hostConfigsMap[host] = hostconfig
			}
		} else {
			log.Debug().Timestamp().Str(cmdName, "host is not set").Send()
		}

	}

	return backyConfigFile
}

func getNestedConfig(nestedConfig, key string) string {
	return fmt.Sprintf("%s.%s", nestedConfig, key)
}

func getCmdFromConfig(key string) string {
	return fmt.Sprintf("commands.%s", key)
}
func getCmdListFromConfig(list string) string {
	return fmt.Sprintf("cmd-configs.%s", list)
}

func resolveDir(path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return path, err
	}
	dir := usr.HomeDir
	if path == "~" {
		// In case of "~", which won't be caught by the "else if"
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		// Use strings.HasPrefix so we don't match paths like
		// "/something/~/something/"
		path = filepath.Join(dir, path[2:])
	}
	return path, nil
}

func injectEnvIntoSSH(envVarsToInject environmentVars, process *ssh.Session, log *zerolog.Logger) {
	if envVarsToInject.file != "" {
		envPath, envPathErr := resolveDir(envVarsToInject.file)
		if envPathErr != nil {
			log.Err(envPathErr).Send()
		}
		file, err := os.Open(envPath)
		if err != nil {
			log.Err(err).Send()
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			envVar := scanner.Text()
			envVarArr := strings.Split(envVar, "=")
			process.Setenv(envVarArr[0], envVarArr[1])
		}
		if err := scanner.Err(); err != nil {
			log.Err(err).Send()
		}
	}
	if len(envVarsToInject.env) > 0 {
		for _, envVal := range envVarsToInject.env {
			if strings.Contains(envVal, "=") {
				envVarArr := strings.Split(envVal, "=")
				process.Setenv(strings.ToUpper(envVarArr[0]), envVarArr[1])
			}
		}
	}
}

func injectEnvIntoLocalCMD(envVarsToInject environmentVars, process *exec.Cmd, log *zerolog.Logger) {
	if envVarsToInject.file != "" {
		envPath, envPathErr := resolveDir(envVarsToInject.file)
		if envPathErr != nil {
			log.Error().Err(envPathErr).Send()
		}
		file, err := os.Open(envPath)
		if err != nil {
			log.Err(err).Send()
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			envVar := scanner.Text()
			process.Env = append(process.Env, envVar)
		}
		if err := scanner.Err(); err != nil {
			log.Err(err).Send()
		}
	}
	if len(envVarsToInject.env) > 0 {
		for _, envVal := range envVarsToInject.env {
			if strings.Contains(envVal, "=") {
				process.Env = append(process.Env, envVal)
			}
		}
	}
	envVarsToInject.env = append(envVarsToInject.env, os.Environ()...)
}

func (cmd *Command) checkCmdExists() bool {
	_, err := exec.LookPath(cmd.Cmd)
	return err == nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func CheckConfigValues(config *viper.Viper) {

	for _, key := range requiredKeys {
		isKeySet := config.IsSet(key)
		if !isKeySet {
			logging.ExitWithMSG(Sprintf("Config key %s is not defined in %s", key, config.ConfigFileUsed()), 1, nil)
		}

	}
}
