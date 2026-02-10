package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an MCP server exposing desktop-cli tools",
	Long: `Start a Model Context Protocol (MCP) server that exposes all desktop-cli
commands as tools. AI agents can call tools directly without shell overhead.

Supported transports:
  stdio             Standard I/O (default, for Claude Code / MCP clients)
  streamable-http   Streamable HTTP transport (for remote agents)

Examples:
  desktop-cli serve
  desktop-cli serve --transport streamable-http --port 8080
  desktop-cli serve --cache-ttl 0`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("transport", "stdio", "Transport: stdio, streamable-http")
	serveCmd.Flags().Int("port", 8080, "HTTP port for streamable-http transport")
	serveCmd.Flags().Int("cache-ttl", 500, "Element tree cache TTL in milliseconds (0 to disable)")
}

func runServe(cmd *cobra.Command, args []string) error {
	transport, _ := cmd.Flags().GetString("transport")
	port, _ := cmd.Flags().GetInt("port")
	cacheTTLMs, _ := cmd.Flags().GetInt("cache-ttl")

	cfg := MCPConfig{
		Transport: transport,
		Port:      port,
		CacheTTL:  time.Duration(cacheTTLMs) * time.Millisecond,
	}

	srv, err := newMCPServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	return srv.serve(cfg)
}
