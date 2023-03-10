package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const versionStr = "0.2.4"

var (
	versionCmd = &cobra.Command{
		Use:   "version [flags]",
		Short: "Prints the version and exits.",
		Run:   version,
	}
	numOnly bool
)

func version(cmd *cobra.Command, args []string) {

	cmd.PersistentFlags().BoolVarP(&numOnly, "num", "n", true, "Output the version number only.")
	if numOnly {
		fmt.Printf("%s\n", versionStr)
	} else {
		fmt.Printf("Version: %s", versionStr)
	}

	os.Exit(0)
}
