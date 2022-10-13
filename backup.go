package main

import (
	"github.com/spf13/viper"
)

type commandBackup struct {
	cmd  string
	args []string
}

type directory struct {
	dst string
	src string
}

type backup struct {
	backupType         string
	local              bool
	commandToRunBefore commandBackup
	commandToRunAfter  commandBackup
	directories        directory
	name               string
}

func main() {

	viper.AddConfigPath(".")
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name

}
