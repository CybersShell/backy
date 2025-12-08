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
	"sync"
	"text/template"

	"embed"

	"github.com/rs/zerolog"
)

//go:embed templates/*.txt
var templates embed.FS

var requiredKeys = []string{"commands"}

var Sprintf = fmt.Sprintf

type CommandExecutor interface {
	Run(cmd *Command, opts *ConfigOpts, logger zerolog.Logger) ([]string, error)
}

type OutputHandler interface {
	CollectOutput(buf *bytes.Buffer, commandName string, logger zerolog.Logger, wantOutput bool) []string
}

type EnvInjector interface {
	Inject(cmd *Command, opts *ConfigOpts)
}

type PackageCommandExecutor struct{}

func (e *PackageCommandExecutor) Run(cmd *Command, opts *ConfigOpts, logger zerolog.Logger) ([]string, error) {
	var (
		ArgsStr       string
		cmdOutBuf     bytes.Buffer
		outputArr     []string
		cmdOutWriters io.Writer
	)

	for _, v := range cmd.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	// Example: Check version operation
	if cmd.PackageOperation == PackageOperationCheckVersion {
		logger.Info().Msg("Checking package versions")

		logger.Info().Msg("")
		for _, p := range cmd.Packages {
			logger.Info().Str("package", p.Name).Msg("Checking installed and remote package versions")
		}
		opts.Logger.Info().Msg("")

		// Execute the package version command
		execCmd := exec.Command(cmd.Cmd, cmd.Args...)
		cmdOutWriters = io.MultiWriter(&cmdOutBuf)

		if IsCmdStdOutEnabled() {
			cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
		}
		execCmd.Stdout = cmdOutWriters
		execCmd.Stderr = cmdOutWriters

		if err := execCmd.Run(); err != nil {
			return nil, fmt.Errorf("error running command %s: %w", ArgsStr, err)
		}

		return parsePackageVersion(cmdOutBuf.String(), logger, cmd, cmdOutBuf)
	}

	// Other package operations (install, upgrade, etc.) can be handled here

	// Default: run as a shell command
	execCmd := exec.Command(cmd.Cmd, cmd.Args...)
	execCmd.Stdout = &cmdOutBuf
	execCmd.Stderr = &cmdOutBuf
	err := execCmd.Run()
	outputArr = logCommandOutput(cmd, cmdOutBuf, logger, outputArr)
	if err != nil {
		logger.Error().Err(fmt.Errorf("error running package command %s: %w", cmd.Name, err)).Send()
		return outputArr, err
	}
	return outputArr, nil
}

type LocalCommandExecutor struct{}

func (e *LocalCommandExecutor) Run(cmd *Command, opts *ConfigOpts, logger zerolog.Logger) ([]string, error) {
	var (
		ArgsStr   string
		cmdOutBuf bytes.Buffer
		outputArr []string
	)

	for _, v := range cmd.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	// Build the command
	var localCMD *exec.Cmd
	if cmd.Shell != "" {
		logger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine in %s", cmd.Name, cmd.Shell)).Send()
		ArgsStr = fmt.Sprintf("%s %s", cmd.Cmd, ArgsStr)
		localCMD = exec.Command(cmd.Shell, "-c", ArgsStr)
	} else {
		localCMD = exec.Command(cmd.Cmd, cmd.Args...)
	}

	// Set working directory
	if cmd.Dir != nil {
		localCMD.Dir = *cmd.Dir
	}

	// Inject environment variables (extract this to an EnvInjector if desired)
	// injectEnvIntoLocalCMD(...)

	// Set output writers
	cmdOutWriters := io.MultiWriter(&cmdOutBuf)
	if IsCmdStdOutEnabled() {
		cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
	}
	localCMD.Stdout = cmdOutWriters
	localCMD.Stderr = cmdOutWriters

	// Run the command
	err := localCMD.Run()
	outputArr = logCommandOutput(cmd, cmdOutBuf, logger, outputArr)
	if err != nil {
		logger.Error().Err(fmt.Errorf("error when running cmd %s: %w", cmd.Name, err)).Send()
		return outputArr, err
	}

	return outputArr, nil
}

