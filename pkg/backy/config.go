package backy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman"
	"git.andrewnw.xyz/CyberShell/backy/pkg/remotefetcher"
	"git.andrewnw.xyz/CyberShell/backy/pkg/usermanager"
	vault "github.com/hashicorp/vault/api"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

const macroStart string = "%{"
const macroEnd string = "}%"
const envMacroStart string = "%{env:"
const vaultMacroStart string = "%{vault:"

func (opts *ConfigOpts) InitConfig() {
	var err error
	homeConfigDir, err := os.UserConfigDir()
	if err != nil {
		logging.ExitWithMSG(err.Error(), 1, nil)
	}
	homeCacheDir, err := os.UserCacheDir()
	if err != nil {
		logging.ExitWithMSG(err.Error(), 1, nil)
	}

	backyHomeConfDir := path.Join(homeConfigDir, "backy")
	configFiles := []string{
		"./backy.yml", "./backy.yaml",
		path.Join(backyHomeConfDir, "backy.yml"),
		path.Join(backyHomeConfDir, "backy.yaml"),
	}

	backyKoanf := koanf.New(".")
	opts.ConfigFilePath = strings.TrimSpace(opts.ConfigFilePath)

	metadataFile := "hashMetadataSample.yml"
	cacheDir := homeCacheDir

	// Load metadata from file
	opts.CachedData, err = remotefetcher.LoadMetadataFromFile(metadataFile)
	if err != nil {
		fmt.Println("Error loading metadata:", err)
		panic(err)
	}

	// Initialize cache with loaded metadata
	cache, err := remotefetcher.NewCache(metadataFile, cacheDir)
	if err != nil {
		fmt.Println("Error initializing cache:", err)
		panic(err)
	}

	// Populate cache with loaded metadata
	for _, data := range opts.CachedData {
		cache.AddDataToStore(data.Hash, *data)
	}

	opts.Cache, err = remotefetcher.NewCache(path.Join(backyHomeConfDir, "cache.yml"), backyHomeConfDir)
	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error initializing cache: %v", err), 1, nil)
	}
	// Initialize the fetcher
	println("Creating new fetcher for source", opts.ConfigFilePath)
	fetcher, err := remotefetcher.NewConfigFetcher(opts.ConfigFilePath, opts.Cache)
	println("Created new fetcher for source", opts.ConfigFilePath)

	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error initializing config fetcher: %v", err), 1, nil)
	}

	if opts.ConfigFilePath != "" {
		loadConfigFile(fetcher, opts.ConfigFilePath, backyKoanf, opts)
	} else {
		loadDefaultConfigFiles(fetcher, configFiles, backyKoanf, opts)
	}

	opts.koanf = backyKoanf
}

func loadConfigFile(fetcher remotefetcher.ConfigFetcher, filePath string, k *koanf.Koanf, opts *ConfigOpts) {
	data, err := fetcher.Fetch(filePath)
	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("Could not fetch config file %s: %v", filePath, err), 1, nil)
	}

	if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error loading config: %v", err), 1, &opts.Logger)
	}
}

func loadDefaultConfigFiles(fetcher remotefetcher.ConfigFetcher, configFiles []string, k *koanf.Koanf, opts *ConfigOpts) {
	cFileFailures := 0
	for _, c := range configFiles {
		data, err := fetcher.Fetch(c)
		if err != nil {
			cFileFailures++
			continue
		}

		if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
			cFileFailures++
			continue
		}

		break
	}

	if cFileFailures == len(configFiles) {
		logging.ExitWithMSG("Could not find any valid config file", 1, nil)
	}
}

