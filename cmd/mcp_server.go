package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mj1618/desktop-cli/internal/platform"
)

// mcpServer wraps the MCP server with the platform provider and cache.
type mcpServer struct {
	provider   *platform.Provider
	cache      *mcpTreeCache
	providerMu sync.Mutex
	mcp        *mcpserver.MCPServer
}

// MCPConfig holds MCP server configuration.
type MCPConfig struct {
	Transport string
	Port      int
	CacheTTL  time.Duration
}

// newMCPServer creates and configures an MCP server with all desktop-cli tools.
func newMCPServer(cfg MCPConfig) (*mcpServer, error) {
	provider, err := platform.NewProvider()
	if err != nil {
		return nil, err
	}

	s := &mcpServer{
		provider: provider,
		cache:    newMCPTreeCache(cfg.CacheTTL),
	}

	s.mcp = mcpserver.NewMCPServer(
		"desktop-cli",
		"1.0.0",
	)

	s.registerTools()
	return s, nil
}

// serve starts the MCP server with the configured transport.
func (s *mcpServer) serve(cfg MCPConfig) error {
	switch cfg.Transport {
	case "stdio":
		return mcpserver.ServeStdio(s.mcp)
	case "streamable-http":
		httpServer := mcpserver.NewStreamableHTTPServer(s.mcp)
		return httpServer.Start(fmt.Sprintf(":%d", cfg.Port))
	default:
		return fmt.Errorf("unsupported transport: %s (use stdio or streamable-http)", cfg.Transport)
	}
}

