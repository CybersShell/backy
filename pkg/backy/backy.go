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
	"text/template"

	"embed"

	"github.com/rs/zerolog"
)

//go:embed templates/*.txt
var templates embed.FS

var requiredKeys = []string{"commands", "cmd-configs"}

var Sprintf = fmt.Sprintf

// RunCmd runs a Command.
// The environment of local commands will be the machine's environment plus any extra
// variables specified in the Env file or Environment.
// Dir can also be specified for local commands.
func (command *Command) RunCmd(log *zerolog.Logger, backyConf *BackyConfigFile) ([]string, error) {

	var (
		outputArr     []string
		ArgsStr       string
		cmdOutBuf     bytes.Buffer
		cmdOutWriters io.Writer

		envVars = environmentVars{
			file: command.Env,
			env:  command.Environment,
		}
	)

	for _, v := range command.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	if command.Host != nil {
		log.Info().Str("Command", fmt.Sprintf("Running command %s %s on host %s", command.Cmd, ArgsStr, *command.Host)).Send()

		if command.RemoteHost.SshClient == nil {
			err := command.RemoteHost.ConnectToSSHHost(log, backyConf)
			if err != nil {
				return nil, err
			}
		}
		commandSession, err := command.RemoteHost.SshClient.NewSession()
		if err != nil {
			log.Err(fmt.Errorf("new ssh session: %w", err)).Send()
			return nil, err
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
			if str, ok := outMap["output"].(string); ok {
				outputArr = append(outputArr, str)
			}
			log.Info().Fields(outMap).Send()
		}

		if err != nil {
			log.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Cmd, err)).Send()
			return outputArr, err
		}
	} else {
		cmdExists := command.checkCmdExists()
		if !cmdExists {
			log.Info().Str(command.Cmd, "not found").Send()
		}

		var err error
		if command.Shell != "" {
			log.Info().Str("Command", fmt.Sprintf("Running command %s %s on local machine in %s", command.Cmd, ArgsStr, command.Shell)).Send()

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
				if str, ok := outMap["output"].(string); ok {
					outputArr = append(outputArr, str)
				}
				log.Info().Fields(outMap).Send()
			}

			if err != nil {
				log.Error().Err(fmt.Errorf("error when running cmd %s: %w", command.Cmd, err)).Send()
				return outputArr, err
			}
			return outputArr, nil
		}
		log.Info().Str("Command", fmt.Sprintf("Running command %s %s on local machine", command.Cmd, ArgsStr)).Send()

		localCMD := exec.Command(command.Cmd, command.Args...)
		if command.Dir != nil {
			localCMD.Dir = *command.Dir
		}
		// fmt.Printf("%v\n", envVars.env)

		injectEnvIntoLocalCMD(envVars, localCMD, log)
		cmdOutWriters = io.MultiWriter(&cmdOutBuf)
		// fmt.Printf("%v\n", localCMD.Environ())

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

			if str, ok := outMap["output"].(string); ok {
				outputArr = append(outputArr, str)
			}
			log.Info().Fields(outMap).Send()
		}
		if err != nil {
			log.Error().Err(fmt.Errorf("error when running cmd %s: %w", command.Cmd, err)).Send()
			return outputArr, err
		}
	}
	return outputArr, nil
}

