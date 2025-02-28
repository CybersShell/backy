// ssh.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var PrivateKeyExtraInfoErr = errors.New("Private key may be encrypted. \nIf encrypted, make sure the password is specified correctly in the correct section. This may be done in one of three ways: \n privatekeypassword: env:PR_KEY_PASS \n privatekeypassword: file:/path/to/password-file \n privatekeypassword: password (not recommended). \n ")
var TS = strings.TrimSpace

// ConnectToHost connects to a host by looking up the config values in the file ~/.ssh/config
// It uses any set values and looks up an unset values in the config files
// remoteConfig is modified directly. The *ssh.Client is returned as part of remoteConfig,
// If configFile is empty, any required configuration is looked up in the default config files
// If any value is not found, defaults are used
func (remoteConfig *Host) ConnectToHost(opts *ConfigOpts) error {

	var connectErr error

	if TS(remoteConfig.ConfigFilePath) == "" {
		remoteConfig.useDefaultConfig = true
	}

	khPathErr := remoteConfig.GetKnownHosts()

	if khPathErr != nil {
		return khPathErr
	}

	if remoteConfig.ClientConfig == nil {
		remoteConfig.ClientConfig = &ssh.ClientConfig{}
	}

	var configFile *os.File

	var sshConfigFileOpenErr error

	if !remoteConfig.useDefaultConfig {
		var err error
		remoteConfig.ConfigFilePath, err = getFullPathWithHomeDir(remoteConfig.ConfigFilePath)
		if err != nil {
			return err
		}
		configFile, sshConfigFileOpenErr = os.Open(remoteConfig.ConfigFilePath)
		if sshConfigFileOpenErr != nil {
			return sshConfigFileOpenErr
		}
	} else {
		defaultConfig, _ := getFullPathWithHomeDir("~/.ssh/config")
		configFile, sshConfigFileOpenErr = os.Open(defaultConfig)
		if sshConfigFileOpenErr != nil {
			return sshConfigFileOpenErr
		}
	}
	remoteConfig.SSHConfigFile = &sshConfigFile{}
	remoteConfig.SSHConfigFile.DefaultUserSettings = ssh_config.DefaultUserSettings
	var decodeErr error
	remoteConfig.SSHConfigFile.SshConfigFile, decodeErr = ssh_config.Decode(configFile)
	if decodeErr != nil {
		return decodeErr
	}

	err := remoteConfig.GetProxyJumpFromConfig(opts.Hosts)

	if err != nil {
		return err
	}

	if remoteConfig.ProxyHost != nil {
		for _, proxyHost := range remoteConfig.ProxyHost {
			err := proxyHost.GetProxyJumpConfig(opts.Hosts, opts)
			opts.Logger.Info().Msgf("Proxy host: %s", proxyHost.Host)
			if err != nil {
				return err
			}
		}
	}

	remoteConfig.ClientConfig.Timeout = time.Second * 30

	remoteConfig.GetPrivateKeyFileFromConfig()

	remoteConfig.GetPort()

	remoteConfig.GetHostName()

	remoteConfig.CombineHostNameWithPort()

	remoteConfig.GetSshUserFromConfig()

	if remoteConfig.HostName == "" {
		return errors.Errorf("No hostname found or specified for host %s", remoteConfig.Host)
	}

	err = remoteConfig.GetAuthMethods(opts)
	if err != nil {
		return err
	}

	hostKeyCallback, err := knownhosts.New(remoteConfig.KnownHostsFile)
	if err != nil {
		return errors.Wrap(err, "could not create hostkeycallback function")
	}
	remoteConfig.ClientConfig.HostKeyCallback = hostKeyCallback
	// opts.Logger.Info().Str("user", remoteConfig.ClientConfig.User).Send()

	remoteConfig.SshClient, connectErr = remoteConfig.ConnectThroughBastion(opts.Logger)
	if connectErr != nil {
		return connectErr
	}
	if remoteConfig.SshClient != nil {
		opts.Hosts[remoteConfig.Host] = remoteConfig
		return nil
	}

	opts.Logger.Info().Msgf("Connecting to host %s", remoteConfig.HostName)
	remoteConfig.SshClient, connectErr = ssh.Dial("tcp", remoteConfig.HostName, remoteConfig.ClientConfig)
	if connectErr != nil {
		return connectErr
	}

	opts.Hosts[remoteConfig.Host] = remoteConfig
	return nil
}

