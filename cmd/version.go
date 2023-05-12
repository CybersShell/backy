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
	vPre    bool
)

func version(cmd *cobra.Command, args []string) {

	cmd.PersistentFlags().BoolVarP(&numOnly, "num", "n", true, "Output the version number only.")
	cmd.PersistentFlags().BoolVarP(&vPre, "vpre", "V", false, "Output the version with v prefixed.")

	if numOnly && !vPre {
		fmt.Printf("%s\n", versionStr)
	} else if vPre {
		fmt.Printf("v%s", versionStr)
	} else {
		fmt.Printf("Backy version: %s", versionStr)
	}

	os.Exit(0)
}
