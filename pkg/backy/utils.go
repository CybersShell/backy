// utils.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"git.andrewnw.xyz/CyberShell/backy/pkg/remotefetcher"
	vault "github.com/hashicorp/vault/api"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"mvdan.cc/sh/v3/shell"
)

func (c *ConfigOpts) LogLvl(level string) BackyOptionFunc {

	return func(bco *ConfigOpts) {
		c.BackyLogLvl = &level
	}
}

// AddCommands adds commands to ConfigOpts
func AddCommands(commands []string) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.executeCmds = append(bco.executeCmds, commands...)
	}
}

// AddCommandLists adds lists to ConfigOpts
func AddCommandLists(lists []string) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.executeLists = append(bco.executeLists, lists...)
	}
}

// SetListsToSearch adds lists to search
func SetListsToSearch(lists []string) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.List.Lists = append(bco.List.Lists, lists...)
	}
}

// AddPrintLists adds lists to print out
func SetCmdsToSearch(cmds []string) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.List.Commands = append(bco.List.Commands, cmds...)
	}
}

// SetLogFile sets the path to the log file
func SetLogFile(logFile string) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.LogFilePath = logFile
	}
}

func SetHostsConfigFile(hostsConfigFile string) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.HostsFilePath = hostsConfigFile
	}
}

// EnableCommandStdOut forces the command output to stdout
func EnableCommandStdOut(setStdOut bool) BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.CmdStdOut = setStdOut
	}
}

// EnableCron enables the execution of command lists at specified times
func EnableCron() BackyOptionFunc {
	return func(bco *ConfigOpts) {
		bco.cronEnabled = true
	}
}