func (remoteHost *Host) GetSshUserFromConfig() {

	if TS(remoteHost.User) == "" {

		remoteHost.User, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "User")

		if TS(remoteHost.User) == "" {

			remoteHost.User = remoteHost.SSHConfigFile.DefaultUserSettings.Get(remoteHost.Host, "User")

			if TS(remoteHost.User) == "" {

				currentUser, _ := user.Current()

				remoteHost.User = currentUser.Username
			}
		}
	}
	remoteHost.ClientConfig.User = remoteHost.User
}

func (remoteHost *Host) GetAuthMethods(opts *ConfigOpts) error {
	var signer ssh.Signer
	var err error
	var privateKey []byte

	remoteHost.Password = strings.TrimSpace(remoteHost.Password)

	remoteHost.PrivateKeyPassword = strings.TrimSpace(remoteHost.PrivateKeyPassword)

	remoteHost.PrivateKeyPath = strings.TrimSpace(remoteHost.PrivateKeyPath)

	if remoteHost.PrivateKeyPath != "" {

		privateKey, err = os.ReadFile(remoteHost.PrivateKeyPath)

		if err != nil {
			return err
		}

		remoteHost.PrivateKeyPassword, err = GetPrivateKeyPassword(remoteHost.PrivateKeyPassword, opts, opts.Logger)

		if err != nil {
			return err
		}

		if remoteHost.PrivateKeyPassword == "" {

			signer, err = ssh.ParsePrivateKey(privateKey)

			if err != nil {
				return errors.Errorf("Failed to open private key file %s: %v \n\n %v", remoteHost.PrivateKeyPath, err, PrivateKeyExtraInfoErr)
			}

			remoteHost.ClientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		} else {

			signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(remoteHost.PrivateKeyPassword))

			if err != nil {
				return errors.Errorf("Failed to open private key file %s: %v \n\n %v", remoteHost.PrivateKeyPath, err, PrivateKeyExtraInfoErr)
			}

			remoteHost.ClientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		}
	}

	if remoteHost.Password == "" {

		remoteHost.Password, err = GetPassword(remoteHost.Password, opts, opts.Logger)

		if err != nil {

			return err
		}

		remoteHost.ClientConfig.Auth = append(remoteHost.ClientConfig.Auth, ssh.Password(remoteHost.Password))
	}

	return nil
}

// GetPrivateKeyFromConfig checks to see if the privateKeyPath is empty.
// If not, it keeps the value.
// If empty, the key is looked for in the specified config file.
// If that path is empty, the default config file is searched.
// If not found in the default file, the privateKeyPath is set to ~/.ssh/id_rsa
func (remoteHost *Host) GetPrivateKeyFileFromConfig() {
	var identityFile string
	if remoteHost.PrivateKeyPath == "" {
		identityFile, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "IdentityFile")
		if identityFile == "" {
			identityFile, _ = remoteHost.SSHConfigFile.DefaultUserSettings.GetStrict(remoteHost.Host, "IdentityFile")
			if identityFile == "" {
				identityFile = "~/.ssh/id_rsa"
			}
		}
	}
	if identityFile == "" {
		identityFile = remoteHost.PrivateKeyPath
	}

	remoteHost.PrivateKeyPath, _ = getFullPathWithHomeDir(identityFile)
}

