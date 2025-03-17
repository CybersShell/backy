package backy

import (
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

const (
	externDirectiveStart      string = "%{"
	externDirectiveEnd        string = "}%"
	externFileDirectiveStart  string = "%{file:"
	envExternDirectiveStart   string = "%{env:"
	vaultExternDirectiveStart string = "%{vault:"
)

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

	// metadataFile := "hashMetadataSample.yml"

	cacheDir := homeCacheDir

	// Load metadata from file
	opts.CachedData, err = remotefetcher.LoadMetadataFromFile(path.Join(backyHomeConfDir, "cache", "cache.yml"))
	if err != nil {
		fmt.Println("Error loading metadata:", err)
		logging.ExitWithMSG(err.Error(), 1, &opts.Logger)
	}

	// Initialize cache with loaded metadata
	cache, err := remotefetcher.NewCache(path.Join(backyHomeConfDir, "cache.yml"), cacheDir)
	if err != nil {
		fmt.Println("Error initializing cache:", err)
		logging.ExitWithMSG(err.Error(), 1, &opts.Logger)
	}

	// Populate cache with loaded metadata
	for _, data := range opts.CachedData {
		if err := cache.AddDataToStore(data.Hash, *data); err != nil {
			logging.ExitWithMSG(err.Error(), 1, &opts.Logger)
		}
	}

	opts.Cache, err = remotefetcher.NewCache(path.Join(backyHomeConfDir, "cache.yml"), backyHomeConfDir)
	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error initializing cache: %v", err), 1, nil)
	}

	if isRemoteURL(opts.ConfigFilePath) {
		p, _ := getRemoteDir(opts.ConfigFilePath)
		opts.ConfigDir = p
	}

	fetcher, err := remotefetcher.NewRemoteFetcher(opts.ConfigFilePath, opts.Cache)
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

func (opts *ConfigOpts) ReadConfig() *ConfigOpts {
	setTerminalEnv()

	backyKoanf := opts.koanf

	if backyKoanf.Exists("variables") {
		unmarshalConfig(backyKoanf, "variables", &opts.Vars, opts.Logger)
	}

	getConfigDir(opts)

	opts.loadEnv()

	if backyKoanf.Bool(getNestedConfig("logging", "cmd-std-out")) {
		os.Setenv("BACKY_CMDSTDOUT", "enabled")
	}

	// override the default value of cmd-std-out if flag is set
	if opts.CmdStdOut {
		os.Setenv("BACKY_CMDSTDOUT", "enabled")
	}

	CheckConfigValues(backyKoanf, opts.ConfigFilePath)

	validateExecCommandsFromCLI(backyKoanf, opts)

	setLoggingOptions(backyKoanf, opts)

	log := setupLogger(opts)
	opts.Logger = log

	log.Info().Str("config file", opts.ConfigFilePath).Send()

	if err := opts.setupVault(); err != nil {
		log.Err(err).Send()
	}

	unmarshalConfig(backyKoanf, "commands", &opts.Cmds, opts.Logger)

	getCommandEnvironments(opts)

	unmarshalConfig(backyKoanf, "hosts", &opts.Hosts, opts.Logger)

	resolveHostConfigs(opts)

	for k, v := range opts.Vars {
		v = getExternalConfigDirectiveValue(v, opts)
		opts.Vars[k] = v
	}

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

	return opts
}

func loadConfigFile(fetcher remotefetcher.RemoteFetcher, filePath string, k *koanf.Koanf, opts *ConfigOpts) {
	data, err := fetcher.Fetch(filePath)
	if err != nil {
		logging.ExitWithMSG(generateFileFetchErrorString(filePath, "config", err), 1, nil)
	}

	if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error loading config: %v", err), 1, &opts.Logger)
	}
}

func loadDefaultConfigFiles(fetcher remotefetcher.RemoteFetcher, configFiles []string, k *koanf.Koanf, opts *ConfigOpts) {
	cFileFailures := 0
	for _, c := range configFiles {
		opts.ConfigFilePath = c
		data, err := fetcher.Fetch(c)
		if err != nil {
			cFileFailures++
			continue
		}

		if data != nil {
			if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err == nil {
				break
			} else {
				logging.ExitWithMSG(fmt.Sprintf("error loading config from file %s: %v", c, err), 1, &opts.Logger)
			}
		}
	}

	if cFileFailures == len(configFiles) {
		logging.ExitWithMSG("Could not find any valid local config file", 1, nil)
	}
}

func setTerminalEnv() {
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		os.Setenv("BACKY_TERM", "enabled")
	} else {
		os.Setenv("BACKY_TERM", "disabled")
	}
}

func validateExecCommandsFromCLI(k *koanf.Koanf, opts *ConfigOpts) {
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
	opts.LogFilePath = logFile

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
		logging.ExitWithMSG(fmt.Sprintf("error unmarshaling key %s into struct: %v", key, err), 1, &log)
	}
}