func (opts *ConfigOpts) ReadConfig() *ConfigOpts {
	setTerminalEnv()

	backyKoanf := opts.koanf

	opts.loadEnv()

	if backyKoanf.Bool(getNestedConfig("logging", "cmd-std-out")) {
		os.Setenv("BACKY_STDOUT", "enabled")
	}

	CheckConfigValues(backyKoanf, opts.ConfigFilePath)

	validateCommands(backyKoanf, opts)

	setLoggingOptions(backyKoanf, opts)

	log := setupLogger(opts)
	opts.Logger = log

	log.Info().Str("config file", opts.ConfigFilePath).Send()

	unmarshalConfig(backyKoanf, "commands", &opts.Cmds, opts.Logger)

	validateCommandEnvironments(opts)

	unmarshalConfig(backyKoanf, "hosts", &opts.Hosts, opts.Logger)

	resolveHostConfigs(opts)

	loadCommandLists(opts, backyKoanf)

	validateCommandLists(opts)

	if opts.cronEnabled && len(opts.CmdConfigLists) == 0 {
		logging.ExitWithMSG("No cron fields detected in any command lists", 1, nil)
	}

	if err := processCmds(opts); err != nil {
		logging.ExitWithMSG(err.Error(), 1, &opts.Logger)
	}

	filterExecuteLists(opts)

	if backyKoanf.Exists("notifications") {
		unmarshalConfig(backyKoanf, "notifications", &opts.NotificationConf, opts.Logger)
	}

	opts.SetupNotify()

	if err := opts.setupVault(); err != nil {
		log.Err(err).Send()
	}

	return opts
}

func setTerminalEnv() {
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		os.Setenv("BACKY_TERM", "enabled")
	} else {
		os.Setenv("BACKY_TERM", "disabled")
	}
}

func validateCommands(k *koanf.Koanf, opts *ConfigOpts) {
	for _, c := range opts.executeCmds {
		if !k.Exists(getCmdFromConfig(c)) {
			logging.ExitWithMSG(fmt.Sprintf("command %s is not in config file %s", c, opts.ConfigFilePath), 1, nil)
		}
	}
}

func setLoggingOptions(k *koanf.Koanf, opts *ConfigOpts) {
	isLoggingVerbose := k.Bool(getLoggingKeyFromConfig("verbose"))

	// if log file is set in config file and not set on command line, use "./backy.log"
	logFile := "./backy.log"
	if opts.LogFilePath == "" && k.Exists(getLoggingKeyFromConfig("file")) {
		logFile = k.String(getLoggingKeyFromConfig("file"))
		opts.LogFilePath = logFile
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if isLoggingVerbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		os.Setenv("BACKY_LOGLEVEL", fmt.Sprintf("%v", zerolog.GlobalLevel()))
	}

	if k.Bool(getLoggingKeyFromConfig("console-disabled")) {
		os.Setenv("BACKY_CONSOLE_LOGGING", "")
	} else {
		os.Setenv("BACKY_CONSOLE_LOGGING", "enabled")
	}
}

func setupLogger(opts *ConfigOpts) zerolog.Logger {
	writers := logging.SetLoggingWriters(opts.LogFilePath)
	return zerolog.New(writers).With().Timestamp().Logger()
}

func unmarshalConfig(k *koanf.Koanf, key string, target interface{}, log zerolog.Logger) {
	if err := k.UnmarshalWithConf(key, target, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error unmarshalling %s struct: %v", key, err), 1, &log)
	}
}

func validateCommandEnvironments(opts *ConfigOpts) {
	for cmdName, cmdConf := range opts.Cmds {
		if err := testFile(cmdConf.Env); err != nil {
			opts.Logger.Info().Str("cmd", cmdName).Err(err).Send()
			os.Exit(1)
		}
		expandEnvVars(opts.backyEnv, cmdConf.Environment)
	}
}

func resolveHostConfigs(opts *ConfigOpts) {
	for hostConfigName, host := range opts.Hosts {
		if host.Host == "" {
			host.Host = hostConfigName
		}
		if host.ProxyJump != "" {
			resolveProxyHosts(host, opts)
		}
	}
}

func resolveProxyHosts(host *Host, opts *ConfigOpts) {
	proxyHosts := strings.Split(host.ProxyJump, ",")
	for _, h := range proxyHosts {
		proxyHost, defined := opts.Hosts[h]
		if !defined {
			proxyHost = &Host{Host: h}
			opts.Hosts[h] = proxyHost
		}
		host.ProxyHost = append(host.ProxyHost, proxyHost)
	}
}