// GetPort checks if the port from the config file is 0
// If it is the port is searched in the SSH config file(s)
func (remoteHost *Host) GetPort() {
	port := fmt.Sprintf("%d", remoteHost.Port)
	// port specifed?
	// port will be 0 if missing from backy config
	if port == "0" {
		port, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "Port")

		if port == "" {

			// get port from default SSH config file
			port = remoteHost.SSHConfigFile.DefaultUserSettings.Get(remoteHost.Host, "Port")

			// set port to be default
			if port == "" {
				port = "22"
			}
		}
	}
	portNum, _ := strconv.ParseUint(port, 10, 16)
	remoteHost.Port = uint16(portNum)
}

func (remoteHost *Host) CombineHostNameWithPort() {

	// if the port is already in the HostName, leave it
	if strings.HasSuffix(remoteHost.HostName, fmt.Sprintf(":%d", remoteHost.Port)) {
		return
	}

	remoteHost.HostName = fmt.Sprintf("%s:%d", remoteHost.HostName, remoteHost.Port)
}

func (remoteHost *Host) GetHostName() {

	if remoteHost.HostName == "" {
		remoteHost.HostName, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "HostName")
		if remoteHost.HostName == "" {
			remoteHost.HostName = remoteHost.SSHConfigFile.DefaultUserSettings.Get(remoteHost.Host, "HostName")
		}
	}
}

func (remoteHost *Host) ConnectThroughBastion(log zerolog.Logger) (*ssh.Client, error) {
	if remoteHost.ProxyHost == nil {
		return nil, nil
	}

	log.Info().Msgf("Connecting to proxy host %s", remoteHost.ProxyHost[0].HostName)

	// connect to the bastion host
	bClient, err := ssh.Dial("tcp", remoteHost.ProxyHost[0].HostName, remoteHost.ProxyHost[0].ClientConfig)
	if err != nil {
		return nil, err
	}
	remoteHost.ProxyHost[0].SshClient = bClient

	// Dial a connection to the service host, from the bastion
	conn, err := bClient.Dial("tcp", remoteHost.HostName)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("Connecting to host %s", remoteHost.HostName)
	ncc, chans, reqs, err := ssh.NewClientConn(conn, remoteHost.HostName, remoteHost.ClientConfig)
	if err != nil {
		return nil, err
	}

	sClient := ssh.NewClient(ncc, chans, reqs)

	return sClient, nil
}

// GetKnownHosts resolves the host's KnownHosts file if it is defined
// if not defined, the default location for this file is used
func (remotehHost *Host) GetKnownHosts() error {
	var knownHostsFileErr error
	if TS(remotehHost.KnownHostsFile) != "" {
		remotehHost.KnownHostsFile, knownHostsFileErr = getFullPathWithHomeDir(remotehHost.KnownHostsFile)
		return knownHostsFileErr
	}
	remotehHost.KnownHostsFile, knownHostsFileErr = getFullPathWithHomeDir("~/.ssh/known_hosts")
	return knownHostsFileErr
}

func GetPrivateKeyPassword(key string, opts *ConfigOpts, log zerolog.Logger) (string, error) {

	var prKeyPassword string
	if strings.HasPrefix(key, "file:") {
		privKeyPassFilePath := strings.TrimPrefix(key, "file:")
		privKeyPassFilePath, _ = getFullPathWithHomeDir(privKeyPassFilePath)
		keyFile, keyFileErr := os.Open(privKeyPassFilePath)
		if keyFileErr != nil {
			return "", errors.Errorf("Private key password file %s failed to open. \n Make sure it is accessible and correct.", privKeyPassFilePath)
		}
		passwordScanner := bufio.NewScanner(keyFile)
		for passwordScanner.Scan() {
			prKeyPassword = passwordScanner.Text()
		}
	} else if strings.HasPrefix(key, "env:") {
		privKey := strings.TrimPrefix(key, "env:")
		privKey = strings.TrimPrefix(privKey, "${")
		privKey = strings.TrimSuffix(privKey, "}")
		privKey = strings.TrimPrefix(privKey, "$")
		prKeyPassword = os.Getenv(privKey)
	} else {
		prKeyPassword = key
	}
	prKeyPassword = GetVaultKey(prKeyPassword, opts, opts.Logger)
	return prKeyPassword, nil
}

