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

	command = getCommandType(command)

	if command.Type == "user" {
		if command.UserOperation == "password" {
			cmdCtxLogger.Info().Str("password", command.UserPassword).Msg("user password to be updated")
		}
	}

	var errSSH error
	// is host defined
	if command.Host != nil {
		print("host is defined")
		outputArr, errSSH = command.RunCmdSSH(cmdCtxLogger, opts)
		if errSSH != nil {
			return outputArr, errSSH
		}
	} else {

		// Handle package operations
		if command.Type == "package" && command.PackageOperation == "checkVersion" {
			cmdCtxLogger.Info().Str("package", command.PackageName).Msg("Checking package versions")

			// Execute the package version command
			cmd := exec.Command(command.Cmd, command.Args...)
			cmdOutWriters = io.MultiWriter(&cmdOutBuf)
			cmd.Stdout = cmdOutWriters
			cmd.Stderr = cmdOutWriters

			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("error running command %s policy: %w", ArgsStr, err)
			}

			return parsePackageVersion(cmdOutBuf.String(), cmdCtxLogger, command, cmdOutBuf)
		}

		var localCMD *exec.Cmd
		var err error
		if command.Shell != "" {
			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine in %s", command.Name, command.Shell)).Send()

			ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)

			localCMD = exec.Command(command.Shell, "-c", ArgsStr)

		} else {

			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine", command.Name)).Send()

			localCMD = exec.Command(command.Cmd, command.Args...)
		}

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
			// if command.GetOutput {
			cmdCtxLogger.Info().Fields(outMap).Send()
			// }
		}
		if err != nil {
			cmdCtxLogger.Error().Err(fmt.Errorf("error when running cmd %s: %w", command.Name, err)).Send()
			return outputArr, err
		}
	}
	return outputArr, nil
}

func cmdListWorker(msgTemps *msgTemplates, jobs <-chan *CmdList, results chan<- CmdResult, opts *ConfigOpts) {
	for list := range jobs {
		fieldsMap := map[string]interface{}{"list": list.Name}
		var cmdLogger zerolog.Logger
		var cmdsRan []string
		var outStructArr []outStruct
		var hasError bool // Tracks if any command in the list failed

		for _, cmd := range list.Order {
			cmdToRun := opts.Cmds[cmd]
			currentCmd := cmdToRun.Name
			fieldsMap["cmd"] = currentCmd
			cmdLogger = cmdToRun.GenerateLogger(opts)
			cmdLogger.Info().Fields(fieldsMap).Send()

			outputArr, runErr := cmdToRun.RunCmd(cmdLogger, opts)
			cmdsRan = append(cmdsRan, cmd)

			if runErr != nil {

				// Log the error and send a failed result
				cmdLogger.Err(runErr).Send()
				results <- CmdResult{CmdName: cmd, ListName: list.Name, Error: runErr}

				// Execute error hooks for the failed command
				cmdToRun.ExecuteHooks("error", opts)

				// Notify failure
				if list.NotifyConfig != nil {
					notifyError(cmdLogger, msgTemps, list, cmdsRan, outStructArr, runErr, cmdToRun)
				}
				hasError = true
				break
			}

			// Collect output if required
			if list.GetOutput || cmdToRun.GetOutput {
				outStructArr = append(outStructArr, outStruct{
					CmdName:     currentCmd,
					CmdExecuted: currentCmd,
					Output:      outputArr,
				})
			}
		}

		// Notify success if no errors occurred
		if !hasError && list.NotifyConfig != nil && (list.NotifyOnSuccess || list.GetOutput) {
			notifySuccess(cmdLogger, msgTemps, list, cmdsRan, outStructArr)
		}

		// Execute success and final hooks for all commands
		for _, cmd := range list.Order {
			cmdToRun := opts.Cmds[cmd]

			// Execute success hooks if the command succeeded
			if !hasError || cmdsRanContains(cmd, cmdsRan) {
				cmdToRun.ExecuteHooks("success", opts)
			}

			// Execute final hooks for every command
			cmdToRun.ExecuteHooks("final", opts)
		}

		// Send the final result for the list
		if hasError {
			results <- CmdResult{CmdName: cmdsRan[len(cmdsRan)-1], ListName: list.Name, Error: fmt.Errorf("list execution failed")}
		} else {
			results <- CmdResult{CmdName: cmdsRan[len(cmdsRan)-1], ListName: list.Name, Error: nil}
		}
	}
}