// ensureRemoteHost ensures localCmd.RemoteHost is set for the given host.
// It prefers opts.Hosts lookup and falls back to a minimal Host entry so remote execution can proceed.
func (opts *ConfigOpts) ensureRemoteHost(localCmd *Command, host string) {
	if localCmd.RemoteHost != nil {
		return
	}
	if opts != nil && opts.Hosts != nil {
		if rh, found := opts.Hosts[host]; found {
			localCmd.RemoteHost = rh
			return
		}
	}
	// fallback: create a minimal Host so RunCmdOnHost sees a non-nil RemoteHost.
	// This uses host as the address/alias; further fields (user/key) will use defaults.
	localCmd.RemoteHost = &Host{Host: host}
}

// ExecCommandOnHostsParallel runs a single configured command concurrently on the command.Hosts list.
// It reuses the standard RunCmd / RunCmdOnHost flow so the behavior is identical to normal execution.
func (opts *ConfigOpts) ExecCommandOnHostsParallel(cmdName string) ([]CmdResult, error) {
	cmdObj, ok := opts.Cmds[cmdName]
	if !ok {
		return nil, fmt.Errorf("command %s not found", cmdName)
	}
	if len(cmdObj.Hosts) == 0 {
		return nil, fmt.Errorf("no hosts configured for command %s", cmdName)
	}

	var wg sync.WaitGroup
	resultsCh := make(chan CmdResult, len(cmdObj.Hosts))

	for _, host := range cmdObj.Hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			// shallow copy to avoid races
			local := *cmdObj
			local.Host = h
			opts.Logger.Debug().Str("host", h).Msg("executing command in parallel on host")

			var err error
			if IsHostLocal(h) {
				_, err := local.RunCmd(local.GenerateLogger(opts), opts)
				resultsCh <- CmdResult{CmdName: cmdName, ListName: "", Error: err}
				return
				// _, err = local.RunCmd(local.GenerateLogger(opts), opts)
			}

			// ensure RemoteHost is populated before calling RunCmdOnHost
			opts.ensureRemoteHost(&local, h)

			_, err = local.RunCmdOnHost(local.GenerateLogger(opts), opts)

			resultsCh <- CmdResult{CmdName: cmdName, ListName: "", Error: err}
		}(host)
	}

	wg.Wait()
	close(resultsCh)

	var results []CmdResult
	for r := range resultsCh {
		results = append(results, r)
	}
	return results, nil
}