// GetPassword gets any password
func GetPassword(pass string, opts *ConfigOpts, log zerolog.Logger) (string, error) {

	pass = strings.TrimSpace(pass)
	if pass == "" {
		return "", nil
	}
	var password string
	if strings.HasPrefix(pass, "file:") {
		passFilePath := strings.TrimPrefix(pass, "file:")
		passFilePath, _ = getFullPathWithHomeDir(passFilePath)
		keyFile, keyFileErr := os.Open(passFilePath)
		if keyFileErr != nil {
			return "", errors.New("Password file failed to open")
		}
		passwordScanner := bufio.NewScanner(keyFile)
		for passwordScanner.Scan() {
			password = passwordScanner.Text()
		}
	} else if strings.HasPrefix(pass, "env:") {
		passEnv := strings.TrimPrefix(pass, "env:")
		passEnv = strings.TrimPrefix(passEnv, "${")
		passEnv = strings.TrimSuffix(passEnv, "}")
		passEnv = strings.TrimPrefix(passEnv, "$")
		password = os.Getenv(passEnv)
	} else {
		password = pass
	}
	password = GetVaultKey(password, opts, opts.Logger)

	return password, nil
}

func (remoteConfig *Host) GetProxyJumpFromConfig(hosts map[string]*Host) error {

	proxyJump, _ := remoteConfig.SSHConfigFile.SshConfigFile.Get(remoteConfig.Host, "ProxyJump")
	if proxyJump == "" {
		proxyJump = remoteConfig.SSHConfigFile.DefaultUserSettings.Get(remoteConfig.Host, "ProxyJump")
	}
	if remoteConfig.ProxyJump == "" && proxyJump != "" {
		remoteConfig.ProxyJump = proxyJump
	}
	proxyJumpHosts := strings.Split(remoteConfig.ProxyJump, ",")
	if remoteConfig.ProxyHost == nil && len(proxyJumpHosts) == 1 {
		remoteConfig.ProxyJump = proxyJump
		proxyHost, proxyHostFound := hosts[proxyJump]
		if proxyHostFound {
			remoteConfig.ProxyHost = append(remoteConfig.ProxyHost, proxyHost)
		} else {
			if proxyJump != "" {
				newProxy := &Host{Host: proxyJump}
				remoteConfig.ProxyHost = append(remoteConfig.ProxyHost, newProxy)
			}
		}
	}

	return nil
}

