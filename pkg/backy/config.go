package backy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	vault "github.com/hashicorp/vault/api"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

var homeDir string
var homeDirErr error
var backyHomeConfDir string
var configFiles []string

func (opts *ConfigOpts) InitConfig() {

	homeDir, homeDirErr = os.UserHomeDir()

	if homeDirErr != nil {
		fmt.Println(homeDirErr)
		logging.ExitWithMSG(homeDirErr.Error(), 1, nil)
	}

	backyHomeConfDir = homeDir + "/.config/backy/"

	configFiles = []string{"./backy.yml", "./backy.yaml", backyHomeConfDir + "backy.yml", backyHomeConfDir + "backy.yaml"}

	backyKoanf := koanf.New(".")

	opts.ConfigFilePath = strings.TrimSpace(opts.ConfigFilePath)

	if opts.ConfigFilePath != "" {
		err := testFile(opts.ConfigFilePath)
		if err != nil {
			logging.ExitWithMSG(fmt.Sprintf("Could not open config file %s: %v", opts.ConfigFilePath, err), 1, nil)
		}

		if err := backyKoanf.Load(file.Provider(opts.ConfigFilePath), yaml.Parser()); err != nil {
			logging.ExitWithMSG(fmt.Sprintf("error loading config: %v", err), 1, &opts.Logger)
		}
	} else {

		cFileFailures := 0
		for _, c := range configFiles {
			if err := backyKoanf.Load(file.Provider(c), yaml.Parser()); err != nil {
				cFileFailures++
			} else {
				opts.ConfigFilePath = c
				break
			}
		}
		if cFileFailures == len(configFiles) {
			logging.ExitWithMSG(fmt.Sprintf("could not find a config file. Put one in the following paths: %v", configFiles), 1, &opts.Logger)
		}
	}

	opts.koanf = backyKoanf
}

