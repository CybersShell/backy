package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const versionStr = "0.2.4"

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Prints the version and exits.",
		Run:   version,
	}
)

func version(cmd *cobra.Command, args []string) {

	fmt.Printf("v%s\n", versionStr)

	os.Exit(0)
}
