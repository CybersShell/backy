package ssh

import (
	"errors"

	"github.com/kevinburke/ssh_config"
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
