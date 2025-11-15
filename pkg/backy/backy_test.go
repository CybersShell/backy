package backy

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"log"
// 	"testing"

// 	"git.andrewnw.xyz/CyberShell/backy/pkg/pkgman"
// 	packagemanagercommon "git.andrewnw.xyz/CyberShell/backy/pkg/pkgman/common"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"github.com/testcontainers/testcontainers-go"
// )

// // TestConfigOptions tests the configuration options for the backy package.
// func Test_ErrorHook(t *testing.T) {

// 	configFile := "../../tests/ErrorHook.yml"
// 	logFile := "ErrorHook.log"
// 	backyConfigOptions := NewConfigOptions(configFile, SetLogFile(logFile))
// 	backyConfigOptions.InitConfig()
// 	backyConfigOptions.ParseConfigurationFile()
// 	backyConfigOptions.RunListConfig("")

// }

// func TestSettingCommandInfoPackageCommandDnf(t *testing.T) {

// 	packagecommand := &Command{
// 		Type:             PackageCommandType,
// 		PackageManager:   "dnf",
// 		Shell:            "zsh",
// 		PackageOperation: PackageOperationCheckVersion,
// 		Packages:         []packagemanagercommon.Package{{Name: "docker-ce"}},
// 	}
// 	dnfPackage, _ := pkgman.PackageManagerFactory("dnf", pkgman.WithoutAuth())

// 	packagecommand.pkgMan = dnfPackage
// 	PackageCommand := getCommandTypeAndSetCommandInfo(packagecommand)

// 	assert.Equal(t, "dnf", PackageCommand.Cmd)

// }

// func TestWithDockerFile(t *testing.T) {
// 	ctx := context.Background()

// 	docker, err := testcontainers.Run(ctx, "",
// 		testcontainers.WithDockerfile(testcontainers.FromDockerfile{
// 			Context:    "../../tests/docker",
// 			Dockerfile: "Dockerfile",
// 			KeepImage:  false,
// 			// BuildOptionsModifier: func(buildOptions *types.ImageBuildOptions) {
// 			// 	buildOptions.Target = "target2"
// 			// },
// 		}),
// 	)
// 	// docker.

// 	if err != nil {
// 		log.Printf("failed to start container: %v", err)
// 		return
// 	}

// 	r, err := docker.Logs(ctx)
// 	if err != nil {
// 		log.Printf("failed to get logs: %v", err)
// 		return
// 	}

// 	logs, err := io.ReadAll(r)
// 	if err != nil {
// 		log.Printf("failed to read logs: %v", err)
// 		return
// 	}

// 	fmt.Println(string(logs))

// 	require.NoError(t, err)
// }