// RunCmd runs a Command.
// The environment of local commands will be the machine's environment plus any extra
// variables specified in the Env file or Environment.
// Dir can also be specified for local commands.
//
// Returns the output as a slice and an error, if any
func (command *Command) RunCmd(cmdCtxLogger zerolog.Logger, opts *ConfigOpts) ([]string, error) {

	var (
		ArgsStr       string
		cmdOutBuf     bytes.Buffer
		cmdOutWriters io.Writer
		errSSH        error

		envVars = environmentVars{
			file: command.Env,
			env:  command.Environment,
		}

		outputArr []string // holds the output strings returned by processes
	)

	if command.Host != "" && command.Hosts != nil {
		cmdCtxLogger.Warn().Msg("both 'host' and 'hosts' are set; 'hosts' will be ignored")
		return nil, fmt.Errorf("both 'host' and 'hosts' are set; please set one or the other")
	} else if command.Hosts != nil {
		opts.ExecCommandOnHostsParallel(command.Name)
		return nil, nil
	}

	// Getting the command type must be done before concatenating the arguments
	command = getCommandTypeAndSetCommandInfo(command)

	for _, v := range command.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	if command.Type == UserCommandType {
		if command.UserOperation == "password" {
			cmdCtxLogger.Info().Str("password", command.UserPassword).Msg("user password to be updated")
		}
	}

	if !IsHostLocal(command.Host) {

		outputArr, errSSH = command.RunCmdOnHost(cmdCtxLogger, opts)
		if errSSH != nil {
			return outputArr, errSSH
		}
	} else {

		switch command.Type {
		case PackageCommandType:
			var executor PackageCommandExecutor
			return executor.Run(command, opts, cmdCtxLogger)
		}

		var localCMD *exec.Cmd

		if command.Type == RemoteScriptCommandType {
			script, err := command.Fetcher.Fetch(command.Cmd)
			if err != nil {
				return nil, err
			}

			if command.Shell == "" {
				command.Shell = "sh"
			}
			localCMD = exec.Command(command.Shell, command.Args...)
			injectEnvIntoLocalCMD(envVars, localCMD, cmdCtxLogger, opts)

			cmdOutWriters = io.MultiWriter(&cmdOutBuf)

			if IsCmdStdOutEnabled() {
				cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
			}
			if command.Output.File != "" {
				file, err := os.Create(command.Output.File)
				if err != nil {
					return nil, fmt.Errorf("error creating output file: %w", err)
				}
				defer file.Close()
				cmdOutWriters = io.MultiWriter(file, &cmdOutBuf)

				if IsCmdStdOutEnabled() {
					cmdOutWriters = io.MultiWriter(os.Stdout, file, &cmdOutBuf)
				}

			}

			localCMD.Stdin = bytes.NewReader(script)
			localCMD.Stdout = cmdOutWriters
			localCMD.Stderr = cmdOutWriters

			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running remoteScript %s on local machine in %s", command.Cmd, command.Shell)).Send()
			err = localCMD.Run()
			if err != nil {
				return nil, fmt.Errorf("error running remote script: %w", err)
			}

			outScanner := bufio.NewScanner(&cmdOutBuf)

			for outScanner.Scan() {
				outMap := make(map[string]interface{})
				outMap["cmd"] = command.Cmd
				outMap["output"] = outScanner.Text()

				if str, ok := outMap["output"].(string); ok {
					outputArr = append(outputArr, str)
				}
				if command.Output.ToLog {
					cmdCtxLogger.Info().Fields(outMap).Send()
				}
			}
			return outputArr, nil
		}

		var err error

		if command.Shell != "" {
			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine in %s", command.Name, command.Shell)).Send()

			ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)

			localCMD = exec.Command(command.Shell, "-c", ArgsStr)

		} else {

			cmdCtxLogger.Info().Str("Command", fmt.Sprintf("Running command %s on local machine", command.Name)).Send()

			// execute package commands in a shell
			if command.Type == PackageCommandType {
				for _, p := range command.Packages {
					cmdCtxLogger.Info().Str("packages", p.Name).Msg("Executing package command")
				}
				ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)
				localCMD = exec.Command("/bin/sh", "-c", ArgsStr)
			} else {
				if command.Env != "" || command.Environment != nil {
					localCMD = exec.Command("/bin/sh", "-c", ArgsStr)
				} else {
					localCMD = exec.Command(command.Cmd, command.Args...)
				}
			}
		}

		if command.Type == UserCommandType {
			if command.UserOperation == "password" {
				localCMD.Stdin = command.stdin
				cmdCtxLogger.Info().Str("password", command.UserPassword).Msg("user password to be updated")
			}
		}
		if command.Dir != nil {
			localCMD.Dir = *command.Dir
		}

		injectEnvIntoLocalCMD(envVars, localCMD, cmdCtxLogger, opts)

		cmdOutWriters = io.MultiWriter(&cmdOutBuf)

		if IsCmdStdOutEnabled() {
			cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
		}

		localCMD.Stdout = cmdOutWriters
		localCMD.Stderr = cmdOutWriters

		err = localCMD.Run()

		outputArr = logCommandOutput(command, cmdOutBuf, cmdCtxLogger, outputArr)
		if err != nil {
			cmdCtxLogger.Error().Err(fmt.Errorf("error when running cmd %s: %w", command.Name, err)).Send()
			return outputArr, err
		}

		if command.Type == UserCommandType {

			if command.UserOperation == "add" {
				if command.UserSshPubKeys != nil {
					var (
						authorizedKeysFile *os.File
						err                error
						userHome           []byte
					)

					cmdCtxLogger.Info().Msg("adding SSH Keys")

					localCMD := exec.Command(fmt.Sprintf("grep \"%s\" /etc/passwd | cut -d: -f6", command.Username))
					userHome, err = localCMD.CombinedOutput()
					if err != nil {
						return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error finding user home from /etc/passwd: %v", err)
					}

					command.UserHome = strings.TrimSpace(string(userHome))
					userSshDir := fmt.Sprintf("%s/.ssh", command.UserHome)

					if _, err := os.Stat(userSshDir); os.IsNotExist(err) {
						err := os.MkdirAll(userSshDir, 0700)
						if err != nil {
							return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error creating directory %s %v", userSshDir, err)
						}
					}

					if _, err := os.Stat(fmt.Sprintf("%s/authorized_keys", userSshDir)); os.IsNotExist(err) {
						_, err := os.Create(fmt.Sprintf("%s/authorized_keys", userSshDir))
						if err != nil {
							return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error creating file %s/authorized_keys: %v", userSshDir, err)
						}
					}

					authorizedKeysFile, err = os.OpenFile(fmt.Sprintf("%s/authorized_keys", userSshDir), 0700, os.ModeAppend)
					if err != nil {
						return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error opening file %s/authorized_keys: %v", userSshDir, err)
					}
					defer authorizedKeysFile.Close()
					for _, k := range command.UserSshPubKeys {
						buf := bytes.NewBufferString(k)
						cmdCtxLogger.Info().Str("key", k).Msg("adding SSH key")
						if _, err := authorizedKeysFile.ReadFrom(buf); err != nil {
							return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), fmt.Errorf("error adding to authorized keys: %v", err)
						}
					}
					localCMD = exec.Command(fmt.Sprintf("chown -R %s:%s %s", command.Username, command.Username, userHome))
					_, err = localCMD.CombinedOutput()
					if err != nil {
						return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.Output.ToLog), err
					}

				}
			}
		}
	}
	return outputArr, nil
}

