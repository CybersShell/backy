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
	"strings"
	"text/template"

	"embed"

	"github.com/rs/zerolog"
)

//go:embed templates/*.txt
var templates embed.FS

var requiredKeys = []string{"commands"}

var Sprintf = fmt.Sprintf

// RunCmd runs a Command.
// The environment of local commands will be the machine's environment plus any extra
// variables specified in the Env file or Environment.
// Dir can also be specified for local commands.
//
// Returns the output as a slice and an error, if any
func (command *Command) RunCmd(cmdCtxLogger zerolog.Logger, opts *ConfigOpts) ([]string, error) {

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

	// is host defined
	if command.Host != nil {
		command.Type = strings.TrimSpace(command.Type)

		if command.Type != "" {
			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running script %s on host %s", command.Name, *command.Host)).Send()
		} else {
			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on host %s", command.Name, *command.Host)).Send()
		}

		if command.RemoteHost.SshClient == nil {
			err := command.RemoteHost.ConnectToSSHHost(opts)
			if err != nil {
				return nil, err
			}
		}

		// create new ssh session
		commandSession, err := command.RemoteHost.SshClient.NewSession()

		// Retry connecting to host; if that fails, error. If it does not fail, try to create new session
		if err != nil {
			connErr := command.RemoteHost.ConnectToSSHHost(opts)
			if connErr != nil {
				return nil, fmt.Errorf("error creating session: %v, and error creating new connection to host: %v", err, connErr)
			}
			commandSession, err = command.RemoteHost.SshClient.NewSession()

			if err != nil {
				return nil, fmt.Errorf("error creating session: %v", err)
			}
		}

		defer commandSession.Close()

		injectEnvIntoSSH(envVars, commandSession, opts, cmdCtxLogger)
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
		// Is command type defined. That is, is it local or not
		if command.Type != "" {

			var (
				script               *bytes.Buffer
				buffer               bytes.Buffer
				scriptEnvFileBuffer  bytes.Buffer
				scriptFileBuffer     bytes.Buffer
				dirErr               error
				scriptEnvFilePresent bool
			)

			defer func() {
				// did the program panic while writing to the buffer?
				if err := recover(); err != nil {
					// cmdCtxLogger.Debug().Msg(fmt.Sprintf("script buffer: %v", script))
					cmdCtxLogger.Info().Msg(fmt.Sprintf("panic occurred writing to buffer: %v", err))
				}
			}()

			if command.Type == "script" {

				if command.ScriptEnvFile != "" {

					command.ScriptEnvFile, dirErr = resolveDir(command.ScriptEnvFile)
					if dirErr != nil {
						return nil, dirErr
					}
					file, err := os.Open(command.ScriptEnvFile)
					if err != nil {
						return nil, err
					}
					defer file.Close()
					_, err = io.Copy(&scriptEnvFileBuffer, file)
					if err != nil {
						return nil, err
					}

					// Bug: writing to buffer triggers panic
					// why?
					// use bytes.Buffer instead of pointer to memory (*bytes.Buffer)

					_, err = buffer.WriteString(scriptEnvFileBuffer.String())
					if err != nil {
						return nil, err
					}
					// write newline
					buffer.WriteByte(0x0A)

					_, err = buffer.WriteString(cmd)
					if err != nil {
						return nil, err
					}
					script = &buffer

				} else {
					script = bytes.NewBufferString(cmd + "\n")
				}

				commandSession.Stdin = script
				if err := commandSession.Shell(); err != nil {
					return nil, err
				}

				if err := commandSession.Wait(); err != nil {
					outScanner := bufio.NewScanner(&cmdOutBuf)
					for outScanner.Scan() {
						outMap := make(map[string]interface{})
						outMap["cmd"] = command.Name
						outMap["output"] = outScanner.Text()
						if str, ok := outMap["output"].(string); ok {
							outputArr = append(outputArr, str)
						}
						cmdCtxLogger.Info().Fields(outMap).Send()
					}
					return outputArr, err
				}

				outScanner := bufio.NewScanner(&cmdOutBuf)
				for outScanner.Scan() {
					outMap := make(map[string]interface{})
					outMap["cmd"] = command.Name
					outMap["output"] = outScanner.Text()
					if str, ok := outMap["output"].(string); ok {
						outputArr = append(outputArr, str)
					}
					cmdCtxLogger.Info().Fields(outMap).Send()
				}
				return outputArr, nil
			}

			if command.Type == "scriptFile" {

				if command.ScriptEnvFile != "" {

					command.ScriptEnvFile, dirErr = resolveDir(command.ScriptEnvFile)
					if dirErr != nil {
						return nil, dirErr
					}
					file, err := os.Open(command.ScriptEnvFile)
					if err != nil {
						return nil, err
					}
					defer file.Close()
					_, err = io.Copy(&scriptEnvFileBuffer, file)
					if err != nil {
						return nil, err
					}
					scriptEnvFilePresent = true
				}

				command.Cmd, dirErr = resolveDir(command.Cmd)

				if dirErr != nil {
					return nil, dirErr
				}

				// treat command.Cmd as a file
				file, err := os.Open(command.Cmd)

				if err != nil {
					return nil, err
				}

				_, err = io.Copy(&scriptFileBuffer, file)

				if err != nil {
					return nil, err
				}

				defer file.Close()

				// append scriptEnvFile to scriptFileBuffer
				if scriptEnvFilePresent {
					_, err := buffer.WriteString(scriptEnvFileBuffer.String())
					if err != nil {
						return nil, err
					}
					// write newline
					buffer.WriteByte(0x0A)

					_, err = buffer.WriteString(scriptFileBuffer.String())
					if err != nil {
						return nil, err
					}
				} else {
					_, err = io.Copy(&buffer, file)
					if err != nil {
						return nil, err
					}
				}

				script := &buffer

				commandSession.Stdin = script
				if err := commandSession.Shell(); err != nil {
					return nil, err
				}
				if err := commandSession.Wait(); err != nil {
					outScanner := bufio.NewScanner(&cmdOutBuf)
					for outScanner.Scan() {
						outMap := make(map[string]interface{})
						outMap["cmd"] = cmd
						outMap["output"] = outScanner.Text()
						if str, ok := outMap["output"].(string); ok {
							outputArr = append(outputArr, str)
						}
						cmdCtxLogger.Info().Fields(outMap).Send()
					}
					return outputArr, err
				}

				outScanner := bufio.NewScanner(&cmdOutBuf)
				for outScanner.Scan() {
					outMap := make(map[string]interface{})
					outMap["cmd"] = cmd
					outMap["output"] = outScanner.Text()
					if str, ok := outMap["output"].(string); ok {
						outputArr = append(outputArr, str)
					}
					cmdCtxLogger.Info().Fields(outMap).Send()
				}
				return outputArr, nil
			}
			return nil, fmt.Errorf("command type not recognized")
		}

		err = commandSession.Run(cmd)
		outScanner := bufio.NewScanner(&cmdOutBuf)
		for outScanner.Scan() {
			outMap := make(map[string]interface{})
			outMap["cmd"] = command.Name
			outMap["output"] = outScanner.Text()
			if str, ok := outMap["output"].(string); ok {
				outputArr = append(outputArr, str)
			}
			cmdCtxLogger.Info().Fields(outMap).Send()
		}

		if err != nil {
			cmdCtxLogger.Error().Err(fmt.Errorf("error when running cmd: %s: %w", command.Name, err)).Send()
			return outputArr, err
		}
	} else {

		var err error
		if command.Shell != "" {
			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine in %s", command.Name, command.Shell)).Send()

			ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)

			localCMD := exec.Command(command.Shell, "-c", ArgsStr)

			if command.Dir != nil {
				localCMD.Dir = *command.Dir
			}
			injectEnvIntoLocalCMD(envVars, localCMD, cmdCtxLogger)

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
				outMap["cmd"] = command.Name
				outMap["output"] = outScanner.Text()
				if str, ok := outMap["output"].(string); ok {
					outputArr = append(outputArr, str)
				}
				cmdCtxLogger.Info().Fields(outMap).Send()
			}

			if err != nil {
				cmdCtxLogger.Error().Err(fmt.Errorf("error when running cmd %s: %w", command.Name, err)).Send()
				return outputArr, err
			}
			return outputArr, nil
		}

		cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine", command.Name)).Send()

		localCMD := exec.Command(command.Cmd, command.Args...)

		if command.Dir != nil {
			localCMD.Dir = *command.Dir
		}

		injectEnvIntoLocalCMD(envVars, localCMD, cmdCtxLogger)

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
			cmdCtxLogger.Info().Fields(outMap).Send()
		}
		if err != nil {
			cmdCtxLogger.Error().Err(fmt.Errorf("error when running cmd %s: %w", command.Name, err)).Send()
			return outputArr, err
		}
	}
	return outputArr, nil
}

