package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"github.com/spf13/cobra"
)

// OpenResult is the YAML output of a successful open.
type OpenResult struct {
	OK     bool   `yaml:"ok"               json:"ok"`
	Action string `yaml:"action"           json:"action"`
	URL    string `yaml:"url,omitempty"    json:"url,omitempty"`
	File   string `yaml:"file,omitempty"   json:"file,omitempty"`
	App    string `yaml:"app,omitempty"    json:"app,omitempty"`
	PID    int    `yaml:"pid,omitempty"    json:"pid,omitempty"`
	State  string `yaml:"state,omitempty"  json:"state,omitempty"`
}

var openCmd = &cobra.Command{
	Use:   "open [url or file path]",
	Short: "Open a URL, file, or application",
	Long: `Open a URL in the default browser, open a file with its default app, or launch an application.

Uses the macOS 'open' command under the hood. Eliminates the multi-step workflow of
focusing a browser, clicking the address bar, selecting text, typing a URL, and pressing enter.`,
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().String("url", "", "Open a URL in the default browser (or --app browser)")
	openCmd.Flags().String("file", "", "Open a file with its default application (or --app)")
	openCmd.Flags().String("app", "", "Use a specific application to open the URL/file, or launch the app by itself")
	openCmd.Flags().Bool("wait", false, "Wait for the application window to appear after opening")
	openCmd.Flags().Int("timeout", 10, "Max seconds to wait for the window (used with --wait)")
	addPostReadFlags(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	urlStr, _ := cmd.Flags().GetString("url")
	fileStr, _ := cmd.Flags().GetString("file")
	appName, _ := cmd.Flags().GetString("app")
	waitForWindow, _ := cmd.Flags().GetBool("wait")
	timeoutSec, _ := cmd.Flags().GetInt("timeout")
	prOpts := getPostReadOptions(cmd)

	// Positional arg: auto-detect URL vs file
	if len(args) > 0 && urlStr == "" && fileStr == "" {
		arg := args[0]
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			urlStr = arg
		} else {
			fileStr = arg
		}
	}

	if urlStr == "" && fileStr == "" && appName == "" {
		return fmt.Errorf("specify a URL, file, or --app to open")
	}

	// Build the macOS `open` command arguments
	var openArgs []string

	if appName != "" {
		openArgs = append(openArgs, "-a", appName)
	}

	if urlStr != "" {
		openArgs = append(openArgs, urlStr)
	} else if fileStr != "" {
		openArgs = append(openArgs, fileStr)
	}

	// Execute the open command
	openExec := exec.Command("open", openArgs...)
	if out, err := openExec.CombinedOutput(); err != nil {
		return fmt.Errorf("open failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	// Determine the app name for post-read and wait
	// If --app was specified, use that. Otherwise infer from what was opened.
	resolvedApp := appName

	// Wait for the application window to appear
	if waitForWindow && resolvedApp != "" {
		provider, err := platform.NewProvider()
		if err != nil {
			return err
		}
		if provider.Reader != nil {
			timeout := time.Duration(timeoutSec) * time.Second
			deadline := time.Now().Add(timeout)
			interval := 500 * time.Millisecond

			for time.Now().Before(deadline) {
				elements, err := provider.Reader.ReadElements(platform.ReadOptions{
					App: resolvedApp,
				})
				if err == nil && len(elements) > 0 {
					break
				}
				time.Sleep(interval)
			}
		}
	}

	// Post-read: include full UI state in agent format
	var state string
	if prOpts.PostRead && resolvedApp != "" {
		provider, err := platform.NewProvider()
		if err == nil {
			state = readPostActionState(provider, resolvedApp, "", 0, 0, prOpts.Delay, prOpts.MaxElements)
		}
	}

	result := OpenResult{
		OK:     true,
		Action: "open",
		App:    appName,
		State:  state,
	}
	if urlStr != "" {
		result.URL = urlStr
	}
	if fileStr != "" {
		result.File = fileStr
	}

	return output.Print(result)
}