// Helper to check if a command is in the list of executed commands
func cmdsRanContains(cmd string, cmdsRan []string) bool {
	for _, c := range cmdsRan {
		if c == cmd {
			return true
		}
	}
	return false
}

// Helper to notify errors
func notifyError(logger zerolog.Logger, templates *msgTemplates, list *CmdList, cmdsRan []string, outStructArr []outStruct, err error, cmd *Command) {
	errStruct := map[string]interface{}{
		"listName":  list.Name,
		"CmdsRan":   cmdsRan,
		"CmdOutput": outStructArr,
		"Err":       err,
		"Command":   cmd.Name,
		"Args":      cmd.Args,
	}
	var errMsg bytes.Buffer
	if e := templates.err.Execute(&errMsg, errStruct); e != nil {
		logger.Err(e).Send()
		return
	}
	if e := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s failed", list.Name), errMsg.String()); e != nil {
		logger.Err(e).Send()
	}
}

// Helper to notify success
func notifySuccess(logger zerolog.Logger, templates *msgTemplates, list *CmdList, cmdsRan []string, outStructArr []outStruct) {
	successStruct := map[string]interface{}{
		"listName":  list.Name,
		"CmdsRan":   cmdsRan,
		"CmdOutput": outStructArr,
	}
	var successMsg bytes.Buffer
	if e := templates.success.Execute(&successMsg, successStruct); e != nil {
		logger.Err(e).Send()
		return
	}
	if e := list.NotifyConfig.Send(context.Background(), fmt.Sprintf("List %s succeeded", list.Name), successMsg.String()); e != nil {
		logger.Err(e).Send()
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
	results := make(chan CmdResult, configListsLen)

	// Start workers
	for w := 1; w <= configListsLen; w++ {
		go cmdListWorker(mTemps, listChan, results, opts)
	}

	// Enqueue jobs
	for listName, cmdConfig := range opts.CmdConfigLists {
		if cmdConfig.Name == "" {
			cmdConfig.Name = listName
		}
		if cron == "" || cron == cmdConfig.Cron {
			listChan <- cmdConfig
		}
	}
	close(listChan)

	// Process results
	for a := 1; a <= configListsLen; a++ {
		result := <-results
		opts.Logger.Debug().Msgf("Processing result for list %s, command %s", result.ListName, result.CmdName)

		// Process final hooks for the list (already handled in worker)
	}
	opts.closeHostConnections()
}

func (opts *ConfigOpts) ExecuteCmds() {
	for _, cmd := range opts.executeCmds {
		cmdToRun := opts.Cmds[cmd]
		cmdLogger := cmdToRun.GenerateLogger(opts)
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

func (cmd *Command) GenerateLogger(opts *ConfigOpts) zerolog.Logger {
	cmdLogger := opts.Logger.With().
		Str("Backy-cmd", cmd.Name).Str("Host", "local machine").
		Logger()

	if cmd.Host != nil {
		cmdLogger = opts.Logger.With().
			Str("Backy-cmd", cmd.Name).Str("Host", *cmd.Host).
			Logger()

	}
	return cmdLogger
}

func (opts *ConfigOpts) ExecCmdsSSH(cmdList []string, hostsList []string) {
	// Iterate over hosts and exec commands
	for _, h := range hostsList {
		host := opts.Hosts[h]
		for _, c := range cmdList {
			cmd := opts.Cmds[c]
			cmd.RemoteHost = host
			cmd.Host = &host.Host
			opts.Logger.Info().Str("host", h).Str("cmd", c).Send()
			_, err := cmd.RunCmdSSH(cmd.GenerateLogger(opts), opts)
			if err != nil {
				opts.Logger.Err(err).Str("host", h).Str("cmd", c).Send()
			}
		}
	}
}

// func executeUserCommands() []string {

// }

// // parseRemoteSources parses source and validates fields using sourceType
// func (c *Command) parseRemoteSources(source, sourceType string) {
// 	switch sourceType {

// 	}
// }
