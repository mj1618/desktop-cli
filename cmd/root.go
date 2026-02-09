package cmd

import (
	"fmt"
	"os"

	"github.com/mj1618/desktop-cli/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "desktop-cli",
	Short: "Read and interact with desktop UI elements",
	Long:  "A CLI tool that lets AI agents read and interact with desktop UI elements via accessibility APIs.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version.Version, version.Commit, version.BuildDate)
}

// notImplemented returns a RunE function that outputs an error for stub commands.
func notImplemented(name string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("command not yet implemented: %s", name)
	}
}