// cmdListWorker
func cmdListWorker(msgTemps *msgTemplates, jobs <-chan *CmdList, results chan<- string, opts *ConfigOpts) {
	// iterate over list to run
	res := CmdListResults{}
	for list := range jobs {
		fieldsMap := make(map[string]interface{})
		fieldsMap["list"] = list.Name
		var cmdLogger zerolog.Logger

		var count int                // count of how many commands have been executed
		var cmdsRan []string         // store the commands that have been executed
		var outStructArr []outStruct // stores output messages

		for _, cmd := range list.Order {

			currentCmd := opts.Cmds[cmd].Name

			fieldsMap["cmd"] = opts.Cmds[cmd].Name
			cmdToRun := opts.Cmds[cmd]

			cmdLogger = cmdToRun.generateLogger(opts)
			cmdLogger.Info().Fields(fieldsMap).Send()

			outputArr, runOutErr := cmdToRun.RunCmd(cmdLogger, opts)

			if list.NotifyConfig != nil {

				// check if the command output should be included
				if cmdToRun.GetOutput || list.GetOutput {
					outputStruct := outStruct{
						CmdName:     cmdToRun.Name,
						CmdExecuted: currentCmd,
						Output:      outputArr,
					}

					outStructArr = append(outStructArr, outputStruct)

				}
			}
			count++
			if runOutErr != nil {
				res.ErrCmd = cmd
				if list.NotifyConfig != nil {
					var errMsg bytes.Buffer
					errStruct := make(map[string]interface{})

					errStruct["listName"] = list.Name
					errStruct["Command"] = currentCmd
					errStruct["Cmd"] = cmd
					errStruct["Args"] = opts.Cmds[cmd].Args
					errStruct["Err"] = runOutErr
					errStruct["CmdsRan"] = cmdsRan
					errStruct["Output"] = outputArr

					errStruct["CmdOutput"] = outStructArr

					tmpErr := msgTemps.err.Execute(&errMsg, errStruct)

					if tmpErr != nil {
						cmdLogger.Err(tmpErr).Send()
					}

					notifySendErr := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s failed", list.Name), errMsg.String())

					if notifySendErr != nil {
						cmdLogger.Err(notifySendErr).Send()
					}
				}

				cmdLogger.Err(runOutErr).Send()

				break
			} else {

				cmdsRan = append(cmdsRan, cmd)

				if count == len(list.Order) {
					var successMsg bytes.Buffer

					// if notification config is not nil, and NotifyOnSuccess is true or GetOuput is true,
					// then send notification
					if list.NotifyConfig != nil && (list.NotifyOnSuccess || list.GetOutput) {
						successStruct := make(map[string]interface{})

						successStruct["listName"] = list.Name
						successStruct["CmdsRan"] = cmdsRan

						successStruct["CmdOutput"] = outStructArr

						tmpErr := msgTemps.success.Execute(&successMsg, successStruct)

						if tmpErr != nil {
							cmdLogger.Err(tmpErr).Send()
							break
						}

						err := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s succeeded", list.Name), successMsg.String())

						if err != nil {
							cmdLogger.Err(err).Send()
						}
					}
				}
			}
		}

		results <- res.ErrCmd
	}

}