func loadCommandLists(opts *ConfigOpts, backyKoanf *koanf.Koanf) {
	var backyConfigFileDir string
	var listConfigFiles []string
	var u *url.URL
	// if config file is remote, use the directory of the remote file
	if isRemoteURL(opts.ConfigFilePath) {
		_, u = getRemoteDir(opts.ConfigFilePath)
		listConfigFiles = []string{u.JoinPath("lists.yml").String(), u.JoinPath("lists.yaml").String()}
	} else {
		backyConfigFileDir = path.Dir(opts.ConfigFilePath)
		listConfigFiles = []string{
			path.Join(backyConfigFileDir, "lists.yml"),
			path.Join(backyConfigFileDir, "lists.yaml"),
		}
	}

	listsConfig := koanf.New(".")

	for _, l := range listConfigFiles {
		if loadListConfigFile(l, listsConfig, opts) {
			break
		}
	}

	if backyKoanf.Exists("cmd-lists") {
		unmarshalConfig(backyKoanf, "cmd-lists", &opts.CmdConfigLists, opts.Logger)
		if backyKoanf.Exists("cmd-lists.file") {
			loadCmdListsFile(backyKoanf, listsConfig, opts)
		}
	}
}

func isRemoteURL(filePath string) bool {
	return strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") || strings.HasPrefix(filePath, "s3://")
}

func getRemoteDir(filePath string) (string, *url.URL) {
	u, err := url.Parse(filePath)
	if err != nil {
		return "", nil
	}
	// u.Path is the path to the file, stripped of scheme and hostname
	u.Path = path.Dir(u.Path)

	return u.String(), u
}

func loadListConfigFile(filePath string, k *koanf.Koanf, opts *ConfigOpts) bool {
	fetcher, err := remotefetcher.NewConfigFetcher(filePath, opts.Cache, remotefetcher.IgnoreFileNotFound())
	if err != nil {
		// if file not found, ignore
		if errors.Is(err, remotefetcher.ErrFileNotFound) {
			return true
		}

		logging.ExitWithMSG(fmt.Sprintf("error initializing config fetcher: %v", err), 1, nil)
	}

	data, err := fetcher.Fetch(filePath)
	if err != nil {
		return false
	}

	if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		return false
	}

	opts.CmdListFile = filePath
	return true
}

func loadCmdListsFile(backyKoanf *koanf.Koanf, listsConfig *koanf.Koanf, opts *ConfigOpts) {
	opts.CmdListFile = strings.TrimSpace(backyKoanf.String("cmd-lists.file"))
	if !path.IsAbs(opts.CmdListFile) {
		opts.CmdListFile = path.Join(path.Dir(opts.ConfigFilePath), opts.CmdListFile)
	}

	fetcher, err := remotefetcher.NewConfigFetcher(opts.CmdListFile, opts.Cache)

	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error initializing config fetcher: %v", err), 1, nil)
	}

	data, err := fetcher.Fetch(opts.CmdListFile)
	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("Could not fetch config file %s: %v", opts.CmdListFile, err), 1, nil)
	}

	if err := listsConfig.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error loading config: %v", err), 1, &opts.Logger)
	}

	unmarshalConfig(listsConfig, "cmd-lists", &opts.CmdConfigLists, opts.Logger)
	opts.Logger.Info().Str("using lists config file", opts.CmdListFile).Send()
}

func validateCommandLists(opts *ConfigOpts) {
	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range opts.CmdConfigLists {
		if opts.cronEnabled && strings.TrimSpace(cmdList.Cron) == "" {
			delete(opts.CmdConfigLists, cmdListName)
			continue
		}
		for _, cmdInList := range cmdList.Order {
			if _, cmdNameFound := opts.Cmds[cmdInList]; !cmdNameFound {
				cmdNotFoundSliceErr = append(cmdNotFoundSliceErr, fmt.Errorf("command %s in list %s is not defined in commands section in config file", cmdInList, cmdListName))
			}
		}
	}

	if len(cmdNotFoundSliceErr) > 0 {
		opts.Logger.Fatal().Errs("commands not found", cmdNotFoundSliceErr).Send()
	}
}

