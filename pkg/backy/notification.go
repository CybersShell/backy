// notification.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0
package backy

import (
	"fmt"
	stdHttp "net/http"
	"strings"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/http"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/matrix"
	"maunium.net/go/mautrix/id"
)

type MatrixStruct struct {
	Homeserver  string    `yaml:"homeserver"`
	Roomid      id.RoomID `yaml:"room-id"`
	AccessToken string    `yaml:"access-token"`
	UserId      id.UserID `yaml:"user-id"`
}

type MailConfig struct {
	Host          string   `yaml:"host"`
	Port          string   `yaml:"port"`
	Username      string   `yaml:"username"`
	SenderAddress string   `yaml:"senderaddress"`
	To            []string `yaml:"to"`
	Password      string   `yaml:"password"`
}

type HttpConfig struct {
	Url     string              `yaml:"url"`
	Headers map[string][]string `yaml:"headers"`
}

// SetupNotify sets up notify instances for each command list.
func (opts *ConfigOpts) SetupNotify() {

	// check if we have individual commands instead of lists to execute
	if len(opts.executeCmds) != 0 {
		return
	}

	for confName, cmdConfig := range opts.CmdConfigLists {

		var services []notify.Notifier
		for _, id := range cmdConfig.Notifications {
			if !strings.Contains(id, ".") {
				opts.Logger.Info().Str("id", id).Str("list", cmdConfig.Name).Msg("key does not contain a \".\"  Make sure to follow the docs: https://backy.cybershell.xyz/config/notifications/")
				logging.ExitWithMSG(fmt.Sprintf("notification id %s in cmd list %s does not contain a \".\" \nMake sure to follow the docs: https://backy.cybershell.xyz/config/notifications/", id, cmdConfig.Name), 1, &opts.Logger)
			}

			confSplit := strings.Split(id, ".")
			confType := confSplit[0]
			confId := confSplit[1]
			switch confType {

			case "mail":
				conf, ok := opts.NotificationConf.MailConfig[confId]
				if !ok {
					opts.Logger.Info().Err(fmt.Errorf("error: ID %s not found in mail object", confId)).Str("list", confName).Send()
					continue
				}
				conf.Password = getExternalConfigDirectiveValue(conf.Password, opts)
				mailConf := setupMail(conf)
				services = append(services, mailConf)
			case "matrix":
				conf, ok := opts.NotificationConf.MatrixConfig[confId]
				if !ok {
					opts.Logger.Info().Err(fmt.Errorf("error: ID %s not found in matrix object", confId)).Str("list", confName).Send()
					continue
				}
				conf.AccessToken = getExternalConfigDirectiveValue(conf.AccessToken, opts)
				mtrxConf, mtrxErr := setupMatrix(conf)
				if mtrxErr != nil {
					opts.Logger.Info().Str("list", confName).Err(fmt.Errorf("error: configuring matrix id %s failed during setup: %w", id, mtrxErr))
					continue
				}
				services = append(services, mtrxConf)
			case "http":
				conf, ok := opts.NotificationConf.HttpConfig[confId]
				if !ok {
					opts.Logger.Info().Err(fmt.Errorf("error: ID %s not found in http object", confId)).Str("list", confName).Send()
					continue
				}
				httpConf := setupHttp(conf)
				services = append(services, httpConf)

			default:
				opts.Logger.Info().Err(fmt.Errorf("id %s not found", id)).Str("list", confName).Send()
			}
		}
		cmdConfig.NotifyConfig = notify.NewWithServices(services...)
	}

	// logging.ExitWithMSG("This was a test of notifications", 0, nil)
}

func setupMatrix(config MatrixStruct) (*matrix.Matrix, error) {
	matrixClient, matrixErr := matrix.New(config.UserId, config.Roomid, config.Homeserver, config.AccessToken)
	if matrixErr != nil {
		return nil, matrixErr
	}
	return matrixClient, nil
}

func setupMail(config MailConfig) *mail.Mail {
	mailClient := mail.New(config.SenderAddress, config.Host+":"+config.Port)
	mailClient.AuthenticateSMTP("", config.Username, config.Password, config.Host)
	mailClient.AddReceivers(config.To...)
	mailClient.BodyFormat(mail.PlainText)
	return mailClient
}

func setupHttp(httpConf HttpConfig) *http.Service {

	httpService := http.New()
	// httpService.AddReceiversURLs(httpConf.Url)
	httpService.AddReceivers(&http.Webhook{
		URL:         httpConf.Url,
		Header:      httpConf.Headers,
		ContentType: "text/plain",
		Method:      stdHttp.MethodPost,
		BuildPayload: func(subject, message string) (payload any) {
			return "[text/plain]: " + subject + " - " + message
		},
	})

	return httpService
}