func (remoteConfig *Host) GetProxyJumpConfig(hosts map[string]*Host, opts *ConfigOpts) error {

	if TS(remoteConfig.ConfigFilePath) == "" {
		remoteConfig.useDefaultConfig = true
	}

	khPathErr := remoteConfig.GetKnownHosts()

	if khPathErr != nil {
		return khPathErr
	}
	if remoteConfig.ClientConfig == nil {
		remoteConfig.ClientConfig = &ssh.ClientConfig{}
	}
	var configFile *os.File
	var sshConfigFileOpenErr error
	if !remoteConfig.useDefaultConfig {

		configFile, sshConfigFileOpenErr = os.Open(remoteConfig.ConfigFilePath)
		if sshConfigFileOpenErr != nil {
			return sshConfigFileOpenErr
		}
	} else {
		defaultConfig, _ := getFullPathWithHomeDir("~/.ssh/config")
		configFile, sshConfigFileOpenErr = os.Open(defaultConfig)
		if sshConfigFileOpenErr != nil {
			return sshConfigFileOpenErr
		}
	}
	remoteConfig.SSHConfigFile = &sshConfigFile{}
	remoteConfig.SSHConfigFile.DefaultUserSettings = ssh_config.DefaultUserSettings
	var decodeErr error
	remoteConfig.SSHConfigFile.SshConfigFile, decodeErr = ssh_config.Decode(configFile)
	if decodeErr != nil {
		return decodeErr
	}
	remoteConfig.GetPrivateKeyFileFromConfig()
	remoteConfig.GetPort()
	remoteConfig.GetHostName()
	remoteConfig.CombineHostNameWithPort()
	remoteConfig.GetSshUserFromConfig()
	remoteConfig.isProxyHost = true
	if remoteConfig.HostName == "" {
		return errors.Errorf("No hostname found or specified for host %s", remoteConfig.Host)
	}
	err := remoteConfig.GetAuthMethods(opts)
	if err != nil {
		return err
	}

	// TODO: Add value/option to config for host key and add bool to check for host key
	hostKeyCallback, err := knownhosts.New(remoteConfig.KnownHostsFile)
	if err != nil {
		return fmt.Errorf("could not create hostkeycallback function: %v", err)
	}
	remoteConfig.ClientConfig.HostKeyCallback = hostKeyCallback
	hosts[remoteConfig.Host] = remoteConfig

	return nil
}

func (command *Command) RunCmdSSH(cmdCtxLogger zerolog.Logger, opts *ConfigOpts) ([]string, error) {
	var (
		ArgsStr       string
		cmdOutBuf     bytes.Buffer
		cmdOutWriters io.Writer

		envVars = environmentVars{
			file: command.Env,
			env:  command.Environment,
		}
	)
	// Getting the command type must be done before concatenating the arguments
	command = getCommandTypeAndSetCommandInfo(command)

	// Prepare command arguments
	for _, v := range command.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	cmdCtxLogger.Info().
		Str("Command", command.Name).
		Str("Host", *command.Host).
		Msgf("Running %s on host %s", getCommandTypeAndSetCommandInfoLabel(command.Type), *command.Host)

	// cmdCtxLogger.Debug().Str("cmd", command.Cmd).Strs("args", command.Args).Send()

	// Ensure SSH client is connected
	if command.RemoteHost.SshClient == nil {
		if err := command.RemoteHost.ConnectToHost(opts); err != nil {
			return nil, fmt.Errorf("failed to connect to host: %w", err)
		}
	}

	// Create new SSH session
	commandSession, err := command.RemoteHost.createSSHSession(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer commandSession.Close()

	// Inject environment variables
	injectEnvIntoSSH(envVars, commandSession, opts, cmdCtxLogger)

	// Set output writers
	cmdOutWriters = io.MultiWriter(&cmdOutBuf)
	if IsCmdStdOutEnabled() {
		cmdOutWriters = io.MultiWriter(os.Stdout, &cmdOutBuf)
	}
	commandSession.Stdout = cmdOutWriters
	commandSession.Stderr = cmdOutWriters

	// Handle command execution based on type
	switch command.Type {
	case ScriptCT:
		return command.runScript(commandSession, cmdCtxLogger, &cmdOutBuf)
	case RemoteScriptCT:
		return command.runRemoteScript(commandSession, cmdCtxLogger, &cmdOutBuf)
	case ScriptFileCT:
		return command.runScriptFile(commandSession, cmdCtxLogger, &cmdOutBuf)
	case PackageCT:
		if command.PackageOperation == PackOpCheckVersion {
			commandSession.Stderr = nil
			// Execute the package version command remotely
			// Parse the output of package version command
			// Compare versions
			// Check if a specific version is specified
			commandSession.Stdout = nil
			return checkPackageVersion(cmdCtxLogger, command, commandSession, cmdOutBuf)
		} else {
			if command.Shell != "" {
				ArgsStr = fmt.Sprintf("%s -c '%s %s'", command.Shell, command.Cmd, ArgsStr)
			} else {
				ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)
			}
			cmdCtxLogger.Debug().Str("cmd + args", ArgsStr).Send()
			// Run simple command
			if err := commandSession.Run(ArgsStr); err != nil {
				return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.OutputToLog), fmt.Errorf("error running command: %w", err)
			}
		}
	default:
		if command.Shell != "" {
			ArgsStr = fmt.Sprintf("%s -c '%s %s'", command.Shell, command.Cmd, ArgsStr)
		} else {
			ArgsStr = fmt.Sprintf("%s %s", command.Cmd, ArgsStr)
		}
		cmdCtxLogger.Debug().Str("cmd + args", ArgsStr).Send()
		// Run simple command
		if err := commandSession.Run(ArgsStr); err != nil {
			return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.OutputToLog), fmt.Errorf("error running command: %w", err)
		}
	}

	return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.OutputToLog), nil
}