func cmdListWorker(msgTemps *msgTemplates, jobs <-chan *CmdList, results chan<- string, opts *ConfigOpts) {
	for list := range jobs {
		fieldsMap := map[string]interface{}{"list": list.Name}
		var cmdLogger zerolog.Logger
		var commandExecuted *Command
		var cmdsRan []string
		var outStructArr []outStruct
		var hasError bool // Tracks if any command in the list failed

		for _, cmd := range list.Order {
			cmdToRun := opts.Cmds[cmd]
			commandExecuted = cmdToRun
			currentCmd := cmdToRun.Name
			fieldsMap["cmd"] = currentCmd
			cmdLogger = cmdToRun.GenerateLogger(opts)
			cmdLogger.Info().Fields(fieldsMap).Send()

			outputArr, runErr := cmdToRun.RunCmd(cmdLogger, opts)
			cmdsRan = append(cmdsRan, cmd)

			if runErr != nil {

				cmdLogger.Err(runErr).Send()

				cmdToRun.ExecuteHooks("error", opts)

				// Notify failure
				if list.NotifyConfig != nil {
					notifyError(cmdLogger, msgTemps, list, cmdsRan, outStructArr, runErr, cmdToRun)
				}

				// Execute error hooks for the failed command
				hasError = true
				break
			}

			if list.GetCommandOutputInNotificationsOnSuccess || cmdToRun.Output.InList {
				outStructArr = append(outStructArr, outStruct{
					CmdName:     currentCmd,
					CmdExecuted: currentCmd,
					Output:      outputArr,
				})
			}
		}

		if !hasError && list.NotifyConfig != nil && list.Notify.OnFailure {
			notifySuccess(cmdLogger, msgTemps, list, cmdsRan, outStructArr)
		}

		if !hasError {
			commandExecuted.ExecuteHooks("success", opts)
		}

		commandExecuted.ExecuteHooks("final", opts)

		results <- "done"
	}
}
func cmdListWorkerWithHosts(msgTemps *msgTemplates, jobs <-chan *CmdList, hosts <-chan *Host, results chan<- string, opts *ConfigOpts) {
	for list := range jobs {
		fieldsMap := map[string]interface{}{"list": list.Name}
		var cmdLogger zerolog.Logger
		var commandExecuted *Command
		var cmdsRan []string
		var outStructArr []outStruct
		var hasError bool // Tracks if any command in the list failed

		for host := range hosts {

			for _, cmd := range list.Order {
				cmdToRun := opts.Cmds[cmd]
				if cmdToRun.Host != host.Host {
					cmdToRun.Host = host.Host
					cmdToRun.RemoteHost = host
				}
				commandExecuted = cmdToRun
				currentCmd := cmdToRun.Name
				fieldsMap["cmd"] = currentCmd
				cmdLogger = cmdToRun.GenerateLogger(opts)
				cmdLogger.Info().Fields(fieldsMap).Send()

				outputArr, runErr := cmdToRun.RunCmd(cmdLogger, opts)
				cmdsRan = append(cmdsRan, cmd)

				if runErr != nil {

					cmdLogger.Err(runErr).Send()

					cmdToRun.ExecuteHooks("error", opts)

					// Notify failure
					if list.NotifyConfig != nil {
						notifyError(cmdLogger, msgTemps, list, cmdsRan, outStructArr, runErr, cmdToRun)
					}

					// Execute error hooks for the failed command
					hasError = true
					break
				}

				if list.GetCommandOutputInNotificationsOnSuccess || cmdToRun.Output.InList {
					outStructArr = append(outStructArr, outStruct{
						CmdName:     currentCmd,
						CmdExecuted: currentCmd,
						Output:      outputArr,
					})
				}
			}

			if !hasError && list.NotifyConfig != nil && list.Notify.OnFailure {
				notifySuccess(cmdLogger, msgTemps, list, cmdsRan, outStructArr)
			}

			if !hasError {
				commandExecuted.ExecuteHooks("success", opts)
			}

			commandExecuted.ExecuteHooks("final", opts)

		}
		results <- "done"
	}
}

