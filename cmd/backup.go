package main

import (
	"git.andrewnw.xyz/CyberShell/backy/pkg/backy"
)

func main() {

	// config := backy.BackupConfig{
	// 	BackupType: "restic",
	// 	Name:       "mail-svr",
	// }
	// home, err := homedir.Dir()
	host := backy.Host{}

	host.Host = "email-svr"

}
