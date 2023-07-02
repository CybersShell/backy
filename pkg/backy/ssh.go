// ssh.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"bufio"
	"fmt"
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

var PrivateKeyExtraInfoErr = errors.New("Private key may be encrypted. \nIf encrypted, make sure the password is specified correctly in the correct section: \n privatekeypassword: env:PR_KEY_PASS \n privatekeypassword: file:/path/to/password-file \n privatekeypassword: password (not recommended). \n ")
var TS = strings.TrimSpace

// ConnectToSSHHost connects to a host by looking up the config values in the directory ~/.ssh/config
// It uses any set values and looks up an unset values in the config files
// It returns an ssh.Client used to run commands against.
// If configFile is empty, any required configuration is looked up in the default config files
// If any value is not found, defaults are used
func (remoteConfig *Host) ConnectToSSHHost(opts *ConfigOpts, config *ConfigFile) error {

	// var sshClient *ssh.Client
	var connectErr error

	if TS(remoteConfig.ConfigFilePath) == "" {
		remoteConfig.useDefaultConfig = true
	}
	khPath, khPathErr := GetKnownHosts(remoteConfig.KnownHostsFile)

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
		defaultConfig, _ := resolveDir("~/.ssh/config")
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

	err := remoteConfig.GetProxyJumpFromConfig(config.Hosts)
	if err != nil {
		return err
	}
	if remoteConfig.ProxyHost != nil {
		for _, proxyHost := range remoteConfig.ProxyHost {
			err := proxyHost.GetProxyJumpConfig(config.Hosts, opts)
			opts.ConfigFile.Logger.Info().Msgf("Proxy host: %s", proxyHost.Host)
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

	hostKeyCallback, err := knownhosts.New(khPath)
	if err != nil {
		return errors.Wrap(err, "could not create hostkeycallback function")
	}
	remoteConfig.ClientConfig.HostKeyCallback = hostKeyCallback
	opts.ConfigFile.Logger.Info().Str("user", remoteConfig.ClientConfig.User).Send()

	remoteConfig.SshClient, connectErr = remoteConfig.ConnectThroughBastion(opts.ConfigFile.Logger)
	if connectErr != nil {
		return connectErr
	}
	if remoteConfig.SshClient != nil {
		config.Hosts[remoteConfig.Host] = remoteConfig
		return nil
	}

	opts.ConfigFile.Logger.Info().Msgf("Connecting to host %s", remoteConfig.HostName)
	remoteConfig.SshClient, connectErr = ssh.Dial("tcp", remoteConfig.HostName, remoteConfig.ClientConfig)
	if connectErr != nil {
		return connectErr
	}
	config.Hosts[remoteConfig.Host] = remoteConfig
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
		remoteHost.PrivateKeyPassword, err = GetPrivateKeyPassword(remoteHost.PrivateKeyPassword, opts, opts.ConfigFile.Logger)
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
		remoteHost.Password, err = GetPassword(remoteHost.Password, opts, opts.ConfigFile.Logger)
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

	remoteHost.PrivateKeyPath, _ = resolveDir(identityFile)
}

// GetPort checks if the port from the config file is 0
// If it is the port is searched in the SSH config file(s)
func (remoteHost *Host) GetPort() {
	port := fmt.Sprintf("%d", remoteHost.Port)
	// port specifed?
	if port == "0" {
		port, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "Port")
		if port == "" {
			port = remoteHost.SSHConfigFile.DefaultUserSettings.Get(remoteHost.Host, "Port")
			if port == "" {
				port = "22"
			}
		}
	}
	portNum, _ := strconv.ParseUint(port, 10, 16)
	remoteHost.Port = uint16(portNum)
}

func (remoteHost *Host) CombineHostNameWithPort() {
	port := fmt.Sprintf(":%d", remoteHost.Port)
	if strings.HasSuffix(remoteHost.HostName, port) {
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
	// sClient is an ssh client connected to the service host, through the bastion host.

	return sClient, nil
}

func GetKnownHosts(khPath string) (string, error) {
	if TS(khPath) != "" {
		return resolveDir(khPath)
	}
	return resolveDir("~/.ssh/known_hosts")
}

func GetPrivateKeyPassword(key string, opts *ConfigOpts, log zerolog.Logger) (string, error) {
	var prKeyPassword string
	if strings.HasPrefix(key, "file:") {
		privKeyPassFilePath := strings.TrimPrefix(key, "file:")
		privKeyPassFilePath, _ = resolveDir(privKeyPassFilePath)
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
	prKeyPassword = GetVaultKey(prKeyPassword, opts, opts.ConfigFile.Logger)
	return prKeyPassword, nil
}

func GetPassword(pass string, opts *ConfigOpts, log zerolog.Logger) (string, error) {
	pass = strings.TrimSpace(pass)
	if pass == "" {
		return "", nil
	}
	var password string
	if strings.HasPrefix(pass, "file:") {
		passFilePath := strings.TrimPrefix(pass, "file:")
		passFilePath, _ = resolveDir(passFilePath)
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
	password = GetVaultKey(password, opts, opts.ConfigFile.Logger)

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

	khPath, khPathErr := GetKnownHosts(remoteConfig.KnownHostsFile)

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
		defaultConfig, _ := resolveDir("~/.ssh/config")
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
	hostKeyCallback, err := knownhosts.New(khPath)
	if err != nil {
		return errors.Wrap(err, "could not create hostkeycallback function")
	}
	remoteConfig.ClientConfig.HostKeyCallback = hostKeyCallback
	hosts[remoteConfig.Host] = remoteConfig

	return nil
}
