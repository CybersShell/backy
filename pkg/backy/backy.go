// backy.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0
package backy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog"
)

var requiredKeys = []string{"commands", "cmd-configs", "logging"}

var Sprintf = fmt.Sprintf

// RunCmd runs a Command.
// The environment of local commands will be the machine's environment plus any extra
// variables specified in the Env file or Environment.
// Dir can also be specified for local commands.
func (command *Command) RunCmd(log *zerolog.Logger) error {

	var (
		ArgsStr       string
		cmdOutBuf     bytes.Buffer
		cmdOutWriters io.Writer

		envVars = environmentVars{
			file: command.Env,
			env:  command.Environment,
		}
	)
	envVars.env = append(envVars.env, os.Environ()...)

	for _, v := range command.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	if command.Host != nil {
		log.Info().Str("Command", fmt.Sprintf("Running command: %s %s on host %s", command.Cmd, ArgsStr, *command.Host)).Send()

		sshc, err := command.RemoteHost.ConnectToSSHHost(log)
		if err != nil {
			return err
		}
		defer sshc.Close()
		commandSession, err := sshc.NewSession()
		if err != nil {
			log.Err(fmt.Errorf("new ssh session: %w", err)).Send()
			return err
		}
		defer commandSession.Close()

		injectEnvIntoSSH(envVars, commandSession, log)
		cmd := command.Cmd
		for _, a := range command.Args {
			cmd += " " + a
		}
		cmdOutWriters = io.MultiWriter(&cmdOutBuf)

		if IsCmdStdOutEnabled() {
			cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
		}

		commandSession.Stdout = cmdOutWriters
		commandSession.Stderr = cmdOutWriters
		err = commandSession.Run(cmd)
		outScanner := bufio.NewScanner(&cmdOutBuf)
		for outScanner.Scan() {
			outMap := make(map[string]interface{})
			outMap["cmd"] = cmd
			outMap["output"] = outScanner.Text()
			log.Info().Fields(outMap).Send()
		}

		if err != nil {
			log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
			return err
		}
	} else {
		cmdExists := command.checkCmdExists()
		if !cmdExists {
			log.Info().Str(command.Cmd, "not found").Send()
		}

		var err error
		if command.Shell != "" {
			log.Info().Str("Command", fmt.Sprintf("Running command: %s %s on local machine in %s", command.Cmd, ArgsStr, command.Shell)).Send()
			ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)
			localCMD := exec.Command(command.Shell, "-c", ArgsStr)
			if command.Dir != nil {
				localCMD.Dir = *command.Dir
			}
			injectEnvIntoLocalCMD(envVars, localCMD, log)

			cmdOutWriters = io.MultiWriter(&cmdOutBuf)

			if IsCmdStdOutEnabled() {
				cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
			}

			localCMD.Stdout = cmdOutWriters
			localCMD.Stderr = cmdOutWriters
			err = localCMD.Run()
			outScanner := bufio.NewScanner(&cmdOutBuf)
			for outScanner.Scan() {
				outMap := make(map[string]interface{})
				outMap["cmd"] = command.Cmd
				outMap["output"] = outScanner.Text()
				log.Info().Fields(outMap).Send()
			}

			if err != nil {
				log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
				return err
			}
			return nil
		}
		log.Info().Str("Command", fmt.Sprintf("Running command: %s %s on local machine", command.Cmd, ArgsStr)).Send()

		localCMD := exec.Command(command.Cmd, command.Args...)
		if command.Dir != nil {
			localCMD.Dir = *command.Dir
		}
		injectEnvIntoLocalCMD(envVars, localCMD, log)
		cmdOutWriters = io.MultiWriter(&cmdOutBuf)

		if IsCmdStdOutEnabled() {
			cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
		}
		localCMD.Stdout = cmdOutWriters
		localCMD.Stderr = cmdOutWriters
		err = localCMD.Run()
		outScanner := bufio.NewScanner(&cmdOutBuf)
		for outScanner.Scan() {
			outMap := make(map[string]interface{})
			outMap["cmd"] = command.Cmd
			outMap["output"] = outScanner.Text()
			log.Info().Fields(outMap).Send()
		}
		if err != nil {
			log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
			return err
		}
	}
	return nil
}

func cmdListWorker(id int, jobs <-chan *CmdList, config *BackyConfigFile, results chan<- string) {
	for list := range jobs {
		var currentCmd string
		fieldsMap := make(map[string]interface{})
		fieldsMap["list"] = list.Name
		cmdLog := config.Logger.Info()
		var count int
		var Msg string
		for _, cmd := range list.Order {
			currentCmd = config.Cmds[cmd].Cmd
			fieldsMap["cmd"] = config.Cmds[cmd].Cmd
			cmdLog.Fields(fieldsMap).Send()
			cmdToRun := config.Cmds[cmd]
			cmdLogger := config.Logger.With().
				Str("backy-cmd", cmd).
				Logger()
			runOutErr := cmdToRun.RunCmd(&cmdLogger)
			count++
			if runOutErr != nil {
				if list.NotifyConfig != nil {
					notifySendErr := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s failed on command %s ", list.Name, cmd),
						fmt.Sprintf("List %s failed on command %s running command %s. \n Error: %v", list.Name, cmd, currentCmd, runOutErr))
					if notifySendErr != nil {
						config.Logger.Err(notifySendErr).Send()
					}
				}
				config.Logger.Err(runOutErr).Send()
				break
			} else {

				if count == len(list.Order) {
					Msg += fmt.Sprintf("%s ", cmd)
					if list.NotifyConfig != nil {
						err := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s succeded", list.Name),
							fmt.Sprintf("Command list %s was completed successfully. The following commands ran:\n %s", list.Name, Msg))
						if err != nil {
							config.Logger.Err(err).Send()
						}
					}
				} else {
					Msg += fmt.Sprintf("%s, ", cmd)
				}
			}
		}

		results <- "done"
	}
}

// RunBackyConfig runs a command list from the BackyConfigFile.
func (config *BackyConfigFile) RunBackyConfig(cron string) {
	configListsLen := len(config.CmdConfigLists)
	listChan := make(chan *CmdList, configListsLen)
	results := make(chan string)

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= 3; w++ {
		go cmdListWorker(w, listChan, config, results)

	}

	// Here we send 5 `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	// configChan <- config.Cmds
	for _, cmdConfig := range config.CmdConfigLists {
		if cron != "" {
			if cron == cmdConfig.Cron {
				listChan <- cmdConfig
			}
		} else {
			listChan <- cmdConfig
		}
	}
	close(listChan)

	for a := 1; a <= configListsLen; a++ {
		<-results
	}

}

func (config *BackyConfigFile) ExecuteCmds() {
	for _, cmd := range config.Cmds {
		cmd.RunCmd(&config.Logger)
	}
}