func checkPackageVersion(cmdCtxLogger zerolog.Logger, command *Command, commandSession *ssh.Session, cmdOutBuf bytes.Buffer) ([]string, error) {
	cmdCtxLogger.Info().Str("package", command.PackageName).Msg("Checking package versions")
	// Prepare command arguments
	ArgsStr := command.Cmd
	for _, v := range command.Args {
		ArgsStr += fmt.Sprintf(" %s", v)
	}

	var err error
	var cmdOut []byte

	if cmdOut, err = commandSession.CombinedOutput(ArgsStr); err != nil {
		cmdOutBuf.Write(cmdOut)

		_, parseErr := parsePackageVersion(string(cmdOut), cmdCtxLogger, command, cmdOutBuf)
		if parseErr != nil {
			return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.OutputToLog), fmt.Errorf("error: package %s not listed: %w", command.PackageName, err)
		}
		return collectOutput(&cmdOutBuf, command.Name, cmdCtxLogger, command.OutputToLog), fmt.Errorf("error running %s: %w", ArgsStr, err)
	}

	return parsePackageVersion(string(cmdOut), cmdCtxLogger, command, cmdOutBuf)
}

// getCommandTypeAndSetCommandInfoLabel returns a human-readable label for the command type.
func getCommandTypeAndSetCommandInfoLabel(commandType CommandType) string {
	if !commandType.IsACommandType() {
		return "command"
	}
	return fmt.Sprintf("%s command", commandType)
}

// runScript handles the execution of inline scripts.
func (command *Command) runScript(session *ssh.Session, cmdCtxLogger zerolog.Logger, outputBuf *bytes.Buffer) ([]string, error) {
	script, err := command.prepareScriptBuffer()
	if err != nil {
		return nil, err
	}
	session.Stdin = script

	if err := session.Shell(); err != nil {
		return nil, fmt.Errorf("error starting shell: %w", err)
	}

	if err := session.Wait(); err != nil {
		return collectOutput(outputBuf, command.Name, cmdCtxLogger, true), fmt.Errorf("error waiting for shell: %w", err)
	}

	return collectOutput(outputBuf, command.Name, cmdCtxLogger, command.GetOutput), nil
}

// runScriptFile handles the execution of script files.
func (command *Command) runScriptFile(session *ssh.Session, cmdCtxLogger zerolog.Logger, outputBuf *bytes.Buffer) ([]string, error) {
	script, err := command.prepareScriptFileBuffer()
	if err != nil {
		return nil, err
	}
	session.Stdin = script

	if err := session.Shell(); err != nil {
		return nil, fmt.Errorf("error starting shell: %w", err)
	}

	if err := session.Wait(); err != nil {
		return collectOutput(outputBuf, command.Name, cmdCtxLogger, true), fmt.Errorf("error waiting for shell: %w", err)
	}

	return collectOutput(outputBuf, command.Name, cmdCtxLogger, command.OutputToLog), nil
}

