// notification.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0
package notification

import (
	"fmt"

	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
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

var services []notify.Notifier

func SetupCommandsNotifiers(backyConfig backy.BackyConfigFile, ids ...string) {

}

// SetupNotify sets up notify instances for each command list.

func SetupNotify(backyConfig backy.BackyConfigFile) {

	for _, cmdConfig := range backyConfig.CmdConfigLists {
		for notifyID, notifConfig := range cmdConfig.NotificationsConfig {
			if cmdConfig.NotificationsConfig[notifyID].Enabled {
				config := notifConfig.Config
				switch notifConfig.Config.GetString("type") {
				case "matrix":
					// println(config.GetString("access-token"))
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
	backyNotify := notify.New()

	backyNotify.UseServices(services...)

	// err := backyNotify.Send(
	// 	context.Background(),
	// 	"Subject/Title",
	// 	"The actual message - Hello, you awesome gophers! :)",
	// )
	// if err != nil {
	// 	panic(err)
	// }
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
	return mailClient
}