// func cmdListWorkerExecuteCommandsInParallel(msgTemps *msgTemplates, jobs <-chan *CmdList, hosts <-chan *Host, results chan<- string, opts *ConfigOpts) {
// 	opts.Logger.Info().Msg("Running commands in parallel")
// 	for list := range jobs {
// 		fieldsMap := map[string]interface{}{"list": list.Name}
// 		var cmdLogger zerolog.Logger
// 		var commandExecuted *Command
// 		var cmdsRan []string
// 		var outStructArr []outStruct
// 		var hasError bool // Tracks if any command in the list failed

// 		for _, cmd := range list.Order {
// 			for host := range hosts {
// 				cmdToRun := opts.Cmds[cmd]
// 				if cmdToRun.Host != host.Host {
// 					cmdToRun.Host = host.Host
// 					cmdToRun.RemoteHost = host
// 				}
// 				commandExecuted = cmdToRun
// 				currentCmd := cmdToRun.Name
// 				fieldsMap["cmd"] = currentCmd
// 				cmdLogger = cmdToRun.GenerateLogger(opts)
// 				cmdLogger.Info().Fields(fieldsMap).Send()

// 				outputArr, runErr := cmdToRun.RunCmd(cmdLogger, opts)
// 				cmdsRan = append(cmdsRan, cmd)

// 				if runErr != nil {

// 					cmdLogger.Err(runErr).Send()

// 					cmdToRun.ExecuteHooks("error", opts)

// 					// Notify failure
// 					if list.NotifyConfig != nil {
// 						notifyError(cmdLogger, msgTemps, list, cmdsRan, outStructArr, runErr, cmdToRun)
// 					}

// 					// Execute error hooks for the failed command
// 					hasError = true
// 					break
// 				}

// 				if list.GetCommandOutputInNotificationsOnSuccess || cmdToRun.Output.InList {
// 					outStructArr = append(outStructArr, outStruct{
// 						CmdName:     currentCmd,
// 						CmdExecuted: currentCmd,
// 						Output:      outputArr,
// 					})
// 				}
// 			}

// 			if !hasError && list.NotifyConfig != nil && list.Notify.OnFailure {
// 				notifySuccess(cmdLogger, msgTemps, list, cmdsRan, outStructArr)
// 			}

// 			if !hasError {
// 				commandExecuted.ExecuteHooks("success", opts)
// 			}

// 			commandExecuted.ExecuteHooks("final", opts)

// 		}
// 		results <- "done"
// 	}
// }

