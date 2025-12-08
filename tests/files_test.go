package tests

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestRunCommandFileTest(t *testing.T) {

	filePath := "packageCommands.yml"
	cmdLineStr := fmt.Sprintf("go run ../backy.go exec host -c checkDockerNoVersion -m localhost --cmdStdOut -f %s", filePath)

	exec.Command("bash", "-c", cmdLineStr).Output()
}
