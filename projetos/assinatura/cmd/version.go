package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version é substituída em build-time via -ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Exibe a versão atual do CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Assinatura CLI %s %s/%s\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
