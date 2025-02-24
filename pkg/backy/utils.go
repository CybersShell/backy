// utils.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
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

// SetCmdStdOut forces the command output to stdout
func SetCmdStdOut(setStdOut bool) BackyOptionFunc {
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

func NewOpts(configFilePath string, opts ...BackyOptionFunc) *ConfigOpts {
	b := &ConfigOpts{}
	b.ConfigFilePath = configFilePath
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func injectEnvIntoSSH(envVarsToInject environmentVars, process *ssh.Session, opts *ConfigOpts, log zerolog.Logger) {
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
			log.Error().Str("envFile", envPath).Err(err).Send()
			goto errEnvFile
		}
		for key, val := range envMap {
			process.Setenv(key, GetVaultKey(val, opts, log))
		}
	}

errEnvFile:
	// fmt.Printf("%v", envVarsToInject.env)
	for _, envVal := range envVarsToInject.env {
		// don't append env Vars for Backy
		if strings.Contains(envVal, "=") {
			envVarArr := strings.Split(envVal, "=")

			process.Setenv(envVarArr[0], GetVaultKey(envVarArr[1], opts, log))
		}
	}
}

func injectEnvIntoLocalCMD(envVarsToInject environmentVars, process *exec.Cmd, log zerolog.Logger) {
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
			process.Env = append(process.Env, envVal)
		}
	}
	process.Env = append(process.Env, os.Environ()...)
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
	envFileInConfigDir := fmt.Sprintf("%s/.env", path.Dir(opts.ConfigFilePath))
	var backyEnv map[string]string
	backyEnv, envFileErr := godotenv.Read(envFileInConfigDir)
	if envFileErr != nil {
		return
	}

	opts.backyEnv = backyEnv
}

// expandEnvVars expands environment variables with the env used in the config
func expandEnvVars(backyEnv map[string]string, envVars []string) {

	env := func(name string) string {
		name = strings.ToUpper(name)
		envVar, found := backyEnv[name]
		if found {
			return envVar
		}
		return ""
	}

	// parse env variables using new macros
	for indx, v := range envVars {
		if strings.HasPrefix(v, macroStart) && strings.HasSuffix(v, macroEnd) {
			if strings.HasPrefix(v, envMacroStart) {
				v = strings.TrimPrefix(v, envMacroStart)
				v = strings.TrimRight(v, macroEnd)
				out, _ := shell.Expand(v, env)
				envVars[indx] = out
			}
		}
	}
}

// getCommandTypeAndSetCommandInfo checks for command type and if the command has already been set
// Checks for types package and user
// Returns the modified Command with the package- or userManager command as Cmd and the package- or userOperation as args, plus any additional Args
func getCommandTypeAndSetCommandInfo(command *Command) *Command {

	if command.Type == PackageCT && !command.packageCmdSet {
		command.packageCmdSet = true
		switch command.PackageOperation {
		case "install":
			command.Cmd, command.Args = command.pkgMan.Install(command.PackageName, command.PackageVersion, command.Args)
		case "remove":
			command.Cmd, command.Args = command.pkgMan.Remove(command.PackageName, command.Args)
		case "upgrade":
			command.Cmd, command.Args = command.pkgMan.Upgrade(command.PackageName, command.PackageVersion)
		case "checkVersion":
			command.Cmd, command.Args = command.pkgMan.CheckVersion(command.PackageName, command.PackageVersion)
		}
	}

	if command.Type == UserCT && !command.userCmdSet {
		command.userCmdSet = true
		switch command.UserOperation {
		case "add":
			command.Cmd, command.Args = command.userMan.AddUser(
				command.Username,
				command.UserHome,
				command.UserShell,
				command.SystemUser,
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
	pkgVersion, err := command.pkgMan.Parse(output)
	// println(output)
	if err != nil {
		cmdCtxLogger.Error().Err(err).Str("package", command.PackageName).Msg("Error parsing package version output")
		return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.GetOutput), err
	}

	cmdCtxLogger.Info().
		Str("Installed", pkgVersion.Installed).
		Str("Candidate", pkgVersion.Candidate).
		Msg("Package version comparison")

	if command.PackageVersion != "" {
		if pkgVersion.Installed == command.PackageVersion {
			cmdCtxLogger.Info().Msgf("Installed version matches specified version: %s", command.PackageVersion)
		} else {
			cmdCtxLogger.Info().Msgf("Installed version does not match specified version: %s", command.PackageVersion)
			err = fmt.Errorf("Installed version does not match specified version: %s", command.PackageVersion)
		}
	} else {
		if pkgVersion.Installed == pkgVersion.Candidate {
			cmdCtxLogger.Info().Msg("Installed and Candidate versions match")
		} else {
			cmdCtxLogger.Info().Msg("Installed and Candidate versions differ")
			err = errors.New("Installed and Candidate versions differ")
		}
	}
	return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, false), err
}
