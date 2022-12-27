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
	PrivateKey string
	Port       uint
	HostName   string
	User       string
}

func GetSSHConfig(host string) (SshConfig, error) {
	var config SshConfig
	hostName := ssh_config.Get(host, "HostName")
	if hostName == "" {
		return SshConfig{}, errors.New("hostname not found")
	}
	config.HostName = hostName
	privKey, err := ssh_config.GetStrict(host, "IdentityFile")
	if err != nil {
		return SshConfig{}, err
	}
	config.PrivateKey = privKey
	User := ssh_config.Get(host, "User")
	if User == "" {
		return SshConfig{}, errors.New("user not found")
	}
	return config, nil
}

func (remoteConfig *Host) connectToSSHHost() (*ssh.Client, error) {
	var sshc *ssh.Client
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
			remoteConfig.HostName, _ = cfg.Get(remoteConfig.Host, "HostName")
			if remoteConfig.HostName == "" {
				remoteConfig.HostName = remoteConfig.Host
			}
			port, _ := cfg.Get(remoteConfig.Host, "Port")
			if port == "" {
				port = "22"
			}
			privateKey, err := os.ReadFile(remoteConfig.PrivateKeyPath)
			remoteConfig.HostName = remoteConfig.HostName + ":" + port
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
			sshc, connectErr = ssh.Dial("tcp", remoteConfig.HostName, sshConfig)
			break
		}

	}
	return sshc, connectErr
}