func cmdListWorker(msgTemps *msgTemplates, jobs <-chan *CmdList, config *BackyConfigFile, results chan<- string) {

	for list := range jobs {
		var currentCmd string
		fieldsMap := make(map[string]interface{})
		fieldsMap["list"] = list.Name

		cmdLog := config.Logger.Info()

		var count int
		var cmdsRan []string

		for _, cmd := range list.Order {
			currentCmd = config.Cmds[cmd].Cmd

			fieldsMap["cmd"] = config.Cmds[cmd].Cmd
			cmdLog.Fields(fieldsMap).Send()
			cmdToRun := config.Cmds[cmd]

			cmdLogger := config.Logger.With().
				Str("backy-cmd", cmd).Str("Host", "local machine").
				Logger()

			if cmdToRun.Host != nil {
				cmdLogger = config.Logger.With().
					Str("backy-cmd", cmd).Str("Host", *cmdToRun.Host).
					Logger()
			}

			outputArr, runOutErr := cmdToRun.RunCmd(&cmdLogger, config)
			count++
			if runOutErr != nil {
				var errMsg bytes.Buffer
				if list.NotifyConfig != nil {
					errStruct := make(map[string]interface{})

					errStruct["listName"] = list.Name
					errStruct["Command"] = currentCmd
					errStruct["Err"] = runOutErr
					errStruct["CmdsRan"] = cmdsRan
					errStruct["Output"] = outputArr

					tmpErr := msgTemps.err.Execute(&errMsg, errStruct)

					if tmpErr != nil {
						config.Logger.Err(tmpErr).Send()
					}

					notifySendErr := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s failed on command %s ", list.Name, cmd), errMsg.String())

					if notifySendErr != nil {
						config.Logger.Err(notifySendErr).Send()
					}
				}

				config.Logger.Err(runOutErr).Send()

				break
			} else {

				if count == len(list.Order) {
					cmdsRan = append(cmdsRan, cmd)
					var successMsg bytes.Buffer

					if list.NotifyConfig != nil {
						successStruct := make(map[string]interface{})
						successStruct["listName"] = list.Name
						successStruct["CmdsRan"] = cmdsRan

						tmpErr := msgTemps.success.Execute(&successMsg, successStruct)

						if tmpErr != nil {
							config.Logger.Err(tmpErr).Send()
							break
						}

						err := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s succeded", list.Name), successMsg.String())

						if err != nil {
							config.Logger.Err(err).Send()
						}
					}
				} else {
					cmdsRan = append(cmdsRan, cmd)
				}
			}
		}

		results <- "done"
	}

}

// RunBackyConfig runs a command list from the BackyConfigFile.
func (config *BackyConfigFile) RunBackyConfig(cron string) {
	mTemps := &msgTemplates{
		err:     template.Must(template.New("error.txt").ParseFS(templates, "templates/error.txt")),
		success: template.Must(template.New("success.txt").ParseFS(templates, "templates/success.txt")),
	}
	configListsLen := len(config.CmdConfigLists)
	listChan := make(chan *CmdList, configListsLen)
	results := make(chan string)

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= configListsLen; w++ {
		go cmdListWorker(mTemps, listChan, config, results)
	}

	// Here we send 5 `jobs` and then `close` that
	// channel to indicate that's all the work we have.
	// configChan <- config.Cmds
	for listName, cmdConfig := range config.CmdConfigLists {
		if cmdConfig.Name == "" {
			cmdConfig.Name = listName
		}
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

	config.closeHostConnections()
}

func (config *BackyConfigFile) ExecuteCmds(opts *BackyConfigOpts) {
	for _, cmd := range opts.executeCmds {
		cmdToRun := config.Cmds[cmd]
		cmdLogger := config.Logger.With().
			Str("backy-cmd", cmd).
			Logger()
		_, runErr := cmdToRun.RunCmd(&cmdLogger, config)
		if runErr != nil {
			config.Logger.Err(runErr).Send()
		}
	}

	config.closeHostConnections()

}

func (c *BackyConfigFile) closeHostConnections() {
	for _, host := range c.Hosts {
		if host.isProxyHost {
			continue
		}
		if host.SshClient != nil {
			if _, err := host.SshClient.NewSession(); err == nil {
				c.Logger.Info().Msgf("Closing host connection %s", host.HostName)
				host.SshClient.Close()
			}
		}
		for _, proxyHost := range host.ProxyHost {
			if proxyHost.isProxyHost {
				continue
			}
			if proxyHost.SshClient != nil {
				if _, err := host.SshClient.NewSession(); err == nil {
					c.Logger.Info().Msgf("Closing connection to proxy host %s", host.HostName)
					host.SshClient.Close()
				}
			}
		}
	}
	for _, host := range c.Hosts {
		if host.SshClient != nil {
			if _, err := host.SshClient.NewSession(); err == nil {
				c.Logger.Info().Msgf("Closing proxy host connection %s", host.HostName)
				host.SshClient.Close()
			}
		}
	}
}