// ReadConfig validates and reads the config file.
func ReadConfig(opts *ConfigOpts) *ConfigOpts {

	if isatty.IsTerminal(os.Stdout.Fd()) {
		os.Setenv("BACKY_TERM", "enabled")
	} else if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		os.Setenv("BACKY_TERM", "enabled")
	} else {
		os.Setenv("BACKY_TERM", "disabled")
	}

	backyKoanf := opts.koanf

	opts.loadEnv()

	if backyKoanf.Bool(getNestedConfig("logging", "cmd-std-out")) {
		os.Setenv("BACKY_STDOUT", "enabled")
	}

	CheckConfigValues(backyKoanf, opts.ConfigFilePath)

	// check for commands in file
	for _, c := range opts.executeCmds {
		if !backyKoanf.Exists(getCmdFromConfig(c)) {
			logging.ExitWithMSG(Sprintf("command %s is not in config file %s", c, opts.ConfigFilePath), 1, nil)
		}
	}

	// TODO: refactor this further down the line

	// for _, l := range opts.executeLists {
	// 	if !backyKoanf.Exists(getCmdListFromConfig(l)) {
	// 		logging.ExitWithMSG(Sprintf("list %s not found", l), 1, nil)
	// 	}
	// }

	// check for verbosity, via
	// 1. config file
	// 2. TODO: CLI flag
	// 3. TODO: ENV var

	var (
		isLoggingVerbose bool
		logFile          string
	)

	isLoggingVerbose = backyKoanf.Bool(getLoggingKeyFromConfig("verbose"))

	logFile = fmt.Sprintf("%s/backy.log", path.Dir(opts.ConfigFilePath)) // get full path to logfile

	if backyKoanf.Exists(getLoggingKeyFromConfig("file")) {
		logFile = backyKoanf.String(getLoggingKeyFromConfig("file"))
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if isLoggingVerbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		globalLvl := zerolog.GlobalLevel()
		os.Setenv("BACKY_LOGLEVEL", Sprintf("%v", globalLvl))
	}

	consoleLoggingDisabled := backyKoanf.Bool(getLoggingKeyFromConfig("console-disabled"))

	os.Setenv("BACKY_CONSOLE_LOGGING", "enabled")
	// Other qualifiers can go here as well
	if consoleLoggingDisabled {
		os.Setenv("BACKY_CONSOLE_LOGGING", "")
	}

	writers := logging.SetLoggingWriters(logFile)

	log := zerolog.New(writers).With().Timestamp().Logger()

	opts.Logger = log

	log.Info().Str("config file", opts.ConfigFilePath).Send()

	unmarshalErr := backyKoanf.UnmarshalWithConf("commands", &opts.Cmds, koanf.UnmarshalConf{Tag: "yaml"})

	if unmarshalErr != nil {

		panic(fmt.Errorf("error unmarshaling cmds struct: %w", unmarshalErr))

	}

	for cmdName, cmdConf := range opts.Cmds {
		envFileErr := testFile(cmdConf.Env)
		if envFileErr != nil {
			opts.Logger.Info().Str("cmd", cmdName).Err(envFileErr).Send()
			os.Exit(1)
		}

		expandEnvVars(opts.backyEnv, cmdConf.Environment)
	}

	// Get host configurations from config file

	unmarshalErr = backyKoanf.UnmarshalWithConf("hosts", &opts.Hosts, koanf.UnmarshalConf{Tag: "yaml"})
	if unmarshalErr != nil {
		panic(fmt.Errorf("error unmarshalling hosts struct: %w", unmarshalErr))
	}
	for hostConfigName, host := range opts.Hosts {
		if host.Host == "" {
			host.Host = hostConfigName
		}
		if host.ProxyJump != "" {
			proxyHosts := strings.Split(host.ProxyJump, ",")
			for hostNum, h := range proxyHosts {
				if hostNum > 1 {
					proxyHost, defined := opts.Hosts[h]
					if defined {
						host.ProxyHost = append(host.ProxyHost, proxyHost)
					} else {
						newProxy := &Host{Host: h}
						host.ProxyHost = append(host.ProxyHost, newProxy)
					}
				} else {
					proxyHost, defined := opts.Hosts[h]
					if defined {
						host.ProxyHost = append(host.ProxyHost, proxyHost)
					} else {
						newHost := &Host{Host: h}
						host.ProxyHost = append(host.ProxyHost, newHost)
					}
				}
			}

		}
	}

	// get command lists
	// command lists should still be in the same file if no:
	// 1. key 'cmd-lists.file' is found
	// 2. hosts.yml or hosts.yaml is found in the same directory as the backy config file
	backyConfigFileDir := path.Dir(opts.ConfigFilePath)

	listsConfig := koanf.New(".")

	listConfigFiles := []string{path.Join(backyConfigFileDir, "lists.yml"), path.Join(backyConfigFileDir, "lists.yaml")}

	log.Info().Strs("list config files", listConfigFiles).Send()
	for _, l := range listConfigFiles {
		cFileFailures := 0
		if err := listsConfig.Load(file.Provider(l), yaml.Parser()); err != nil {
			cFileFailures++
		} else {
			opts.ConfigFilePath = l
			break
		}

		if cFileFailures == len(configFiles) {

			logging.ExitWithMSG(fmt.Sprintf("could not find a config file. Put one in the following paths: %v", listConfigFiles), 1, &opts.Logger)

			// logging.ExitWithMSG((fmt.Sprintf("error unmarshalling cmd list struct: %v", unmarshalErr)), 1, &opts.Logger)
		}

	}
	_ = listsConfig.UnmarshalWithConf("cmd-lists", &opts.CmdConfigLists, koanf.UnmarshalConf{Tag: "yaml"})

	if backyKoanf.Exists("cmd-lists") {

		unmarshalErr = backyKoanf.UnmarshalWithConf("cmd-lists", &opts.CmdConfigLists, koanf.UnmarshalConf{Tag: "yaml"})
		// if unmarshalErr is not nil, look for a cmd-lists.file key
		if unmarshalErr != nil {

			// if file key exists, resolve file path and try to read and unmarshal file into command lists config
			if backyKoanf.Exists("cmd-lists.file") {
				opts.CmdListFile = strings.TrimSpace(backyKoanf.String("cmd-lists.file"))

				cmdListFilePath := path.Clean(opts.CmdListFile)

				if !strings.HasPrefix(cmdListFilePath, "/") {
					opts.CmdListFile = path.Join(backyConfigFileDir, cmdListFilePath)
				}

				err := testFile(opts.CmdListFile)

				if err != nil {
					logging.ExitWithMSG(fmt.Sprintf("Could not open config file %s: %v. \n\nThe cmd-lists config should be in the main config file or should be in a lists.yml or lists.yaml file.", opts.CmdListFile, err), 1, nil)
				}

				if err := listsConfig.Load(file.Provider(opts.CmdListFile), yaml.Parser()); err != nil {
					logging.ExitWithMSG(fmt.Sprintf("error loading config: %v", err), 1, &opts.Logger)
				}

				log.Info().Str("lists config file", opts.CmdListFile).Send()

			}

		}

	}

	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range opts.CmdConfigLists {
		if opts.useCron {
			cron := strings.TrimSpace(cmdList.Cron)
			if cron == "" {
				delete(opts.CmdConfigLists, cmdListName)
			}
		}
		for _, cmdInList := range cmdList.Order {
			_, cmdNameFound := opts.Cmds[cmdInList]
			if !cmdNameFound {
				cmdNotFoundStr := fmt.Sprintf("command %s in list %s is not defined in commands section in config file", cmdInList, cmdListName)
				cmdNotFoundErr := errors.New(cmdNotFoundStr)
				cmdNotFoundSliceErr = append(cmdNotFoundSliceErr, cmdNotFoundErr)
			}
		}
	}

	// Exit program if command is not found from list
	if len(cmdNotFoundSliceErr) > 0 {
		var cmdNotFoundErrorLog = log.Fatal()
		cmdNotFoundErrorLog.Errs("commands not found", cmdNotFoundSliceErr).Send()
	}

	if opts.useCron && (len(opts.CmdConfigLists) == 0) {
		logging.ExitWithMSG("No cron fields detected in any command lists", 1, nil)
	}

	for c := range opts.Cmds {
		if opts.executeCmds != nil && !contains(opts.executeCmds, c) {
			delete(opts.Cmds, c)
		}
	}

	if len(opts.executeLists) > 0 {
		for l := range opts.CmdConfigLists {
			if !contains(opts.executeLists, l) {
				delete(opts.CmdConfigLists, l)
			}
		}
	}

	if backyKoanf.Exists("notifications") {

		unmarshalErr = backyKoanf.UnmarshalWithConf("notifications", &opts.NotificationConf, koanf.UnmarshalConf{Tag: "yaml"})
		if unmarshalErr != nil {
			fmt.Printf("error unmarshalling notifications object: %v", unmarshalErr)
		}
	}

	for _, cmd := range opts.Cmds {
		if cmd.Host != nil {
			host, hostFound := opts.Hosts[*cmd.Host]
			if hostFound {
				cmd.RemoteHost = host
				cmd.RemoteHost.Host = host.Host
				if host.HostName != "" {
					cmd.RemoteHost.HostName = host.HostName
				}
			} else {
				opts.Hosts[*cmd.Host] = &Host{Host: *cmd.Host}
				cmd.RemoteHost = &Host{Host: *cmd.Host}
			}
		}

	}
	opts.SetupNotify()
	if err := opts.setupVault(); err != nil {
		log.Err(err).Send()
	}

	return opts
}

