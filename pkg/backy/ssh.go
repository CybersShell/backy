package backy

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
)

type SshConfig struct {
	// Config file to open
	configFile string

	// Private key path
	privateKey string

	// Port to connect to
	port uint16

	// host to check
	host string

	// host name to connect to
	hostName []string

	user string
}

func (config SshConfig) GetSSHConfig() (SshConfig, error) {
	hostNames := ssh_config.GetAll(config.host, "HostName")
	if hostNames == nil {
		return SshConfig{}, errors.New("hostname not found")
	}
	config.hostName = hostNames
	privKey, err := ssh_config.GetStrict(config.host, "IdentityFile")
	if err != nil {
		return SshConfig{}, err
	}
	config.privateKey = privKey
	User := ssh_config.Get(config.host, "User")
	if User == "" {
		return SshConfig{}, errors.New("user not found")
	}
	return config, nil
}

func (remoteConfig *Host) ConnectToSSHHost() (*ssh.Client, error) {

	var sshClient *ssh.Client
	var connectErr error

	f, _ := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "config"))
	cfg, _ := ssh_config.Decode(f)
	for _, host := range cfg.Hosts {
		if host.Matches(remoteConfig.Host) {
			var identityFile string
			if remoteConfig.PrivateKeyPath == "" {
				identityFile, _ = cfg.Get(remoteConfig.Host, "IdentityFile")
				usr, _ := user.Current()
				dir := usr.HomeDir
				if identityFile == "~" {
					// In case of "~", which won't be caught by the "else if"
					identityFile = dir
				} else if strings.HasPrefix(identityFile, "~/") {
					// Use strings.HasPrefix so we don't match paths like
					// "/something/~/something/"
					identityFile = filepath.Join(dir, identityFile[2:])
				}
				remoteConfig.PrivateKeyPath = filepath.Join(identityFile)
			}
			remoteConfig.HostName, _ = cfg.GetAll(remoteConfig.Host, "HostName")
			if remoteConfig.HostName == nil {
				remoteConfig.HostName[0] = remoteConfig.Host
				port, _ := cfg.Get(remoteConfig.Host, "Port")
				if port == "" {
					port = "22"
				}
				remoteConfig.HostName[0] = remoteConfig.HostName[0] + ":" + port
			} else {
				for index, hostName := range remoteConfig.HostName {
					port, _ := cfg.Get(remoteConfig.Host, "Port")
					if port == "" {
						port = "22"
					}
					remoteConfig.HostName[index] = hostName + ":" + port
				}
			}
			privateKey, err := os.ReadFile(remoteConfig.PrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("read private key error: %w", err)
			}
			signer, err := ssh.ParsePrivateKey(privateKey)
			if err != nil {
				return nil, fmt.Errorf("parse private key error: %w", err)
			}
			sshConfig := &ssh.ClientConfig{
				User:            remoteConfig.User,
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}
			for _, host := range remoteConfig.HostName {
				sshClient, connectErr = ssh.Dial("tcp", host, sshConfig)
				if connectErr != nil {
					panic(fmt.Errorf("error when connecting to host %s: %w", host, connectErr))
				}
			}
			break
		}

	}
	return sshClient, connectErr
}