func (s *mcpServer) registerTools() {
	// list
	s.mcp.AddTool(
		mcp.NewTool("list",
			mcp.WithDescription("List available windows and applications on the desktop"),
			mcp.WithBoolean("apps", mcp.Description("List running applications instead of windows")),
			mcp.WithString("app", mcp.Description("Filter by application name")),
			mcp.WithNumber("pid", mcp.Description("Filter by process ID")),
		),
		s.handleList,
	)

	// read
	s.mcp.AddTool(
		mcp.NewTool("read",
			mcp.WithDescription("Read the UI element tree from an application's accessibility layer. Returns interactive elements with IDs, roles, titles, bounds, and actions."),
			mcp.WithString("app", mcp.Description("Application name (e.g. 'Safari', 'Chrome')")),
			mcp.WithString("window", mcp.Description("Filter by window title substring")),
			mcp.WithNumber("window-id", mcp.Description("Filter by system window ID")),
			mcp.WithNumber("pid", mcp.Description("Filter by process ID")),
			mcp.WithNumber("depth", mcp.Description("Max depth to traverse (0 = unlimited)")),
			mcp.WithString("text", mcp.Description("Filter elements by text content")),
			mcp.WithBoolean("focused", mcp.Description("Only return the currently focused element")),
			mcp.WithNumber("scope-id", mcp.Description("Limit to descendants of this element ID")),
			mcp.WithNumber("max-elements", mcp.Description("Max elements in output (0 = unlimited)")),
		),
		s.handleRead,
	)

	// click
	s.mcp.AddTool(
		mcp.NewTool("click",
			mcp.WithDescription("Click on a UI element by text, ID, or screen coordinates"),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("text", mcp.Description("Find and click element by text")),
			mcp.WithNumber("id", mcp.Description("Click element by ID")),
			mcp.WithNumber("x", mcp.Description("Click at X coordinate")),
			mcp.WithNumber("y", mcp.Description("Click at Y coordinate")),
			mcp.WithString("button", mcp.Description("Mouse button: left, right, middle")),
			mcp.WithBoolean("double", mcp.Description("Double-click")),
			mcp.WithString("roles", mcp.Description("Filter by role when using text")),
			mcp.WithBoolean("exact", mcp.Description("Require exact text match")),
			mcp.WithNumber("scope-id", mcp.Description("Limit text search to descendants of element ID")),
			mcp.WithBoolean("near", mcp.Description("Click nearest interactive element to text match")),
			mcp.WithString("near-direction", mcp.Description("Direction for near: left, right, above, below")),
		),
		s.handleClick,
	)

	// type
	s.mcp.AddTool(
		mcp.NewTool("type",
			mcp.WithDescription("Type text or press key combinations. Optionally target a specific element first."),
			mcp.WithString("text", mcp.Description("Text to type")),
			mcp.WithString("key", mcp.Description("Key combo (e.g. 'cmd+c', 'enter', 'tab')")),
			mcp.WithString("target", mcp.Description("Find element by text and focus before typing")),
			mcp.WithNumber("id", mcp.Description("Focus element by ID first")),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("roles", mcp.Description("Filter by role when using text")),
			mcp.WithBoolean("exact", mcp.Description("Require exact text match")),
			mcp.WithNumber("scope-id", mcp.Description("Limit text search")),
			mcp.WithNumber("delay", mcp.Description("Delay between keystrokes in ms")),
		),
		s.handleType,
	)

	// action
	s.mcp.AddTool(
		mcp.NewTool("action",
			mcp.WithDescription("Perform an accessibility action (press, cancel, pick, increment, decrement, confirm, showMenu) on a UI element"),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithNumber("id", mcp.Description("Element ID")),
			mcp.WithString("text", mcp.Description("Find element by text")),
			mcp.WithString("action", mcp.Description("Action to perform (default: press)")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("roles", mcp.Description("Filter by role")),
			mcp.WithBoolean("exact", mcp.Description("Require exact text match")),
			mcp.WithNumber("scope-id", mcp.Description("Limit text search")),
		),
		s.handleAction,
	)

	// set_value
	s.mcp.AddTool(
		mcp.NewTool("set_value",
			mcp.WithDescription("Set the value of a UI element directly via accessibility API"),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithNumber("id", mcp.Description("Element ID")),
			mcp.WithString("text", mcp.Description("Find element by text")),
			mcp.WithString("value", mcp.Description("Value to set")),
			mcp.WithString("attribute", mcp.Description("Attribute to set (default: value)")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("roles", mcp.Description("Filter by role")),
			mcp.WithBoolean("exact", mcp.Description("Require exact text match")),
			mcp.WithNumber("scope-id", mcp.Description("Limit text search")),
		),
		s.handleSetValue,
	)

	// scroll
	s.mcp.AddTool(
		mcp.NewTool("scroll",
			mcp.WithDescription("Scroll within a window or element"),
			mcp.WithString("direction", mcp.Description("Scroll direction: up, down, left, right"), mcp.Required()),
			mcp.WithNumber("amount", mcp.Description("Scroll clicks (default: 3)")),
			mcp.WithNumber("x", mcp.Description("Scroll at X coordinate")),
			mcp.WithNumber("y", mcp.Description("Scroll at Y coordinate")),
			mcp.WithNumber("id", mcp.Description("Scroll within element by ID")),
			mcp.WithString("text", mcp.Description("Find element by text and scroll within")),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithString("window", mcp.Description("Scope to window")),
		),
		s.handleScroll,
	)

	// wait
	s.mcp.AddTool(
		mcp.NewTool("wait",
			mcp.WithDescription("Wait for a UI condition to be met (element appearing/disappearing)"),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("for-text", mcp.Description("Wait for element with this text")),
			mcp.WithString("for-role", mcp.Description("Wait for element with this role")),
			mcp.WithNumber("for-id", mcp.Description("Wait for element with this ID")),
			mcp.WithBoolean("gone", mcp.Description("Wait until condition is NO LONGER true")),
			mcp.WithNumber("timeout", mcp.Description("Max seconds to wait (default: 30)")),
			mcp.WithNumber("interval", mcp.Description("Polling interval in ms (default: 500)")),
			mcp.WithNumber("pid", mcp.Description("Filter by process ID")),
			mcp.WithNumber("window-id", mcp.Description("Filter by window ID")),
		),
		s.handleWait,
	)

	// screenshot
	s.mcp.AddTool(
		mcp.NewTool("screenshot",
			mcp.WithDescription("Capture a screenshot of a window or the entire screen"),
			mcp.WithString("app", mcp.Description("Capture app's frontmost window")),
			mcp.WithString("window", mcp.Description("Capture window by title")),
			mcp.WithNumber("window-id", mcp.Description("Capture window by system ID")),
			mcp.WithNumber("pid", mcp.Description("Capture frontmost window of PID")),
			mcp.WithString("format", mcp.Description("Image format: png, jpg (default: png)")),
			mcp.WithNumber("quality", mcp.Description("JPEG quality 1-100 (default: 80)")),
			mcp.WithNumber("scale", mcp.Description("Scale factor 0.1-1.0 (default: 0.5)")),
		),
		s.handleScreenshot,
	)

	// hover
	s.mcp.AddTool(
		mcp.NewTool("hover",
			mcp.WithDescription("Move the mouse to hover over a UI element (trigger tooltips, etc.)"),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("text", mcp.Description("Find element by text")),
			mcp.WithNumber("id", mcp.Description("Element ID")),
			mcp.WithNumber("x", mcp.Description("Hover at X coordinate")),
			mcp.WithNumber("y", mcp.Description("Hover at Y coordinate")),
			mcp.WithString("roles", mcp.Description("Filter by role")),
			mcp.WithBoolean("exact", mcp.Description("Require exact text match")),
			mcp.WithNumber("scope-id", mcp.Description("Limit text search")),
		),
		s.handleHover,
	)

	// focus
	s.mcp.AddTool(
		mcp.NewTool("focus",
			mcp.WithDescription("Bring a window or application to the foreground"),
			mcp.WithString("app", mcp.Description("Focus application by name")),
			mcp.WithString("window", mcp.Description("Focus window by title")),
			mcp.WithNumber("window-id", mcp.Description("Focus window by system ID")),
			mcp.WithNumber("pid", mcp.Description("Focus application by PID")),
			mcp.WithBoolean("new-document", mcp.Description("After focusing, dismiss dialog and create new document")),
		),
		s.handleFocus,
	)

	// fill
	s.mcp.AddTool(
		mcp.NewTool("fill",
			mcp.WithDescription("Set multiple form fields in one call, reading the UI tree only once"),
			mcp.WithString("app", mcp.Description("Target application"), mcp.Required()),
			mcp.WithString("window", mcp.Description("Target window")),
			mcp.WithNumber("window-id", mcp.Description("Target by window ID")),
			mcp.WithNumber("pid", mcp.Description("Target by PID")),
			mcp.WithArray("fields", mcp.Description("Array of {label, id, value, method} objects"), mcp.Required()),
			mcp.WithString("submit", mcp.Description("After filling, click element with this text")),
			mcp.WithBoolean("tab-between", mcp.Description("Use Tab to move between fields")),
			mcp.WithString("method", mcp.Description("How to set values: 'set-value' or 'type'")),
		),
		s.handleFill,
	)

	// assert
	s.mcp.AddTool(
		mcp.NewTool("assert",
			mcp.WithDescription("Assert a UI element's state (existence, value, checked, focused, etc.)"),
			mcp.WithString("app", mcp.Description("Scope to application")),
			mcp.WithString("window", mcp.Description("Scope to window")),
			mcp.WithString("text", mcp.Description("Find element by text")),
			mcp.WithNumber("id", mcp.Description("Element ID")),
			mcp.WithString("roles", mcp.Description("Filter by role")),
			mcp.WithBoolean("exact", mcp.Description("Require exact text match")),
			mcp.WithNumber("scope-id", mcp.Description("Limit text search")),
			mcp.WithString("value", mcp.Description("Assert element has this value")),
			mcp.WithString("value-contains", mcp.Description("Assert value contains substring")),
			mcp.WithBoolean("checked", mcp.Description("Assert element is checked")),
			mcp.WithBoolean("unchecked", mcp.Description("Assert element is unchecked")),
			mcp.WithBoolean("disabled", mcp.Description("Assert element is disabled")),
			mcp.WithBoolean("enabled", mcp.Description("Assert element is enabled")),
			mcp.WithBoolean("focused", mcp.Description("Assert element is focused")),
			mcp.WithBoolean("gone", mcp.Description("Assert element does NOT exist")),
			mcp.WithNumber("timeout", mcp.Description("Retry for N seconds")),
			mcp.WithNumber("interval", mcp.Description("Retry interval in ms")),
		),
		s.handleAssert,
	)

	// open
	s.mcp.AddTool(
		mcp.NewTool("open",
			mcp.WithDescription("Open a URL, file, or application"),
			mcp.WithString("url", mcp.Description("URL to open")),
			mcp.WithString("file", mcp.Description("File path to open")),
			mcp.WithString("app", mcp.Description("Application to open or open with")),
		),
		s.handleOpen,
	)

	// do (batch)
	s.mcp.AddTool(
		mcp.NewTool("do",
			mcp.WithDescription("Execute multiple actions in a batch. Steps execute sequentially. Supports: click, type, action, set-value, scroll, hover, focus, wait, assert, fill, read, open, sleep, if-exists, if-focused, try"),
			mcp.WithString("app", mcp.Description("Default app for all steps")),
			mcp.WithString("window", mcp.Description("Default window for all steps")),
			mcp.WithArray("steps", mcp.Description("Array of step objects"), mcp.Required()),
			mcp.WithBoolean("stop-on-error", mcp.Description("Stop on first error (default: true)")),
		),
		s.handleDo,
	)
}