func getNestedConfig(nestedConfig, key string) string {
	return fmt.Sprintf("%s.%s", nestedConfig, key)
}

func getCmdFromConfig(key string) string {
	return fmt.Sprintf("commands.%s", key)
}

func getLoggingKeyFromConfig(key string) string {
	if key == "" {
		return "logging"
	}
	return fmt.Sprintf("logging.%s", key)
}

func getCmdListFromConfig(list string) string {
	return fmt.Sprintf("cmd-lists.%s", list)
}

func (opts *ConfigOpts) setupVault() error {
	if !opts.koanf.Bool("vault.enabled") {
		return nil
	}
	config := vault.DefaultConfig()

	config.Address = opts.koanf.String("vault.address")
	if strings.TrimSpace(config.Address) == "" {
		config.Address = os.Getenv("VAULT_ADDR")
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return err
	}

	token := opts.koanf.String("vault.token")
	if strings.TrimSpace(token) == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("no token found, but one was required. \n\nSet the config key vault.token or the environment variable VAULT_TOKEN")
	}

	client.SetToken(token)

	unmarshalErr := opts.koanf.UnmarshalWithConf("vault.keys", &opts.VaultKeys, koanf.UnmarshalConf{Tag: "yaml"})
	if unmarshalErr != nil {
		logging.ExitWithMSG(fmt.Sprintf("error unmarshalling vault.keys into struct: %v", unmarshalErr), 1, &opts.Logger)
	}

	opts.vaultClient = client

	return nil
}

func getVaultSecret(vaultClient *vault.Client, key *VaultKey) (string, error) {
	var (
		secret *vault.KVSecret
		err    error
	)

	if key.ValueType == "KVv2" {
		secret, err = vaultClient.KVv2(key.MountPath).Get(context.Background(), key.Path)
	} else if key.ValueType == "KVv1" {
		secret, err = vaultClient.KVv1(key.MountPath).Get(context.Background(), key.Path)
	} else if key.ValueType != "" {
		return "", fmt.Errorf("type %s for key %s not known. Valid types are KVv1 or KVv2", key.ValueType, key.Name)
	} else {
		return "", fmt.Errorf("type for key %s must be specified. Valid types are KVv1 or KVv2", key.Name)

	}
	if err != nil {
		return "", fmt.Errorf("unable to read secret: %v", err)
	}

	value, ok := secret.Data[key.Name].(string)
	if !ok {
		return "", fmt.Errorf("value type assertion failed: %T %#v", secret.Data[key.Name], secret.Data[key.Name])
	}

	return value, nil
}

func isVaultKey(str string) (string, bool) {
	str = strings.TrimSpace(str)
	return strings.TrimPrefix(str, "vault:"), strings.HasPrefix(str, "vault:")
}

func parseVaultKey(str string, keys []*VaultKey) (*VaultKey, error) {
	keyName, isKey := isVaultKey(str)
	if !isKey {
		return nil, nil
	}

	for _, k := range keys {
		if k.Name == keyName {
			return k, nil
		}
	}
	return nil, fmt.Errorf("key %s not found in vault keys", keyName)
}

func GetVaultKey(str string, opts *ConfigOpts, log zerolog.Logger) string {
	key, err := parseVaultKey(str, opts.VaultKeys)
	if key == nil && err == nil {
		return str
	}
	if err != nil && key == nil {
		log.Err(err).Send()
		return ""
	}

	value, secretErr := getVaultSecret(opts.vaultClient, key)
	if secretErr != nil {
		log.Err(secretErr).Send()
		return value
	}
	return value
}
