package backy

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gopkg.in/natefinch/lumberjack.v2"
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
	hostName string

	user string
}

func (config SshConfig) GetSSHConfig() (SshConfig, error) {
	hostNames := ssh_config.Get(config.host, "HostName")
	if hostNames == "" {
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
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s: ", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s", i))
	}

	fileLogger := &lumberjack.Logger{
		Filename:   "./backy.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}

	// fileOutput := zerolog.ConsoleWriter{Out: fileLogger, TimeFormat: time.RFC3339}
	// fileOutput.FormatLevel = func(i interface{}) string {
	// 	return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	// }
	// fileOutput.FormatMessage = func(i interface{}) string {
	// 	return fmt.Sprintf("%s", i)
	// }
	// fileOutput.FormatFieldName = func(i interface{}) string {
	// 	return fmt.Sprintf("%s: ", i)
	// }
	// fileOutput.FormatFieldValue = func(i interface{}) string {
	// 	return strings.ToUpper(fmt.Sprintf("%s", i))
	// }
	zerolog.TimeFieldFormat = time.RFC1123
	writers := zerolog.MultiLevelWriter(os.Stdout, fileLogger)
	log := zerolog.New(writers).With().Timestamp().Logger()

	var sshClient *ssh.Client
	var connectErr error

	khPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	f, _ := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "config"))
	cfg, _ := ssh_config.Decode(f)
	for _, host := range cfg.Hosts {
		// var hostKey ssh.PublicKey
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
				log.Debug().Str("Private key path", remoteConfig.PrivateKeyPath).Send()
			}
			remoteConfig.HostName, _ = cfg.Get(remoteConfig.Host, "HostName")
			remoteConfig.User, _ = cfg.Get(remoteConfig.Host, "User")
			if remoteConfig.HostName == "" {
				port, _ := cfg.Get(remoteConfig.Host, "Port")
				if port == "" {
					port = "22"
				}
				// remoteConfig.HostName[0] = remoteConfig.Host + ":" + port
			} else {
				// for index, hostName := range remoteConfig.HostName {
				port, _ := cfg.Get(remoteConfig.Host, "Port")
				if port == "" {
					port = "22"
				}
				remoteConfig.HostName = remoteConfig.HostName + ":" + port
				// remoteConfig.HostName[index] = hostName + ":" + port
			}

			// TODO: Add value/option to config for host key and add bool to check for host key
			hostKeyCallback, err := knownhosts.New(khPath)
			if err != nil {
				log.Fatal().Err(err).Msg("could not create hostkeycallback function")
			}
			privateKey, err := os.ReadFile(remoteConfig.PrivateKeyPath)
			if err != nil {
				log.Fatal().Err(err).Msg("read private key error")
			}
			signer, err := ssh.ParsePrivateKey(privateKey)
			if err != nil {
				log.Fatal().Err(err).Msg("parse private key error")
			}
			sshConfig := &ssh.ClientConfig{
				User:            remoteConfig.User,
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
				HostKeyCallback: hostKeyCallback,
				// HostKeyAlgorithms: []string{ssh.KeyAlgoECDSA256},
			}
			// for _, host := range remoteConfig.HostName {
			log.Info().Msgf("Connecting to host %s", remoteConfig.HostName)

			sshClient, connectErr = ssh.Dial("tcp", remoteConfig.HostName, sshConfig)
			if connectErr != nil {
				log.Fatal().Str("host", remoteConfig.HostName).Err(connectErr).Send()
			}
			// }
			break
		}

	}
	return sshClient, connectErr
}
