package cmd

// import (
// 	"bufio"
// 	"encoding/json"
// 	"os"
// 	"os/exec"
// 	"strings"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// // TestConfigOptions tests the configuration options for the backy package.
// func Test_ErrorHook(t *testing.T) {
// 	configFile := "-f ../../tests/ErrorHook.yml"
// 	logFile := "--log-file=ErrorHook.log"
// 	backyCommand := exec.Command("go", "run", "../../backy.go", configFile, logFile, "backup")
// 	backyCommand.Stderr = os.Stdout
// 	backyCommand.Stdout = os.Stdout
// 	err := backyCommand.Run()
// 	assert.Nil(t, err)
// 	os.Remove("ErrorHook.log")
// 	logFileData, logFileErr := os.ReadFile("ErrorHook.log")
// 	if logFileErr != nil {
// 		assert.FailNow(t, logFileErr.Error())

// 	}
// 	var JsonData []map[string]interface{}
// 	jsonScanner := bufio.NewScanner(strings.NewReader(string(logFileData)))

// 	for jsonScanner.Scan() {
// 		var jsonDataLine map[string]interface{}
// 		err = json.Unmarshal(jsonScanner.Bytes(), &jsonDataLine)
// 		assert.Nil(t, err)
// 		JsonData = append(JsonData, jsonDataLine)
// 	}
// 	for _, v := range JsonData {
// 		_, ok := v["error"]
// 		if !ok {
// 			assert.FailNow(t, "error does not exist\n")
// 			// return
// 		}
// 	}
// 	// t.Logf("%s", logFileData)
// 	// t.Logf("%v", JsonData)
// }

// func TestBackupErrorHook(t *testing.T) {
// 	logFile = "ErrorHook.log"

// 	configFile = "../tests/ErrorHook.yml"

// }