// RunListConfig runs a command list from the ConfigFile.
func (opts *ConfigOpts) RunListConfig(cron string) {
	mTemps := &msgTemplates{
		err:     template.Must(template.New("error.txt").ParseFS(templates, "templates/error.txt")),
		success: template.Must(template.New("success.txt").ParseFS(templates, "templates/success.txt")),
	}
	configListsLen := len(opts.CmdConfigLists)
	listChan := make(chan *CmdList, configListsLen)
	results := make(chan string)

	// This starts up list workers, initially blocked
	// because there are no jobs yet.
	for w := 1; w <= configListsLen; w++ {
		go cmdListWorker(mTemps, listChan, results, opts)
	}

	for listName, cmdConfig := range opts.CmdConfigLists {
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
		l := <-results

		opts.Logger.Debug().Msg(l)

		if l != "" {
			// execute error hooks
			opts.Logger.Debug().Msg("hooks are working")
			opts.Cmds[l].ExecuteHooks("error", opts)
		} else {
			// execute success hooks
			opts.Cmds[l].ExecuteHooks("success", opts)

		}

		// execute final hooks
		opts.Cmds[l].ExecuteHooks("final", opts)

	}

	opts.closeHostConnections()
}

func (config *ConfigOpts) ExecuteCmds(opts *ConfigOpts) {
	for _, cmd := range opts.executeCmds {
		cmdToRun := opts.Cmds[cmd]
		cmdLogger := cmdToRun.generateLogger(opts)
		_, runErr := cmdToRun.RunCmd(cmdLogger, opts)
		if runErr != nil {
			opts.Logger.Err(runErr).Send()

			cmdToRun.ExecuteHooks("error", opts)
		} else {

			cmdToRun.ExecuteHooks("success", opts)
		}

		cmdToRun.ExecuteHooks("final", opts)
	}

	opts.closeHostConnections()

}

