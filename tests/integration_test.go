package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestIntegration_ExecuteCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		expectFail bool
	}{
		{
			name:       "Version Command",
			args:       []string{"version"},
			expectFail: false,
		},
		{
			name:       "Invalid Command",
			args:       []string{"invalid"},
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", append([]string{"run", "../backy.go"}, tt.args...)...)
			output, err := cmd.CombinedOutput()

			if tt.expectFail && err == nil {
				t.Fatalf("Expected failure but got success. Output: %s", string(output))
			}

			if !tt.expectFail && err != nil {
				t.Fatalf("Expected success but got failure. Error: %v, Output: %s", err, string(output))
			}
		})
	}
}

func TestIntegration_ExecuteCommandWithConfig(t *testing.T) {
	configFile := "./SuccessHook.yml"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatalf("Config file not found: %s", configFile)
	}

	args := []string{"--config", configFile}
	hosts := []string{"localhost"}

	execListArgs := setupExecListRunOnHosts([]string{"echoTestSuccess"}, hosts, args)

	cmd := exec.Command("go", execListArgs...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Command execution failed. Error: %v, Output: %s", err, string(output))
	}

	if len(output) == 0 {
		t.Fatal("Expected command output, got none")
	}

	t.Logf("Command output:\n\n%s\n\n", string(output))
}

func setupExecListRunOnHosts(cmds, hosts, args []string) []string {
	baseArgs := []string{"run", "../backy.go", "exec", "host"}
	for _, h := range hosts {
		hostArg := "-m"
		hostArg += h
		baseArgs = append(baseArgs, hostArg)
	}
	for _, c := range cmds {
		cmdArg := "-c"
		cmdArg += c
		baseArgs = append(baseArgs, cmdArg)
	}
	return append(baseArgs, args...)
}
