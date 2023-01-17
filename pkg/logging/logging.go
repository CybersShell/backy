package logging

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
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
