package logging

import (
	"errors"
	"fmt"
	"log/syslog"
	"os"

	"github.com/spf13/viper"
)

type Logging struct {
	Err    error
	Output string
}

type Logfile struct {
	LogfilePath string
}

func OpenLogFile(config *viper.Viper) (interface{}, error) {
	var logFile *os.File
	var syslogWriter *syslog.Writer
	var err error
	logType := config.GetString("global.logging.type")
	if logType != "" {

		switch logType {
		case "file":
			logFile, err = os.OpenFile(config.GetString("global.logging.file"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err != nil {
				return nil, err
			}
			return logFile, nil
		case "syslog":
			syslogWriter, err = syslog.New(syslog.LOG_SYSLOG, "Backy")
			if err != nil {
				return nil, fmt.Errorf("Unable to set logfile: " + err.Error())
			}
			return syslogWriter, nil
		}
	}
	return nil, errors.New("log type not specified; Please set global.logging.type in your config file")
}
