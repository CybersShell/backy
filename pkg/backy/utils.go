// utils.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"mvdan.cc/sh/v3/shell"
)

func injectEnvIntoSSH(envVarsToInject environmentVars, process *ssh.Session, opts *BackyConfigOpts) {
	if envVarsToInject.file != "" {
		envPath, envPathErr := resolveDir(envVarsToInject.file)
		if envPathErr != nil {
			opts.ConfigFile.Logger.Fatal().Str("envFile", envPath).Err(envPathErr).Send()
		}
		file, err := os.Open(envPath)
		if err != nil {
			opts.ConfigFile.Logger.Fatal().Str("envFile", envPath).Err(err).Send()
		}
		defer file.Close()

		envMap, err := godotenv.Parse(file)
		if err != nil {
			opts.ConfigFile.Logger.Error().Str("envFile", envPath).Err(err).Send()
			goto errEnvFile
		}
		for key, val := range envMap {
			process.Setenv(key, GetVaultKey(val, opts))
		}
	}

errEnvFile:
	// fmt.Printf("%v", envVarsToInject.env)
	for _, envVal := range envVarsToInject.env {
		// don't append env Vars for Backy
		if strings.Contains(envVal, "=") {
			envVarArr := strings.Split(envVal, "=")

			process.Setenv(envVarArr[0], GetVaultKey(envVarArr[1], opts))
		}
	}
}

func injectEnvIntoLocalCMD(envVarsToInject environmentVars, process *exec.Cmd, log *zerolog.Logger) {
	if envVarsToInject.file != "" {
		envPath, _ := resolveDir(envVarsToInject.file)

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

func CheckConfigValues(config *viper.Viper) {

	for _, key := range requiredKeys {
		isKeySet := config.IsSet(key)
		if !isKeySet {
			logging.ExitWithMSG(Sprintf("Config key %s is not defined in %s. Please make sure this value is set and has the appropriate keys set.", key, config.ConfigFileUsed()), 1, nil)
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

func (c *BackyConfigOpts) LogLvl(level string) BackyOptionFunc {

	return func(bco *BackyConfigOpts) {
		c.BackyLogLvl = &level
	}
}

// AddCommands adds commands to BackyConfigOpts
func AddCommands(commands []string) BackyOptionFunc {
	return func(bco *BackyConfigOpts) {
		bco.executeCmds = append(bco.executeCmds, commands...)
	}
}

// AddCommandLists adds lists to BackyConfigOpts
func AddCommandLists(lists []string) BackyOptionFunc {
	return func(bco *BackyConfigOpts) {
		bco.executeLists = append(bco.executeLists, lists...)
	}
}

// UseCron enables the execution of command lists at specified times
func UseCron() BackyOptionFunc {
	return func(bco *BackyConfigOpts) {
		bco.useCron = true
	}
}

// UseCron enables the execution of command lists at specified times
func (c *BackyConfigOpts) AddViper(v *viper.Viper) BackyOptionFunc {
	return func(bco *BackyConfigOpts) {
		c.viper = v
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
		CmdConfigLists: make(map[string]*CmdList),
		Hosts:          make(map[string]*Host),
		Notifications:  make(map[string]*NotificationsConfig),
	}
}

func IsTerminalActive() bool {
	return os.Getenv("BACKY_TERM") == "enabled"
}

func IsCmdStdOutEnabled() bool {
	return os.Getenv("BACKY_STDOUT") == "enabled"
}

func resolveDir(path string) (string, error) {
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

func (opts *BackyConfigOpts) loadEnv() {
	envFileInConfigDir := fmt.Sprintf("%s/.env", path.Dir(opts.viper.ConfigFileUsed()))
	var backyEnv map[string]string
	backyEnv, envFileErr := godotenv.Read(envFileInConfigDir)
	if envFileErr != nil {
		return
	}

	opts.backyEnv = backyEnv
}

func expandEnvVars(backyEnv map[string]string, envVars []string) {

	env := func(name string) string {
		name = strings.ToUpper(name)
		envVar, found := backyEnv[name]
		if found {
			return envVar
		}
		return ""
	}

	for indx, v := range envVars {
		if strings.Contains(v, "$") || (strings.Contains(v, "${") && strings.Contains(v, "}")) {
			out, _ := shell.Expand(v, env)
			envVars[indx] = out
		}
	}
}