func NewConfigOptions(configFilePath string, opts ...BackyOptionFunc) *ConfigOpts {
	b := &ConfigOpts{}
	b.ConfigFilePath = configFilePath
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func injectEnvIntoSSH(envVarsToInject environmentVars, session *ssh.Session, opts *ConfigOpts, log zerolog.Logger) error {
	if envVarsToInject.file != "" {
		envPath, envPathErr := getFullPathWithHomeDir(envVarsToInject.file)
		if envPathErr != nil {
			log.Fatal().Str("envFile", envPath).Err(envPathErr).Send()
		}
		file, err := os.Open(envPath)
		if err != nil {
			log.Fatal().Str("envFile", envPath).Err(err).Send()
		}
		defer file.Close()

		envMap, err := godotenv.Parse(file)
		if err != nil {
			log.Fatal().Str("envFile", envPath).Err(err).Send()
		}
		for key, val := range envMap {
			err = session.Setenv(key, getExternalConfigDirectiveValue(val, opts, AllowedExternalDirectiveVault))
			if err != nil {
				log.Info().Err(err).Send()
				return fmt.Errorf("failed to set environment variable %s: %w", val, err)
			}
		}
	}

	// fmt.Printf("%v", envVarsToInject.env)
	for _, envVal := range envVarsToInject.env {
		// don't append env Vars for Backy
		if strings.Contains(envVal, "=") {
			envVarArr := strings.Split(envVal, "=")

			err := session.Setenv(envVarArr[0], getExternalConfigDirectiveValue(envVarArr[1], opts, AllowedExternalDirectiveVaultFile))
			if err != nil {
				log.Info().Err(err).Send()
				return fmt.Errorf("failed to set environment variable %s: %w", envVarArr[1], err)
			}
		}
	}
	return nil
}

func injectEnvIntoLocalCMD(envVarsToInject environmentVars, process *exec.Cmd, log zerolog.Logger, opts *ConfigOpts) {
	if envVarsToInject.file != "" {
		envPath, _ := getFullPathWithHomeDir(envVarsToInject.file)

		file, fileErr := os.Open(envPath)
		if fileErr != nil {
			log.Error().Str("envFile", envPath).Err(fileErr).Send()
			goto errEnvFile
		}
		defer file.Close()
		envMap, err := godotenv.Parse(file)
		if err != nil {
			log.Error().Str("envFile", envPath).Err(err).Send()
			goto errEnvFile
		}
		for key, val := range envMap {
			process.Env = append(process.Env, fmt.Sprintf("%s=%s", key, val))
		}

	}
errEnvFile:

	for _, envVal := range envVarsToInject.env {
		if strings.Contains(envVal, "=") {
			envVarArr := strings.Split(envVal, "=")
			process.Env = append(process.Env, fmt.Sprintf("%s=%s", envVarArr[0], getExternalConfigDirectiveValue(envVarArr[1], opts, AllowedExternalDirectiveVault)))
		}
	}
	process.Env = append(process.Env, os.Environ()...)
}

func prependEnvVarsToCommand(envVars environmentVars, opts *ConfigOpts, command string, args []string, cmdCtxLogger zerolog.Logger) string {
	var envPrefix string
	if envVars.file != "" {
		envPath, envPathErr := getFullPathWithHomeDir(envVars.file)
		if envPathErr != nil {
			cmdCtxLogger.Fatal().Str("envFile", envPath).Err(envPathErr).Send()
		}
		file, err := os.Open(envPath)
		if err != nil {
			log.Fatal().Str("envFile", envPath).Err(err).Send()
		}
		defer file.Close()

		envMap, err := godotenv.Parse(file)
		if err != nil {
			log.Fatal().Str("envFile", envPath).Err(err).Send()
		}
		for key, val := range envMap {
			envPrefix += fmt.Sprintf("%s=%s ", key, getExternalConfigDirectiveValue(val, opts, AllowedExternalDirectiveVaultEnv))
		}
	}
	for _, value := range envVars.env {
		envVarArr := strings.Split(value, "=")
		envPrefix += fmt.Sprintf("%s=%s ", envVarArr[0], getExternalConfigDirectiveValue(envVarArr[1], opts, AllowedExternalDirectiveVault))
		envPrefix += "\n"
	}
	return envPrefix + command + " " + strings.Join(args, " ")
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func CheckConfigValues(config *koanf.Koanf, file string) {

	for _, key := range requiredKeys {
		isKeySet := config.Exists(key)
		if !isKeySet {
			logging.ExitWithMSG(Sprintf("Config key %s is not defined in %s. Please make sure this value is set and has the appropriate keys set.", key, file), 1, nil)
		}
	}
}

func testFile(c string) error {
	if strings.TrimSpace(c) != "" {
		file, fileOpenErr := os.Open(c)
		file.Close()
		if errors.Is(fileOpenErr, os.ErrNotExist) {
			return fileOpenErr
		}
	}
	return nil
}

func IsTerminalActive() bool {
	return os.Getenv("BACKY_TERM") == "enabled"
}

func IsCmdStdOutEnabled() bool {
	return os.Getenv("BACKY_CMDSTDOUT") == "enabled"
}

func getFullPathWithHomeDir(path string) (string, error) {
	path = strings.TrimSpace(path)

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path, err
		}
		// In case of "~", which won't be caught by the "else if"
		path = homeDir
	} else if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path, err
		}
		// Use strings.HasPrefix so we don't match paths like
		// "/something/~/something/"
		path = filepath.Join(homeDir, path[2:])
	}
	return path, nil
}

// loadEnv loads a .env file from the config file directory
func (opts *ConfigOpts) loadEnv() {
	var backyEnv map[string]string
	var envFileInConfigDir string
	var envFileErr error
	if isRemoteURL(opts.ConfigFilePath) {
		_, u := getRemoteDir(opts.ConfigFilePath)
		envFileInConfigDir = u.JoinPath(".env").String()
		envFetcher, err := remotefetcher.NewRemoteFetcher(envFileInConfigDir, opts.Cache)
		if err != nil {
			return
		}
		data, err := envFetcher.Fetch(envFileInConfigDir)
		if err != nil {
			return
		}
		backyEnv, envFileErr = godotenv.UnmarshalBytes(data)
		if envFileErr != nil {
			return
		}

	} else {
		envFileInConfigDir = fmt.Sprintf("%s/.env", path.Dir(opts.ConfigFilePath))
		backyEnv, envFileErr = godotenv.Read(envFileInConfigDir)
		if envFileErr != nil {
			return
		}
	}

	opts.backyEnv = backyEnv
}

