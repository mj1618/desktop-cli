# Feature: MCP Server Mode — Persistent Process with Cached State

## Priority: HIGH (eliminates per-call overhead, enables caching)

## Problem

Every `desktop-cli` invocation spawns a new process, initializes the platform provider, and exits. This costs ~50-100ms per call in process startup overhead alone. Over a 20-step agent workflow, that's 1-2 seconds of pure waste.

More importantly, each call re-reads the full accessibility tree from scratch. Consecutive calls like `read → click --id 5 → read` traverse the entire tree 3 times even though most of the tree hasn't changed. There's no way to cache state across calls because each process is independent.

AI agents (Claude, GPT, etc.) already support MCP (Model Context Protocol) natively, meaning they can call tools directly without shell overhead. Running desktop-cli as an MCP server would:
1. Eliminate process startup overhead (~50-100ms per call)
2. Enable accessibility tree caching across calls
3. Allow persistent element ID tracking
4. Provide a cleaner integration path than shell command execution

## What to Build

### 1. Command Definition — `cmd/serve.go` (new file)

```
desktop-cli serve [flags]

Flags:
  --transport <type>    Transport: stdio (default), sse, streamable-http
  --port <port>         Port for HTTP transports (default: 8080)
  --cache-ttl <ms>      Tree cache TTL in milliseconds (default: 500)
```

### 2. MCP Tools to Expose

Each existing CLI command becomes an MCP tool. The tool names and parameters map directly:

```json
{
  "tools": [
    {
      "name": "list",
      "description": "List windows and applications",
      "inputSchema": {
        "type": "object",
        "properties": {
          "windows": { "type": "boolean" },
          "apps": { "type": "boolean" },
          "app": { "type": "string" },
          "pid": { "type": "integer" }
        }
      }
    },
    {
      "name": "read",
      "description": "Read UI elements from a window's accessibility tree",
      "inputSchema": {
        "type": "object",
        "properties": {
          "app": { "type": "string" },
          "window": { "type": "string" },
          "depth": { "type": "integer" },
          "roles": { "type": "string" },
          "text": { "type": "string" },
          "flat": { "type": "boolean" },
          "prune": { "type": "boolean" },
          "format": { "type": "string", "enum": ["yaml", "json", "agent"] },
          "focused": { "type": "boolean" },
          "scope_id": { "type": "integer" },
          "children": { "type": "boolean" }
        }
      }
    },
    {
      "name": "click",
      "description": "Click an element by text, ID, or coordinates",
      "inputSchema": { ... }
    },
    {
      "name": "type",
      "description": "Type text or press key combinations",
      "inputSchema": { ... }
    },
    {
      "name": "action",
      "description": "Perform accessibility action on an element",
      "inputSchema": { ... }
    },
    {
      "name": "set_value",
      "description": "Set an element's value directly",
      "inputSchema": { ... }
    },
    {
      "name": "scroll",
      "description": "Scroll within a window or element",
      "inputSchema": { ... }
    },
    {
      "name": "wait",
      "description": "Wait for a UI condition to be met",
      "inputSchema": { ... }
    },
    {
      "name": "screenshot",
      "description": "Capture a screenshot",
      "inputSchema": { ... }
    },
    {
      "name": "do",
      "description": "Execute multiple actions in a batch",
      "inputSchema": { ... }
    }
  ]
}
```

### 3. Tree Caching Layer

The server maintains a per-app accessibility tree cache with configurable TTL:

```go
type TreeCache struct {
    mu      sync.RWMutex
    entries map[string]*CacheEntry  // key: "app:window" or "pid:windowID"
    ttl     time.Duration
}

type CacheEntry struct {
    elements  []model.Element
    timestamp time.Time
    readOpts  platform.ReadOptions
}
```

Cache behavior:
- **Read operations**: Return cached tree if TTL hasn't expired and read options match
- **Write operations** (click, type, action, set-value): Invalidate cache for the target app after the action completes
- **Wait/observe**: Bypass cache entirely (need fresh reads for polling)
- **Default TTL: 500ms**: Short enough to catch most UI changes, long enough to avoid redundant reads during rapid `read → act by id` sequences

### 4. Usage Examples

Agent configuration (e.g., in Claude Code MCP settings):
```json
{
  "mcpServers": {
    "desktop": {
      "command": "desktop-cli",
      "args": ["serve"]
    }
  }
}
```

Or as HTTP server for remote agents:
```bash
desktop-cli serve --transport streamable-http --port 8080
```

### 5. Implementation Approach

Use the `github.com/mark3labs/mcp-go` library (the standard Go MCP SDK) to implement the server:

```go
func runServe(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }

    cache := NewTreeCache(cacheTTL)
    server := mcpServer.NewMCPServer("desktop-cli", "1.0.0")

    // Register each command as an MCP tool
    server.AddTool(mcp.Tool{
        Name: "read",
        Description: "Read UI elements from a window",
        InputSchema: readSchema,
    }, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
        return handleRead(provider, cache, args)
    })

    // ... register other tools ...

    // Start transport
    transport, _ := cmd.Flags().GetString("transport")
    switch transport {
    case "stdio":
        return mcpServer.ServeStdio(server)
    case "sse":
        return mcpServer.ServeSSE(server, addr)
    case "streamable-http":
        return mcpServer.ServeStreamableHTTP(server, addr)
    }
}
```

## Files to Create

- `cmd/serve.go` — MCP server command
- `internal/server/cache.go` — Tree cache implementation
- `internal/server/handlers.go` — MCP tool handler functions (reuse extracted command logic)

## Files to Modify

- `go.mod` / `go.sum` — Add `github.com/mark3labs/mcp-go` dependency
- `README.md` — Add MCP server documentation
- `SKILL.md` — Add MCP configuration instructions

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli serve --help` shows usage and flags
- [ ] `desktop-cli serve` starts an MCP server on stdio transport
- [ ] All existing commands are exposed as MCP tools with correct schemas
- [ ] MCP tools return the same output as CLI commands
- [ ] Tree cache returns cached results within TTL
- [ ] Cache is invalidated after write actions (click, type, action, set-value)
- [ ] `--cache-ttl 0` disables caching
- [ ] HTTP transport works with `--transport streamable-http --port 8080`
- [ ] Server handles concurrent tool calls safely (mutex on cache, provider)
- [ ] Screenshot tool returns base64 image content in MCP response
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Depends on `do` command refactoring**: The MCP handlers should reuse the same extracted internal functions that the `do` command uses. Implement the `do` command first to drive the refactoring, then the MCP server becomes straightforward.
- **mcp-go library**: Use `github.com/mark3labs/mcp-go` — it's the standard Go MCP implementation. Handles protocol negotiation, tool registration, and transport abstraction.
- **Concurrency**: MCP servers can receive concurrent tool calls. The platform provider (especially CGo calls to macOS APIs) may not be thread-safe. Use a mutex around provider calls, or document that calls are serialized.
- **Screenshot handling**: MCP supports returning image content. Screenshots should return `mcp.ImageContent` with base64-encoded PNG data, not text.
- **Cache key design**: Use `fmt.Sprintf("%s:%s:%d:%d", app, window, pid, windowID)` as the cache key. Include the full `ReadOptions` hash to ensure cache hits only match identical queries.
- **Graceful shutdown**: Handle SIGINT/SIGTERM to close the server cleanly.
- **No `observe` tool**: The `observe` command streams JSONL, which doesn't map cleanly to MCP's request/response model. Skip it for now — agents can use `wait` or repeated `read` calls instead. Could add MCP notifications/resources later.
