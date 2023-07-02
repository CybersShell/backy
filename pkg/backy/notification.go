// notification.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0
package backy

import (
	"fmt"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/matrix"
	"maunium.net/go/mautrix/id"
)

type matrixStruct struct {
	homeserver  string
	roomid      id.RoomID
	accessToken string
	userId      id.UserID
}

type mailConfig struct {
	senderaddress string
	host          string
	to            []string
	username      string
	password      string
	port          string
}

func SetupCommandsNotifiers(backyConfig ConfigFile, ids ...string) {

}

// SetupNotify sets up notify instances for each command list.

func (backyConfig *ConfigFile) SetupNotify() {

	for _, cmdConfig := range backyConfig.CmdConfigLists {
		var services []notify.Notifier
		for notifyID := range backyConfig.Notifications {
			if contains(cmdConfig.Notifications, notifyID) {

				if backyConfig.Notifications[notifyID].Enabled {
					config := backyConfig.Notifications[notifyID].Config
					switch config.GetString("type") {
					case "matrix":
						mtrx := matrixStruct{
							userId:      id.UserID(config.GetString("user-id")),
							roomid:      id.RoomID(config.GetString("room-id")),
							accessToken: config.GetString("access-token"),
							homeserver:  config.GetString("homeserver"),
						}
						mtrxClient, _ := setupMatrix(mtrx)
						services = append(services, mtrxClient)
					case "mail":
						mailCfg := mailConfig{
							senderaddress: config.GetString("senderaddress"),
							password:      config.GetString("password"),
							username:      config.GetString("username"),
							to:            config.GetStringSlice("to"),
							host:          config.GetString("host"),
							port:          fmt.Sprint(config.GetUint16("port")),
						}
						mailClient := setupMail(mailCfg)
						services = append(services, mailClient)
					}
				}
			}
		}
		cmdConfig.NotifyConfig = notify.NewWithServices(services...)
	}

	// logging.ExitWithMSG("This was a test of notifications", 0, nil)
}

func setupMatrix(config matrixStruct) (*matrix.Matrix, error) {
	matrixClient, matrixErr := matrix.New(config.userId, config.roomid, config.homeserver, config.accessToken)
	if matrixErr != nil {
		panic(matrixErr)
	}
	return matrixClient, nil

}

func setupMail(config mailConfig) *mail.Mail {
	mailClient := mail.New(config.senderaddress, config.host+":"+config.port)
	mailClient.AuthenticateSMTP("", config.username, config.password, config.host)
	mailClient.AddReceivers(config.to...)
	mailClient.BodyFormat(mail.PlainText)
	return mailClient
}