func expandEnvVars(backyEnv map[string]string, envVars []string) {

	env := func(name string) string {
		envVar, found := backyEnv[name]
		if found {
			return envVar
		}
		return ""
	}

	for indx, v := range envVars {

		if strings.HasPrefix(v, envExternDirectiveStart) && strings.HasSuffix(v, externDirectiveEnd) {
			v = strings.TrimPrefix(v, envExternDirectiveStart)
			v = strings.TrimRight(v, externDirectiveEnd)
			out, _ := shell.Expand(v, env)
			envVars[indx] = out
		}

	}
}

func getCommandTypeAndSetCommandInfo(command *Command) *Command {

	if command.Type == PackageCommandType && !command.packageCmdSet {
		command.packageCmdSet = true
		switch command.PackageOperation {
		case PackageOperationInstall:
			command.Cmd, command.Args = command.pkgMan.Install(command.Packages, command.Args)
		case PackageOperationRemove:
			command.Cmd, command.Args = command.pkgMan.Remove(command.Packages, command.Args)
		case PackageOperationUpgrade:
			command.Cmd, command.Args = command.pkgMan.Upgrade(command.Packages)
		case PackageOperationCheckVersion:
			command.Cmd, command.Args = command.pkgMan.CheckVersion(command.Packages)
		}
	}

	if command.Type == UserCommandType && !command.userCmdSet {
		command.userCmdSet = true
		switch command.UserOperation {
		case "add":
			command.Cmd, command.Args = command.userMan.AddUser(
				command.Username,
				command.UserHome,
				command.UserShell,
				command.UserIsSystem,
				command.UserCreateHome,
				command.UserGroups,
				command.Args)
		case "modify":
			command.Cmd, command.Args = command.userMan.ModifyUser(
				command.Username,
				command.UserHome,
				command.UserShell,
				command.UserGroups)
		case "checkIfExists":
			command.Cmd, command.Args = command.userMan.UserExists(command.Username)
		case "delete":
			command.Cmd, command.Args = command.userMan.RemoveUser(command.Username)
		case "password":
			command.Cmd, command.stdin, command.UserPassword = command.userMan.ModifyPassword(command.Username, command.UserPassword)
		}
	}

	return command
}

func parsePackageVersion(output string, cmdCtxLogger zerolog.Logger, command *Command, cmdOutBuf bytes.Buffer) ([]string, error) {

	var err error
	var errs []error
	pkgVersionOnSystem, err := command.pkgMan.ParseRemotePackageManagerVersionOutput(output)
	if err != nil {
		cmdCtxLogger.Error().AnErr("Error parsing package version output", err).Send()
		return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error parsing package version output: %v", err)
	}

	for _, p := range pkgVersionOnSystem {
		packageIndex := getPackageIndexFromCommand(command, p.Name)
		if packageIndex == -1 {
			cmdCtxLogger.Error().Str("package", p.Name).Msg("Package not found in command")
			continue
		}
		command.Packages[packageIndex].VersionCheck = p.VersionCheck
		packageFromCommand := command.Packages[packageIndex]
		cmdCtxLogger.Info().
			Str("Installed", packageFromCommand.VersionCheck.Installed).
			Msg("Package version comparison")

		versionLogger := cmdCtxLogger.With().Str("package", packageFromCommand.Name).Logger()

		if packageFromCommand.Version != "" {
			versionLogger := cmdCtxLogger.With().Str("package", packageFromCommand.Name).Str("Specified Version", packageFromCommand.Version).Logger()
			packageVersionRegex, PkgRegexErr := regexp.Compile(packageFromCommand.Version)
			if PkgRegexErr != nil {
				versionLogger.Error().Err(PkgRegexErr).Msg("Error compiling package version regex")
				errs = append(errs, PkgRegexErr)
				continue
			}
			if p.Version == packageFromCommand.Version {
				versionLogger.Info().Msgf("Installed version matches specified version: %s", packageFromCommand.Version)
			} else if packageVersionRegex.MatchString(p.VersionCheck.Installed) {
				versionLogger.Info().Msgf("Installed version contains specified version: %s", packageFromCommand.Version)
			} else {
				versionLogger.Info().Msg("Installed version does not match specified version")
				errs = append(errs, fmt.Errorf("installed version of %s does not match specified version: %s", packageFromCommand.Name, packageFromCommand.Version))
			}
		} else {
			if p.VersionCheck.Installed == p.VersionCheck.Candidate {
				versionLogger.Info().Msg("Installed and Candidate versions match")
			} else {
				cmdCtxLogger.Info().Msg("Installed and Candidate versions differ")
				errs = append(errs, errors.New("installed and Candidate versions differ"))
			}
		}
	}
	if errs == nil {
		return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), nil
	}
	return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error parsing package version output: %v", errs)
}

