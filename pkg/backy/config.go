package backy

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"mvdan.cc/sh/v3/shell"
)

// ReadConfig validates and reads the config file.
func ReadConfig(opts *BackyConfigOpts) *BackyConfigFile {

	if isatty.IsTerminal(os.Stdout.Fd()) {
		os.Setenv("BACKY_TERM", "enabled")
	} else if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		os.Setenv("BACKY_TERM", "enabled")
	} else {
		os.Setenv("BACKY_TERM", "disabled")
	}

	backyConfigFile := NewConfig()
	backyViper := opts.viper
	// loadEnv(backyViper)
	envFileInConfigDir := fmt.Sprintf("%s/.env", path.Dir(backyViper.ConfigFileUsed()))

	envFileErr := godotenv.Load()
	if envFileErr != nil {
		_ = godotenv.Load(envFileInConfigDir)
	}
	if backyViper.GetBool(getNestedConfig("logging", "cmd-std-out")) {
		os.Setenv("BACKY_STDOUT", "enabled")
	}

	CheckConfigValues(backyViper)
	for _, c := range opts.executeCmds {
		if !backyViper.IsSet(getCmdFromConfig(c)) {
			logging.ExitWithMSG(Sprintf("command %s is not in config file %s", c, backyViper.ConfigFileUsed()), 1, nil)
		}
	}

	for _, l := range opts.executeLists {
		if !backyViper.IsSet(getCmdListFromConfig(l)) {
			logging.ExitWithMSG(Sprintf("list %s not found", l), 1, nil)
		}
	}

	var backyLoggingOpts *viper.Viper
	isBackyLoggingOptsSet := backyViper.IsSet("logging")
	if isBackyLoggingOptsSet {
		backyLoggingOpts = backyViper.Sub("logging")
	}
	verbose := backyLoggingOpts.GetBool("verbose")

	logFile := backyLoggingOpts.GetString("file")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		globalLvl := zerolog.GlobalLevel()
		os.Setenv("BACKY_LOGLEVEL", Sprintf("%x", globalLvl))
	}

	consoleLoggingEnabled := backyLoggingOpts.GetBool("console")

	// Other qualifiers can go here as well
	if consoleLoggingEnabled {
		os.Setenv("BACKY_CONSOLE_LOGGING", "enabled")
	} else {
		os.Setenv("BACKY_CONSOLE_LOGGING", "")
	}

	writers := logging.SetLoggingWriters(backyLoggingOpts, logFile)

	log := zerolog.New(writers).With().Timestamp().Logger()

	backyConfigFile.Logger = log

	commandsMap := backyViper.GetStringMapString("commands")
	commandsMapViper := backyViper.Sub("commands")
	unmarshalErr := commandsMapViper.Unmarshal(&backyConfigFile.Cmds)
	if unmarshalErr != nil {
		panic(fmt.Errorf("error unmarshalling cmds struct: %w", unmarshalErr))
	}

	hostConfigsMap := make(map[string]*viper.Viper)

	for cmdName, cmdConf := range backyConfigFile.Cmds {
		envFileErr := testFile(cmdConf.Env)
		if envFileErr != nil {
			backyConfigFile.Logger.Info().Str("cmd", cmdName).Err(envFileErr).Send()
			os.Exit(1)
		}

		host := cmdConf.Host
		if host != nil {
			if backyViper.IsSet(getNestedConfig("hosts", *host)) {
				hostconfig := backyViper.Sub(getNestedConfig("hosts", *host))
				hostConfigsMap[*host] = hostconfig
			}
		}
	}

	hostsMapViper := backyViper.Sub("hosts")
	unmarshalErr = hostsMapViper.Unmarshal(&backyConfigFile.Hosts)
	if unmarshalErr != nil {
		panic(fmt.Errorf("error unmarshalling hosts struct: %w", unmarshalErr))
	}
	for _, v := range backyConfigFile.Hosts {

		if v.JumpHost != "" {
			proxyHost, defined := backyConfigFile.Hosts[v.JumpHost]
			if defined {
				v.ProxyHost = proxyHost
			}
		}
	}
	cmdListCfg := backyViper.Sub("cmd-configs")
	unmarshalErr = cmdListCfg.Unmarshal(&backyConfigFile.CmdConfigLists)
	if unmarshalErr != nil {
		panic(fmt.Errorf("error unmarshalling cmd list struct: %w", unmarshalErr))
	}

	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range backyConfigFile.CmdConfigLists {
		if opts.useCron {
			cron := strings.TrimSpace(cmdList.Cron)
			if cron == "" {
				delete(backyConfigFile.CmdConfigLists, cmdListName)
			}
		}
		for _, cmdInList := range cmdList.Order {
			_, cmdNameFound := backyConfigFile.Cmds[cmdInList]
			if !cmdNameFound {
				cmdNotFoundStr := fmt.Sprintf("command %s in list %s is not defined in config file", cmdInList, cmdListName)
				cmdNotFoundErr := errors.New(cmdNotFoundStr)
				cmdNotFoundSliceErr = append(cmdNotFoundSliceErr, cmdNotFoundErr)
			}
		}
		for _, notificationID := range cmdList.Notifications {
			if !backyViper.IsSet(getNestedConfig("notifications", notificationID)) {
				logging.ExitWithMSG(fmt.Sprintf("%s in list %s not found in notifications", notificationID, cmdListName), 1, nil)
			}
		}
	}

	if len(cmdNotFoundSliceErr) > 0 {
		var cmdNotFoundErrorLog = log.Fatal()
		cmdNotFoundErrorLog.Errs("commands not found", cmdNotFoundSliceErr).Send()
	}

	if opts.useCron && len(backyConfigFile.CmdConfigLists) > 0 {
		log.Info().Msg("Starting cron mode...")

	} else if opts.useCron && (len(backyConfigFile.CmdConfigLists) == 0) {
		logging.ExitWithMSG("No cron fields detected in any command lists", 1, nil)
	}

	for c := range commandsMap {
		if opts.executeCmds != nil && !contains(opts.executeCmds, c) {
			delete(backyConfigFile.Cmds, c)
		}
	}

	if len(opts.executeLists) > 0 {
		for l := range backyConfigFile.CmdConfigLists {
			if !contains(opts.executeLists, l) {
				delete(backyConfigFile.CmdConfigLists, l)
			}
		}
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

	for _, cmd := range backyConfigFile.Cmds {
		if cmd.Host != nil {
			host, hostFound := backyConfigFile.Hosts[*cmd.Host]
			if hostFound {
				cmd.RemoteHost = host
				cmd.RemoteHost.Host = host.Host
				if host.HostName != "" {
					cmd.RemoteHost.HostName = host.HostName
				}
			} else {
				cmd.RemoteHost = &Host{Host: *cmd.Host}
			}
		}

	}
	backyConfigFile.SetupNotify()
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

func (opts *BackyConfigOpts) InitConfig() {
	if opts.viper != nil {
		return
	}
	backyViper := viper.New()

	if strings.TrimSpace(opts.ConfigFilePath) != "" {
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
	opts.viper = backyViper
}

func loadEnv(backyViper *viper.Viper) {
	envFileInConfigDir := fmt.Sprintf("%s/.env", path.Dir(backyViper.ConfigFileUsed()))
	var backyEnv map[string]string
	backyEnv, envFileErr := godotenv.Read()

	// envFile, envFileErr := os.Open(".env")
	if envFileErr != nil {
		backyEnv, _ = godotenv.Read(envFileInConfigDir)
	}
	envFileErr = godotenv.Load()
	if envFileErr != nil {
		_ = godotenv.Load(envFileInConfigDir)
	}

	env := func(name string) string {
		name = strings.ToUpper(name)
		envVar, found := backyEnv[name]
		if found {
			return envVar
		}
		return ""
	}
	envVars := []string{"APP=${BACKY_APP}"}

	for indx, v := range envVars {
		if strings.Contains(v, "$") || (strings.Contains(v, "${") && strings.Contains(v, "}")) {
			out, _ := shell.Expand(v, env)
			envVars[indx] = out
			// println(out)
		}
	}
}