func cmdListWorkerExecuteCommandsInParallel(msgTemps *msgTemplates, jobs <-chan *CmdList, hosts <-chan *Host, results chan<- string, opts *ConfigOpts) {
	opts.Logger.Info().Msg("Running commands in parallel")
	for list := range jobs {
		fieldsMap := map[string]interface{}{"list": list.Name}
		var cmdLogger zerolog.Logger
		var commandExecuted *Command
		var cmdsRan []string
		var outStructArr []outStruct
		var hasError bool // Tracks if any command in the list failed

		var wg sync.WaitGroup
		hostList := []*Host{}
		for host := range hosts {
			hostList = append(hostList, host)
		}
		println("Total hosts to run commands on:", len(hostList))
		println("Total commands to run:", len(list.Order))

		for _, cmd := range list.Order {
			cmdsRan = append(cmdsRan, cmd)
			println("Running cmd:", cmd, "on", len(hostList), "hosts")
			outputChan := make(chan outStruct, len(hostList))
			errorChan := make(chan error, len(hostList))
			// cmdToRun := opts.Cmds[cmd]
			origCmd := opts.Cmds[cmd]

			for _, host := range hostList {
				wg.Add(1)
				cmdToRun := *origCmd // shallow copy
				commandExecuted = origCmd
				if cmdToRun.Host != host.Host {
					cmdToRun.Host = host.Host
					cmdToRun.RemoteHost = host
				}
				cmdLogger = cmdToRun.GenerateLogger(opts)
				cmdLogger.Info().Fields(fieldsMap).Send()
				print("Running cmd on: ", host.Host, "\n")

				go func(cmd string, host *Host) {
					defer wg.Done()
					currentCmd := cmdToRun.Name
					fieldsMap["cmd"] = currentCmd

					outputArr, runErr := cmdToRun.RunCmd(cmdLogger, opts)
					if runErr != nil {
						cmdLogger.Err(runErr).Send()
						cmdToRun.ExecuteHooks("error", opts)
						errorChan <- runErr
						return
					}

					if list.GetCommandOutputInNotificationsOnSuccess || cmdToRun.Output.InList {
						outputChan <- outStruct{
							CmdName:     currentCmd,
							CmdExecuted: currentCmd,
							Output:      outputArr,
						}
					}
				}(cmd, host)
			}

			wg.Wait()
			close(outputChan)
			close(errorChan)

			for out := range outputChan {
				outStructArr = append(outStructArr, out)
			}
			if len(errorChan) > 0 {
				hasError = true
				runErr := <-errorChan
				if list.NotifyConfig != nil {
					notifyError(cmdLogger, msgTemps, list, cmdsRan, outStructArr, runErr, commandExecuted)
				}
				break
			}

			if !hasError && list.NotifyConfig != nil && list.Notify.OnFailure {
				notifySuccess(cmdLogger, msgTemps, list, cmdsRan, outStructArr)
			}

			if !hasError {
				commandExecuted.ExecuteHooks("success", opts)
			}

		}
		commandExecuted.ExecuteHooks("final", opts)
		results <- "done"
	}
}

func notifyError(logger zerolog.Logger, templates *msgTemplates, list *CmdList, cmdsRan []string, outStructArr []outStruct, err error, cmd *Command) {
	errStruct := map[string]interface{}{
		"listName":  list.Name,
		"CmdsRan":   cmdsRan,
		"CmdOutput": outStructArr,
		"Err":       err,
		"CmdName":   cmd.Name,
		"Command":   cmd.Cmd,
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
	results := make(chan string, configListsLen)

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
		<-results
	}
	opts.closeHostConnections()
}

func (opts *ConfigOpts) ExecuteListOnHosts(lists []string, parallel bool) {

	mTemps := &msgTemplates{
		err:     template.Must(template.New("error.txt").ParseFS(templates, "templates/error.txt")),
		success: template.Must(template.New("success.txt").ParseFS(templates, "templates/success.txt")),
	}
	// for _, l := range opts.CmdConfigLists {
	// 	if !slices.Contains(lists, l.Name) {
	// 		delete(opts.CmdConfigLists, l.Name)
	// 	}
	// }
	configListsLen := len(opts.CmdConfigLists)
	listChan := make(chan *CmdList, configListsLen)
	hostChan := make(chan *Host, len(opts.Hosts))
	results := make(chan string, configListsLen)

	// Start workers
	for w := 1; w <= configListsLen; w++ {
		if parallel {
			go cmdListWorkerExecuteCommandsInParallel(mTemps, listChan, hostChan, results, opts)
		} else {
			go cmdListWorkerWithHosts(mTemps, listChan, hostChan, results, opts)
		}
	}

	// Enqueue jobs
	for listName, cmdConfig := range opts.CmdConfigLists {
		if cmdConfig.Name == "" {
			cmdConfig.Name = listName
		}
		listChan <- cmdConfig
	}
	for _, h := range opts.Hosts {
		if h.isProxyHost {
			continue
		}
		hostChan <- h
		// for _, proxyHost := range h.ProxyHost {
		// 	if proxyHost.isProxyHost {
		// 		continue
		// 	}
		// 	hostChan <- proxyHost
		// }
	}
	close(listChan)
	close(hostChan)

	// Process results
	for a := 1; a <= configListsLen; a++ {
		<-results
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
				Str("backy-cmd", v).Str("hookType", "error").
				Logger()
			cmdLogger.Info().Msgf("Running error hook command %s", v)
			// URGENT: Never returns
			_, _ = errCmd.RunCmd(cmdLogger, opts)
			return
		}

	case "success":
		for _, v := range cmd.Hooks.Success {
			successCmd := opts.Cmds[v]
			cmdLogger := opts.Logger.With().
				Str("backy-cmd", v).Str("hookType", "success").
				Logger()
			cmdLogger.Info().Msgf("Running success hook command %s", v)
			_, _ = successCmd.RunCmd(cmdLogger, opts)
		}
	case "final":
		for _, v := range cmd.Hooks.Final {
			finalCmd := opts.Cmds[v]
			cmdLogger := opts.Logger.With().
				Str("backy-cmd", v).Str("hookType", "final").
				Logger()
			cmdLogger.Info().Msgf("Running final hook command %s", v)
			_, _ = finalCmd.RunCmd(cmdLogger, opts)
		}
	}
}

