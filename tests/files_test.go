package tests

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestRunCommandFileTest(t *testing.T) {
	filePath := "packageCommands.yml"
	cmdLineStr := fmt.Sprintf("go run ../backy.go exec host -c checkDockerNoVersion -m localhost --cmdStdOut -f %s", filePath)

	cmd := exec.Command("bash", "-c", cmdLineStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v, Output: %s", err, string(output))
	}

	if len(output) == 0 {
		t.Fatal("Expected command output, got none")
	}
}