// prepareScriptBuffer prepares a buffer for inline scripts.
func (command *Command) prepareScriptBuffer() (*bytes.Buffer, error) {
	var buffer bytes.Buffer

	if command.ScriptEnvFile != "" {
		envBuffer, err := readFileToBuffer(command.ScriptEnvFile)
		if err != nil {
			return nil, err
		}
		buffer.Write(envBuffer.Bytes())
		buffer.WriteByte('\n')
	}

	buffer.WriteString(command.Cmd + "\n")
	return &buffer, nil
}

// prepareScriptFileBuffer prepares a buffer for script files.
func (command *Command) prepareScriptFileBuffer() (*bytes.Buffer, error) {
	var buffer bytes.Buffer

	// Handle script environment file
	if command.ScriptEnvFile != "" {
		envBuffer, err := readFileToBuffer(command.ScriptEnvFile)
		if err != nil {
			return nil, err
		}
		buffer.Write(envBuffer.Bytes())
		buffer.WriteByte('\n')
	}

	// Handle script file
	scriptBuffer, err := readFileToBuffer(command.Cmd)
	if err != nil {
		return nil, err
	}
	buffer.Write(scriptBuffer.Bytes())

	return &buffer, nil
}

// runRemoteScript handles the execution of remote scripts
func (command *Command) runRemoteScript(session *ssh.Session, cmdCtxLogger zerolog.Logger, outputBuf *bytes.Buffer) ([]string, error) {
	script, err := command.Fetcher.Fetch(command.Cmd)
	if err != nil {
		return nil, err
	}
	if command.Shell == "" {
		command.Shell = "sh"
	}
	session.Stdin = bytes.NewReader(script)
	err = session.Run(command.Shell)

	if err != nil {
		return collectOutput(outputBuf, command.Name, cmdCtxLogger, command.OutputToLog), fmt.Errorf("error running remote script: %w", err)
	}

	return collectOutput(outputBuf, command.Name, cmdCtxLogger, command.OutputToLog), nil
}

// readFileToBuffer reads a file into a buffer.
func readFileToBuffer(filePath string) (*bytes.Buffer, error) {
	resolvedPath, err := getFullPathWithHomeDir(filePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, file); err != nil {
		return nil, err
	}

	return &buffer, nil
}

// collectOutput collects output from a buffer and logs it.
func collectOutput(buf *bytes.Buffer, commandName string, logger zerolog.Logger, wantOutput bool) []string {
	var outputArr []string
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		outputArr = append(outputArr, line)
		if wantOutput {
			logger.Info().Str("cmd", commandName).Str("output", line).Send()
		}
	}
	return outputArr
}

// createSSHSession attempts to create a new SSH session and retries on failure.
func (h *Host) createSSHSession(opts *ConfigOpts) (*ssh.Session, error) {
	session, err := h.SshClient.NewSession()
	if err == nil {
		return session, nil
	}

	// Retry connection and session creation
	if connErr := h.ConnectToHost(opts); connErr != nil {
		return nil, fmt.Errorf("session creation failed: %v, connection retry failed: %v", err, connErr)
	}
	return h.SshClient.NewSession()
}

func (h *Host) DetectOS(opts *ConfigOpts) (string, error) {
	err := h.ConnectToHost(opts)

	if err != nil {
		return "", err
	}
	var session *ssh.Session
	session, err = h.createSSHSession(opts)
	if err != nil {
		return "", err
	}
	// Execute the "uname -a" command on the remote machine
	output, err := session.CombinedOutput("uname")
	if err != nil {
		return "", fmt.Errorf("failed to execute OS detection command: %v", err)
	}

	// Parse the output to determine the OS
	osName := string(output)
	return osName, nil
}

func CheckIfHostHasHostName(host string) (bool, string) {
	HostName, err := ssh_config.DefaultUserSettings.GetStrict(host, "HostName")
	if err != nil {
		return false, ""
	}
	println(HostName)
	return HostName != "", HostName
}