func getPackageIndexFromCommand(command *Command, name string) int {
	for i, v := range command.Packages {
		if name == v.Name {
			return i
		}
	}

	return -1
}

func getExternalConfigDirectiveValue(key string, opts *ConfigOpts, allowedDirectives AllowedExternalDirectives) string {
	if !(strings.HasPrefix(key, externDirectiveStart) && strings.HasSuffix(key, externDirectiveEnd)) {
		return key
	}
	key = replaceVarInString(opts.Vars, key, opts.Logger)
	opts.Logger.Debug().Str("expanding external key", key).Send()

	if newKeyStr, directiveFound := strings.CutPrefix(key, envExternDirectiveStart); directiveFound {
		if IsExternalDirectiveEnv(allowedDirectives) {

			key = strings.TrimSuffix(newKeyStr, externDirectiveEnd)
			key = os.Getenv(key)
		} else {
			opts.Logger.Error().Msgf("Config key with value %s does not support env directive", key)
		}
	}

	if newKeyStr, directiveFound := strings.CutPrefix(key, externFileDirectiveStart); directiveFound {
		if IsExternalDirectiveFile(allowedDirectives) {

			var err error
			var keyValue []byte
			key = strings.TrimSuffix(newKeyStr, externDirectiveEnd)
			key, err = getFullPathWithHomeDir(key)
			if err != nil {
				opts.Logger.Err(err).Send()
				return ""
			}
			if !path.IsAbs(key) {
				key = path.Join(opts.ConfigDir, key)
			}
			keyValue, err = os.ReadFile(key)
			if err != nil {
				opts.Logger.Err(err).Send()
				return ""
			}
			key = string(keyValue)
		} else {
			opts.Logger.Error().Msgf("Config key with value %s does not support file directive", key)
		}
	}

	if newKeyStr, directiveFound := strings.CutPrefix(key, vaultExternDirectiveStart); directiveFound {
		if IsExternalDirectiveVault(allowedDirectives) {

			key = strings.TrimSuffix(newKeyStr, externDirectiveEnd)
			key = GetVaultKey(key, opts, opts.Logger)
		} else {
			opts.Logger.Error().Msgf("Config key with value %s does not support vault directive", key)
		}
	}

	return key
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

	value, ok := secret.Data[key.Key].(string)
	if !ok {
		return "", fmt.Errorf("value type assertion failed for vault key %s: %T %#v", key.Name, secret.Data[key.Name], secret.Data[key.Name])
	}

	return value, nil
}

func getVaultKeyData(keyName string, keys []*VaultKey) (*VaultKey, error) {
	for _, k := range keys {
		if k.Name == keyName {
			return k, nil
		}
	}
	return nil, fmt.Errorf("key %s not found in vault keys", keyName)
}

func GetVaultKey(str string, opts *ConfigOpts, log zerolog.Logger) string {
	key, err := getVaultKeyData(str, opts.VaultKeys)
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

func IsExternalDirectiveFile(allowedExternalDirectives AllowedExternalDirectives) bool {
	return strings.Contains(allowedExternalDirectives.String(), "file")
}

func IsExternalDirectiveEnv(allowedExternalDirectives AllowedExternalDirectives) bool {
	return strings.Contains(allowedExternalDirectives.String(), "env")
}

func IsExternalDirectiveVault(allowedExternalDirectives AllowedExternalDirectives) bool {
	return strings.Contains(allowedExternalDirectives.String(), "vault")
}