func getCommandEnvironments(opts *ConfigOpts) {
	for cmdName, cmdConf := range opts.Cmds {
		if cmdConf.Env == "" {
			continue
		}
		opts.Logger.Debug().Str("env file", cmdConf.Env).Str("cmd", cmdName).Send()
		if err := testFile(cmdConf.Env); err != nil {
			logging.ExitWithMSG("Could not open file"+cmdConf.Env+": "+err.Error(), 1, &opts.Logger)
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

func getConfigDir(opts *ConfigOpts) {
	if isRemoteURL(opts.ConfigFilePath) {
		p, _ := getRemoteDir(opts.ConfigFilePath)
		opts.ConfigDir = p
	} else {
		opts.ConfigDir = path.Dir(opts.ConfigFilePath)
	}
}

func loadCommandLists(opts *ConfigOpts, backyKoanf *koanf.Koanf) {
	var listConfigFiles []string
	var u *url.URL
	var p string
	if isRemoteURL(opts.ConfigFilePath) {
		p, u = getRemoteDir(opts.ConfigFilePath)
		opts.ConfigDir = p
		listConfigFiles = []string{u.JoinPath("lists.yml").String(), u.JoinPath("lists.yaml").String()}
	} else {
		opts.ConfigDir = path.Dir(opts.ConfigFilePath)
		listConfigFiles = []string{
			// "./lists.yml", "./lists.yaml",
			path.Join(opts.ConfigDir, "lists.yml"),
			path.Join(opts.ConfigDir, "lists.yaml"),
		}
	}

	listsConfig := koanf.New(".")

	for _, l := range listConfigFiles {
		if loadListConfigFile(l, listsConfig, opts) {
			break
		}
	}

	if backyKoanf.Exists("cmdLists") {
		if backyKoanf.Exists("cmdLists.file") {
			loadCmdListsFile(backyKoanf, listsConfig, opts)
		} else {
			unmarshalConfig(backyKoanf, "cmdLists", &opts.CmdConfigLists, opts.Logger)
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
	fetcher, err := remotefetcher.NewRemoteFetcher(filePath, opts.Cache, remotefetcher.IgnoreFileNotFound())
	if err != nil {
		// if file not found, ignore
		if errors.Is(err, remotefetcher.ErrIgnoreFileNotFound) {
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

	unmarshalConfig(k, "cmdLists", &opts.CmdConfigLists, opts.Logger)
	keyNotSupported("cmd-lists", "cmdLists", k, opts, true)
	opts.CmdListFile = filePath
	return true
}

func loadCmdListsFile(backyKoanf *koanf.Koanf, listsConfig *koanf.Koanf, opts *ConfigOpts) {
	opts.CmdListFile = strings.TrimSpace(backyKoanf.String("cmdLists.file"))
	if !path.IsAbs(opts.CmdListFile) {
		opts.CmdListFile = path.Join(path.Dir(opts.ConfigFilePath), opts.CmdListFile)
	}

	fetcher, err := remotefetcher.NewRemoteFetcher(opts.CmdListFile, opts.Cache)

	if err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error initializing config fetcher: %v", err), 1, nil)
	}

	data, err := fetcher.Fetch(opts.CmdListFile)
	if err != nil {
		logging.ExitWithMSG(generateFileFetchErrorString(opts.CmdListFile, "list config", err), 1, nil)
	}

	if err := listsConfig.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		logging.ExitWithMSG(fmt.Sprintf("error loading config: %v", err), 1, &opts.Logger)
	}

	keyNotSupported("cmd-lists", "cmdLists", listsConfig, opts, true)
	unmarshalConfig(listsConfig, "cmdLists", &opts.CmdConfigLists, opts.Logger)
	opts.Logger.Info().Str("using lists config file", opts.CmdListFile).Send()
}

func generateFileFetchErrorString(file, fileType string, err error) string {
	return fmt.Sprintf("Could not fetch %s file %s: %v", file, fileType, err)
}

func validateCommandLists(opts *ConfigOpts) {
	var cmdNotFoundSliceErr []error
	for cmdListName, cmdList := range opts.CmdConfigLists {
		// if cron is enabled and cron is not set, delete the list
		if opts.cronEnabled && strings.TrimSpace(cmdList.Cron) == "" {
			opts.Logger.Debug().Str("cron", "enabled").Str("list", cmdListName).Msg("cron not set, deleting list")
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

// func getCmdListFromConfig(list string) string {
// 	return fmt.Sprintf("cmdLists.%s", list)
// }

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
		logging.ExitWithMSG(fmt.Sprintf("error unmarshaling vault.keys into struct: %v", unmarshalErr), 1, &opts.Logger)
	}

	opts.vaultClient = client

	return nil
}

func processCmds(opts *ConfigOpts) error {

	// process commands
	for cmdName, cmd := range opts.Cmds {
		for i, v := range cmd.Args {
			v = replaceVarInString(opts.Vars, v, opts.Logger)
			cmd.Args[i] = v
		}
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
			cmdHost := replaceVarInString(opts.Vars, *cmd.Host, opts.Logger)
			if cmdHost != *cmd.Host {
				cmd.Host = &cmdHost
			}
			host, hostFound := opts.Hosts[*cmd.Host]
			if hostFound {
				cmd.RemoteHost = host
				cmd.RemoteHost.Host = host.Host
				if host.HostName != "" {
					cmd.RemoteHost.HostName = host.HostName
				}
			} else {
				opts.Logger.Info().Msgf("adding host %s to host list", *cmd.Host)
				if opts.Hosts == nil {
					opts.Hosts = make(map[string]*Host)
				}
				opts.Hosts[*cmd.Host] = &Host{Host: *cmd.Host}
				cmd.RemoteHost = &Host{Host: *cmd.Host}
			}
		} else {

			if cmd.Dir != nil {

				cmdDir, err := getFullPathWithHomeDir(*cmd.Dir)
				if err != nil {
					return err
				}
				cmd.Dir = &cmdDir
			} else {
				cmd.Dir = &opts.ConfigDir
			}
		}

		if cmd.Type == PackageCT {
			if cmd.PackageManager == "" {
				return fmt.Errorf("package manager is required for package command %s", cmd.PackageName)
			}
			if cmd.PackageOperation.String() == "" {
				return fmt.Errorf("package operation is required for package command %s", cmd.PackageName)
			}
			if cmd.PackageName == "" {
				return fmt.Errorf("package name is required for package command %s", cmd.PackageName)
			}
			var err error

			// Validate the operation
			if cmd.PackageOperation.IsAPackageOperation() {

				cmd.pkgMan, err = pkgman.PackageManagerFactory(cmd.PackageManager, pkgman.WithoutAuth())
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unsupported package operation %s for command %s", cmd.PackageOperation, cmd.Name)
			}

		}

		// Parse user commands
		if cmd.Type == UserCT {
			if cmd.Username == "" {
				return fmt.Errorf("username is required for user command %s", cmd.Name)
			}
			cmd.Username = replaceVarInString(opts.Vars, cmd.Username, opts.Logger)
			err := detectOSType(cmd, opts)
			if err != nil {
				opts.Logger.Info().Err(err).Str("command", cmdName).Send()
			}

			// Validate the operation
			switch cmd.UserOperation {
			case "add", "remove", "modify", "checkIfExists", "delete", "password":
				cmd.userMan, err = usermanager.NewUserManager(cmd.OS)

				if cmd.UserOperation == "password" {
					opts.Logger.Debug().Msg("changing password for user: " + cmd.Username)
					cmd.UserPassword = getExternalConfigDirectiveValue(cmd.UserPassword, opts)
				}
				if cmd.Host != nil {
					host, ok := opts.Hosts[*cmd.Host]
					if ok {
						cmd.userMan, err = usermanager.NewUserManager(host.OS)
					}
				}
				for indx, key := range cmd.UserSshPubKeys {
					opts.Logger.Debug().Msg("adding SSH Keys")
					key = getExternalConfigDirectiveValue(key, opts)
					cmd.UserSshPubKeys[indx] = key
				}
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported user operation %s for command %s", cmd.UserOperation, cmd.Name)
			}

		}

		if cmd.Type == RemoteScriptCT {
			var fetchErr error
			if !isRemoteURL(cmd.Cmd) {
				return fmt.Errorf("remoteScript command %s must be a remote resource", cmdName)
			}
			cmd.Fetcher, fetchErr = remotefetcher.NewRemoteFetcher(cmd.Cmd, opts.Cache, remotefetcher.WithFileType("script"))
			if fetchErr != nil {
				return fmt.Errorf("error initializing remote fetcher for remoteScript: %v", fetchErr)
			}

		}
		if cmd.OutputFile != "" {
			var err error
			cmd.OutputFile, err = getFullPathWithHomeDir(cmd.OutputFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func processHooks(cmd *Command, hooks []string, opts *ConfigOpts, hookType string) error {

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
		if runtime.GOOS == "linux" {
			cmd.OS = "linux"
			opts.Logger.Info().Msg("Unix/Linux type OS detected")
			return nil
		}
		return fmt.Errorf("using an os that is not yet supported for user commands")
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

func keyNotSupported(oldKey, newKey string, koanf *koanf.Koanf, opts *ConfigOpts, deprecated bool) {

	if koanf.Exists(oldKey) {
		if deprecated {
			opts.Logger.Warn().Str("key", oldKey).Msg("key is deprecated. Use " + newKey + " instead.")
		} else {
			opts.Logger.Fatal().Err(fmt.Errorf("key %s found; it has changed to %s", oldKey, newKey)).Send()
		}
	}
}

func replaceVarInString(vars map[string]string, str string, logger zerolog.Logger) string {
	if strings.Contains(str, "%{var:") && strings.Contains(str, "}%") {
		logger.Debug().Msgf("replacing vars in string %s", str)
		for k, v := range vars {
			if strings.Contains(str, "%{var:"+k+"}%") {
				str = strings.ReplaceAll(str, "%{var:"+k+"}%", v)
			}
		}
		if strings.Contains(str, "%{var:") && strings.Contains(str, "}%") {
			logger.Warn().Msg("could not replace all vars in string")
		}
	}
	return str
}