func (cmd *Command) GenerateLogger(opts *ConfigOpts) zerolog.Logger {
	cmdLogger := opts.Logger.With().
		Str("Backy-cmd", cmd.Name).Str("Host", "local machine").
		Logger()

	if !IsHostLocal(cmd.Host) {
		cmdLogger = opts.Logger.With().
			Str("Backy-cmd", cmd.Name).Str("Host", cmd.Host).
			Logger()
	}
	return cmdLogger
}

func (opts *ConfigOpts) ExecCmdsOnHosts(cmdList []string, hostsList []string) {
	// Iterate over hosts and exec commands
	for _, h := range hostsList {
		host := opts.Hosts[h]
		for _, c := range cmdList {
			cmd := opts.Cmds[c]
			cmd.RemoteHost = host
			cmd.Host = h
			if IsHostLocal(h) {
				_, err := cmd.RunCmd(cmd.GenerateLogger(opts), opts)
				if err != nil {
					opts.Logger.Err(err).Str("host", h).Str("cmd", c).Send()
				}
			} else {

				cmd.Host = host.Host
				opts.Logger.Info().Str("host", h).Str("cmd", c).Send()
				_, err := cmd.RunCmdOnHost(cmd.GenerateLogger(opts), opts)
				if err != nil {
					opts.Logger.Err(err).Str("host", h).Str("cmd", c).Send()
				}
			}
		}
	}
}

func (opts *ConfigOpts) ExecCmdsOnHostsInParallel(cmdList []string, hostsList []string) {
	opts.Logger.Info().Msg("Executing commands in parallel on hosts")
	// Iterate over hosts and exec commands
	for _, c := range cmdList {
		for _, h := range hostsList {
			host := opts.Hosts[h]
			cmd := opts.Cmds[c]
			cmd.RemoteHost = host
			cmd.Host = h
			if IsHostLocal(h) {
				_, err := cmd.RunCmd(cmd.GenerateLogger(opts), opts)
				if err != nil {
					opts.Logger.Err(err).Str("host", h).Str("cmd", c).Send()
				}
			} else {

				cmd.Host = host.Host
				opts.Logger.Info().Str("host", h).Str("cmd", c).Send()
				_, err := cmd.RunCmdOnHost(cmd.GenerateLogger(opts), opts)
				if err != nil {
					opts.Logger.Err(err).Str("host", h).Str("cmd", c).Send()
				}
			}
		}
	}
}

func logCommandOutput(command *Command, cmdOutBuf bytes.Buffer, cmdCtxLogger zerolog.Logger, outputArr []string) []string {

	outScanner := bufio.NewScanner(&cmdOutBuf)

	for outScanner.Scan() {
		outMap := make(map[string]interface{})
		outMap["cmd"] = command.Name
		outMap["output"] = outScanner.Text()

		if str, ok := outMap["output"].(string); ok {
			outputArr = append(outputArr, str)
		}
		if command.Output.ToLog {
			cmdCtxLogger.Info().Fields(outMap).Send()
		}
	}
	return outputArr
}

// func executeUserCommands() []string {

// }

// // parseRemoteSources parses source and validates fields using sourceType
// func (c *Command) parseRemoteSources(source, sourceType string) {
// 	switch sourceType {

// 	}
// }