func (c *ConfigOpts) closeHostConnections() {
	for _, host := range c.Hosts {
		c.Logger.Info().Str("server", host.HostName)
		if host.isProxyHost {
			continue
		}
		if host.SshClient != nil {
			if _, err := host.SshClient.NewSession(); err == nil {
				c.Logger.Info().Msgf("Closing host connection %s", host.HostName)
				host.SshClient.Close()
				host.SshClient = nil
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
					host.SshClient = nil
				}
			}
		}
	}
	for _, host := range c.Hosts {
		if host.SshClient != nil {
			if _, err := host.SshClient.NewSession(); err == nil {
				c.Logger.Info().Msgf("Closing proxy host connection %s", host.HostName)
				host.SshClient.Close()
				host.SshClient = nil
			}
		}
	}
}

func (cmd *Command) ExecuteHooks(hookType string, opts *ConfigOpts) {
	if cmd.Hooks == nil {
		return
	}
	switch hookType {
	case "error":
		for _, v := range cmd.Hooks.Error {
			errCmd := opts.Cmds[v]
			cmdLogger := opts.Logger.With().
				Str("backy-cmd", v).
				Logger()
			errCmd.RunCmd(cmdLogger, opts)
		}

	case "success":
		for _, v := range cmd.Hooks.Success {
			successCmd := opts.Cmds[v]
			cmdLogger := opts.Logger.With().
				Str("backy-cmd", v).
				Logger()
			successCmd.RunCmd(cmdLogger, opts)
		}
	case "final":
		for _, v := range cmd.Hooks.Final {
			finalCmd := opts.Cmds[v]
			cmdLogger := opts.Logger.With().
				Str("backy-cmd", v).
				Logger()
			finalCmd.RunCmd(cmdLogger, opts)
		}
	}
}

func (cmd *Command) generateLogger(opts *ConfigOpts) zerolog.Logger {
	cmdLogger := opts.Logger.With().
		Str("backy-cmd", cmd.Name).Str("Host", "local machine").
		Logger()

	if cmd.Host != nil {
		cmdLogger = opts.Logger.With().
			Str("backy-cmd", cmd.Name).Str("Host", *cmd.Host).
			Logger()

	}
	return cmdLogger
}
