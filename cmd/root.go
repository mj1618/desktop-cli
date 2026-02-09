package cmd

import (
	"fmt"
	"os"

	"github.com/mj1618/desktop-cli/internal/output"
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
	rootCmd.PersistentFlags().String("format", "yaml", "Output format: yaml, json")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Use the root persistent flag directly to avoid conflicts with
		// subcommand local flags (e.g. screenshot --format png/jpg).
		format, _ := rootCmd.PersistentFlags().GetString("format")
		switch format {
		case "yaml":
			output.OutputFormat = output.FormatYAML
		case "json":
			output.OutputFormat = output.FormatJSON
		default:
			return fmt.Errorf("unsupported format: %s (use yaml or json)", format)
		}
		if prettyFlag := cmd.Flags().Lookup("pretty"); prettyFlag != nil {
			if pretty, err := cmd.Flags().GetBool("pretty"); err == nil && pretty {
				output.PrettyOutput = true
			}
		}
		return nil
	}
}

// notImplemented returns a RunE function that outputs an error for stub commands.
func notImplemented(name string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("command not yet implemented: %s", name)
	}
}
