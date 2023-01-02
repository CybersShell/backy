package backy

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

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
		var hostKey ssh.PublicKey
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
				port, _ := cfg.Get(remoteConfig.Host, "Port")
				if port == "" {
					port = "22"
				}
				remoteConfig.HostName[0] = remoteConfig.Host + ":" + port
			} else {
				for index, hostName := range remoteConfig.HostName {
					port, _ := cfg.Get(remoteConfig.Host, "Port")
					if port == "" {
						port = "22"
					}
					remoteConfig.HostName[index] = hostName + ":" + port
					hostKey = getHostKey(hostName)
					println("HostName: " + remoteConfig.HostName[0])
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
				HostKeyCallback: ssh.FixedHostKey(hostKey),
				Timeout:         5 * time.Second,
			}
			for _, host := range remoteConfig.HostName {
				println("Connecting to " + host)
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

func getHostKey(host string) ssh.PublicKey {
	// parse OpenSSH known_hosts file
	// ssh or use ssh-keyscan to get initial key
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], base64.StdEncoding.EncodeToString([]byte(host))) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				log.Fatalf("error parsing %q: %v", fields[2], err)
			}
			break
		}
	}

	if hostKey == nil {
		log.Fatalf("no hostkey found for %s", host)
	}

	return hostKey
}
