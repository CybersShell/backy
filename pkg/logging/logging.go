package logging

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logging struct {
	Err    error
	Output string
}

type Logfile struct {
	LogfilePath string
}

func ExitWithMSG(msg string, code int, log *zerolog.Logger) {
	fmt.Printf("%s\n", msg)
	os.Exit(code)
}

func SetLoggingWriters(v *viper.Viper, logFile string) (writers zerolog.LevelWriter) {

	console := zerolog.ConsoleWriter{}
	if IsConsoleLoggingEnabled() {

		console = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC1123}
		console.FormatLevel = func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
		}
		console.FormatMessage = func(i any) string {
			if i == nil {
				return ""
			}
			return fmt.Sprintf("MSG: %s", i)
		}
		console.FormatFieldName = func(i interface{}) string {
			return fmt.Sprintf("%s: ", i)
		}
		console.FormatFieldValue = func(i interface{}) string {
			return fmt.Sprintf("%s", i)
			// return strings.ToUpper(fmt.Sprintf("%s", i))
		}
	}

	fileLogger := &lumberjack.Logger{
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}
	if strings.TrimSpace(logFile) != "" {
		fileLogger.Filename = logFile
	} else {
		fileLogger.Filename = "./backy.log"
	}

	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// zerolog.TimeFieldFormat = time.RFC1123
	writers = zerolog.MultiLevelWriter(fileLogger)

	if IsConsoleLoggingEnabled() {
		writers = zerolog.MultiLevelWriter(console, fileLogger)
	}
	return
}

func IsConsoleLoggingEnabled() bool {
	return os.Getenv("BACKY_CONSOLE_LOGGING") == "enabled"
}
