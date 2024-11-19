package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const versionStr = "0.5.0"

var (
	versionCmd = &cobra.Command{
		Use:   "version [flags]",
		Short: "Prints the version and exits",
		Long:  "Prints the version and exits. No arguments just prints the version number only.",
		Run:   version,
	}
	numOnly bool
	vPre    bool
)

func init() {
	versionCmd.PersistentFlags().BoolVarP(&numOnly, "num", "n", false, "Output the version number only.")
	versionCmd.PersistentFlags().BoolVarP(&vPre, "vpre", "V", false, "Output the version with v prefixed.")
}

func version(cmd *cobra.Command, args []string) {

	if numOnly && !vPre {
		fmt.Printf("%s\n", versionStr)
	} else if vPre && !numOnly {
		fmt.Printf("v%s\n", versionStr)
	} else {
		fmt.Printf("Backy version: %s\n", versionStr)
	}

	os.Exit(0)
}
