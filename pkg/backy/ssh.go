// ssh.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var ErrPrivateKeyFileFailedToOpen = errors.New("Private key file failed to open.")
var TS = strings.TrimSpace

// ConnectToSSHHost connects to a host by looking up the config values in the directory ~/.ssh/config
// It uses any set values and looks up an unset values in the config files
// It returns an ssh.Client used to run commands against.
func (remoteConfig *Host) ConnectToSSHHost(log *zerolog.Logger) (*ssh.Client, error) {

	var sshClient *ssh.Client
	var connectErr error

	// TODO: add JumpHost config check

	// if !remoteConfig.UseConfigFiles {
	// 	log.Info().Msg("Not using config files")
	// }
	if TS(remoteConfig.ConfigFilePath) == "" {
		remoteConfig.useDefaultConfig = true
	}

	khPath, khPathErr := GetKnownHosts(remoteConfig.KnownHostsFile)

	if khPathErr != nil {
		return nil, khPathErr
	}
	if remoteConfig.ClientConfig == nil {
		remoteConfig.ClientConfig = &ssh.ClientConfig{}
	}
	var sshConfigFile *os.File
	var sshConfigFileOpenErr error
	if !remoteConfig.useDefaultConfig {

		sshConfigFile, sshConfigFileOpenErr = os.Open(remoteConfig.ConfigFilePath)
		if sshConfigFileOpenErr != nil {
			return nil, sshConfigFileOpenErr
		}
	} else {
		defaultConfig, _ := resolveDir("~/.ssh/config")
		sshConfigFile, sshConfigFileOpenErr = os.Open(defaultConfig)
		if sshConfigFileOpenErr != nil {
			return nil, sshConfigFileOpenErr
		}
	}
	remoteConfig.SSHConfigFile.DefaultUserSettings = ssh_config.DefaultUserSettings

	cfg, decodeErr := ssh_config.Decode(sshConfigFile)
	if decodeErr != nil {
		return nil, decodeErr
	}
	remoteConfig.SSHConfigFile.SshConfigFile = cfg
	remoteConfig.GetPrivateKeyFromConfig()
	remoteConfig.GetHostNameWithPort()
	remoteConfig.GetSshUserFromConfig()
	log.Info().Msgf("Port: %v", remoteConfig.Port)
	if remoteConfig.HostName == "" {
		return nil, errors.New("No hostname found or specified")
	}
	err := remoteConfig.GetAuthMethods()
	if err != nil {
		return nil, err
	}

	// TODO: Add value/option to config for host key and add bool to check for host key
	hostKeyCallback, err := knownhosts.New(khPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not create hostkeycallback function")
	}
	remoteConfig.ClientConfig.HostKeyCallback = hostKeyCallback
	log.Info().Str("user", remoteConfig.ClientConfig.User).Send()

	log.Info().Msgf("Connecting to host %s", remoteConfig.HostName)
	remoteConfig.ClientConfig.Timeout = time.Second * 30
	sshClient, connectErr = ssh.Dial("tcp", remoteConfig.HostName, remoteConfig.ClientConfig)
	if connectErr != nil {
		return nil, connectErr
	}
	return sshClient, nil
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
func (remoteHost *Host) GetAuthMethods() error {
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
		remoteHost.PrivateKeyPassword, err = GetPrivateKeyPassword(remoteHost.PrivateKeyPassword)
		if err != nil {
			return err
		}
		if remoteHost.PrivateKeyPassword == "" {
			signer, err = ssh.ParsePrivateKey(privateKey)
			if err != nil {
				return ErrPrivateKeyFileFailedToOpen
			}
			remoteHost.ClientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(remoteHost.PrivateKeyPassword))
			if err != nil {
				return err
			}
			remoteHost.ClientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		}
	}
	if remoteHost.Password == "" {
		remoteHost.Password, err = GetPassword(remoteHost.Password)
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
// If that path is empty, the default config file is searched
// If not found in the default file, the privateKeyPath is set to ~/.ssh/id_rsa
func (remoteHost *Host) GetPrivateKeyFromConfig() {
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

// GetHostNameWithPort checks if the port from the config file is 0
// If it is the port is searched in the SSH config file(s)
func (remoteHost *Host) GetHostNameWithPort() {
	port := fmt.Sprintf("%v", remoteHost.Port)

	if remoteHost.HostName == "" {
		remoteHost.HostName, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "HostName")
		if remoteHost.HostName == "" {
			remoteHost.HostName = remoteHost.SSHConfigFile.DefaultUserSettings.Get(remoteHost.Host, "HostName")
		}
	}
	// no port specifed
	if port == "0" {
		port, _ = remoteHost.SSHConfigFile.SshConfigFile.Get(remoteHost.Host, "Port")
		if port == "" {
			port = remoteHost.SSHConfigFile.DefaultUserSettings.Get(remoteHost.Host, "Port")
			if port == "" {
				port = "22"
			}
		}
		println(port)
	}
	if !strings.HasSuffix(remoteHost.HostName, ":"+port) {
		remoteHost.HostName = remoteHost.HostName + ":" + port
	}
}

func (remoteHost *Host) ConnectThroughBastion() (*ssh.Client, error) {
	// connect to the bastion host
	bClient, err := ssh.Dial("tcp", remoteHost.ProxyHost.HostName, remoteHost.ProxyHost.ClientConfig)
	if err != nil {
		return nil, err
	}

	// Dial a connection to the service host, from the bastion
	conn, err := bClient.Dial("tcp", remoteHost.HostName)
	if err != nil {
		return nil, err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, remoteHost.HostName, remoteHost.ClientConfig)
	if err != nil {
		log.Fatal(err)
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

func GetPrivateKeyPassword(key string) (string, error) {
	var prKeyPassword string
	if strings.HasPrefix(key, "file:") {
		privKeyPassFilePath := strings.TrimPrefix(key, "file:")
		privKeyPassFilePath, _ = resolveDir(privKeyPassFilePath)
		keyFile, keyFileErr := os.Open(privKeyPassFilePath)
		if keyFileErr != nil {
			return "", ErrPrivateKeyFileFailedToOpen
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
	return prKeyPassword, nil
}

func GetPassword(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	var password string
	if strings.HasPrefix(key, "file:") {
		passFilePath := strings.TrimPrefix(key, "file:")
		passFilePath, _ = resolveDir(passFilePath)
		keyFile, keyFileErr := os.Open(passFilePath)
		if keyFileErr != nil {
			return "", errors.New("Password file failed to open")
		}
		passwordScanner := bufio.NewScanner(keyFile)
		for passwordScanner.Scan() {
			password = passwordScanner.Text()
		}
	} else if strings.HasPrefix(key, "env:") {
		passEnv := strings.TrimPrefix(key, "env:")
		passEnv = strings.TrimPrefix(passEnv, "${")
		passEnv = strings.TrimSuffix(passEnv, "}")
		passEnv = strings.TrimPrefix(passEnv, "$")
		password = os.Getenv(passEnv)
	} else {
		password = key
	}
	return password, nil
}