func filterExecuteLists(opts *ConfigOpts) {
	if len(opts.executeLists) > 0 {
		for l := range opts.CmdConfigLists {
			if !contains(opts.executeLists, l) {
				delete(opts.CmdConfigLists, l)
			}
		}
	}
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

func processCmds(opts *ConfigOpts) error {

	// process commands
	for cmdName, cmd := range opts.Cmds {

		if cmd.Name == "" {
			cmd.Name = cmdName
		}
		// println("Cmd.Name = " + cmd.Name)
		hooks := cmd.Hooks
		// resolve hooks
		if hooks != nil {

			processHookSuccess := processHooks(cmd, hooks.Error, opts, "error")
			if processHookSuccess != nil {
				return processHookSuccess
			}
			processHookSuccess = processHooks(cmd, hooks.Success, opts, "success")
			if processHookSuccess != nil {
				return processHookSuccess
			}
			processHookSuccess = processHooks(cmd, hooks.Final, opts, "final")
			if processHookSuccess != nil {
				return processHookSuccess
			}
		}

		// resolve hosts
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

		// Parse package commands
		if cmd.Type == "package" {
			if cmd.PackageManager == "" {
				return fmt.Errorf("package manager is required for package command %s", cmd.PackageName)
			}
			if cmd.PackageOperation == "" {
				return fmt.Errorf("package operation is required for package command %s", cmd.PackageName)
			}
			if cmd.PackageName == "" {
				return fmt.Errorf("package name is required for package command %s", cmd.PackageName)
			}
			var err error

			// Validate the operation
			switch cmd.PackageOperation {
			case "install", "remove", "upgrade", "checkVersion":
				cmd.pkgMan, err = pkgman.PackageManagerFactory(cmd.PackageManager, pkgman.WithoutAuth())
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported package operation %s for command %s", cmd.PackageOperation, cmd.Name)
			}
		}

		// Parse user commands
		if cmd.Type == "user" {
			if cmd.Username == "" {
				return fmt.Errorf("username is required for user command %s", cmd.Name)
			}

			detectOSType(cmd, opts)
			var err error

			// Validate the operation
			switch cmd.UserOperation {
			case "add", "remove", "modify", "checkIfExists", "delete", "password":
				cmd.userMan, err = usermanager.NewUserManager(cmd.OS)
				if cmd.Host != nil {
					host, ok := opts.Hosts[*cmd.Host]
					if ok {
						cmd.userMan, err = usermanager.NewUserManager(host.OS)
					}
				}
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported user operation %s for command %s", cmd.UserOperation, cmd.Name)
			}

		}
	}
	return nil
}

// processHooks evaluates if hooks are valid Commands
//
// Takes the following arguments:
//
//  1. a []string of hooks
//  2. a map of Commands as arguments
//  3. a string hookType, must be the hook type
//
// The cmds.hookRef is modified in this function.
//
// Returns the following:
//
//	An error, if any, if the command is not found
func processHooks(cmd *Command, hooks []string, opts *ConfigOpts, hookType string) error {

	// initialize hook type
	var hookCmdFound bool
	cmd.hookRefs = map[string]map[string]*Command{}
	cmd.hookRefs[hookType] = map[string]*Command{}

	for _, hook := range hooks {

		var hookCmd *Command
		// TODO: match by Command.Name

		hookCmd, hookCmdFound = opts.Cmds[hook]

		if !hookCmdFound {
			return fmt.Errorf("error in command %s hook %s list: command %s not found", cmd.Name, hookType, hook)
		}

		cmd.hookRefs[hookType][hook] = hookCmd

		// Recursive, decide if this is good
		// if hookCmd.hookRefs == nil {
		// }
		// hookRef[hookType][h] = hookCmd
	}
	return nil
}

func detectOSType(cmd *Command, opts *ConfigOpts) error {
	if cmd.Host == nil {
		if runtime.GOOS == "linux" { // also can be specified to FreeBSD
			cmd.OS = "linux"
			opts.Logger.Info().Msg("Unix/Linux type OS detected")
		}
	}
	host, ok := opts.Hosts[*cmd.Host]
	if ok {
		if host.OS != "" {
			return nil
		}

		os, err := host.DetectOS(opts)
		os = strings.TrimSpace(os)
		if err != nil {
			return err
		}
		if os == "" {
			return fmt.Errorf("error detecting os for command %s: empty string", cmd.Name)
		}
		if strings.Contains(os, "linux") {
			os = "linux"
		}
		host.OS = os
	}
	return nil
}
