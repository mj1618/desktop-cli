package server

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mj1618/desktop-cli/cmd"
	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/output"
	"github.com/mj1618/desktop-cli/internal/platform"
	"gopkg.in/yaml.v3"
)

// resultToText serializes a StepResult to YAML for MCP response.
func resultToText(result cmd.StepResult) string {
	b, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Sprintf("ok: %v\naction: %s\nerror: %s", result.OK, result.Action, result.Error)
	}
	return string(b)
}

// writeActionHandler wraps an executeX call: locks provider, executes, invalidates cache.
func (s *Server) writeActionHandler(
	request mcp.CallToolRequest,
	action string,
	fn func(*platform.Provider, map[string]interface{}, string, string) (cmd.StepResult, error),
) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	result, err := fn(s.provider, params, app, window)
	if err != nil {
		result.OK = false
		result.Error = err.Error()
		return mcp.NewToolResultError(resultToText(result)), nil
	}
	result.OK = true

	if app != "" {
		s.cache.InvalidateApp(app)
	} else {
		s.cache.InvalidateAll()
	}

	return mcp.NewToolResultText(resultToText(result)), nil
}

func (s *Server) handleList(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	apps := cmd.BoolParam(params, "apps", false)
	pid := cmd.IntParam(params, "pid", 0)
	appName := cmd.StringParam(params, "app", "")

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	if s.provider.Reader == nil {
		return mcp.NewToolResultError("reader not available on this platform"), nil
	}

	windows, err := s.provider.Reader.ListWindows(platform.ListOptions{
		Apps: apps,
		PID:  pid,
		App:  appName,
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if apps {
		seen := make(map[string]bool)
		type appEntry struct {
			App string `yaml:"app" json:"app"`
			PID int    `yaml:"pid" json:"pid"`
		}
		var entries []appEntry
		for _, w := range windows {
			if !seen[w.App] {
				seen[w.App] = true
				entries = append(entries, appEntry{App: w.App, PID: w.PID})
			}
		}
		b, _ := yaml.Marshal(entries)
		return mcp.NewToolResultText(string(b)), nil
	}

	b, _ := yaml.Marshal(windows)
	return mcp.NewToolResultText(string(b)), nil
}

func (s *Server) handleRead(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")
	windowID := cmd.IntParam(params, "window-id", 0)
	pid := cmd.IntParam(params, "pid", 0)
	depth := cmd.IntParam(params, "depth", 0)
	scopeID := cmd.IntParam(params, "scope-id", 0)
	text := cmd.StringParam(params, "text", "")
	focused := cmd.BoolParam(params, "focused", false)

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	if s.provider.Reader == nil {
		return mcp.NewToolResultError("reader not available on this platform"), nil
	}

	opts := platform.ReadOptions{
		App:      app,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
		Depth:    depth,
	}

	elements, err := s.cache.ReadElements(s.provider.Reader, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Auto-prune web content
	hasWeb := model.HasWebContent(elements)
	if hasWeb {
		elements = model.PruneEmptyGroups(elements)
	}

	// Scope to descendants
	if scopeID > 0 {
		scopeEl := findElementByID(elements, scopeID)
		if scopeEl == nil {
			return mcp.NewToolResultError(fmt.Sprintf("scope element with id %d not found", scopeID)), nil
		}
		elements = scopeEl.Children
	}

	// Text filter
	if text != "" {
		elements = model.FilterByText(elements, text)
	}

	// Focused filter
	if focused {
		elements = model.FilterByFocused(elements)
	}

	// Resolve window title
	windowTitle := window
	if windowTitle == "" {
		for _, el := range elements {
			if el.Role == "window" && el.Title != "" {
				windowTitle = el.Title
				break
			}
		}
	}

	agentStr := output.FormatAgentString(app, pid, windowTitle, elements)
	return mcp.NewToolResultText(agentStr), nil
}

func (s *Server) handleClick(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "click", cmd.ExecuteClick)
}

func (s *Server) handleType(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "type", cmd.ExecuteType)
}

func (s *Server) handleAction(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "action", cmd.ExecuteAction)
}

func (s *Server) handleSetValue(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "set_value", cmd.ExecuteSetValue)
}

func (s *Server) handleScroll(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "scroll", cmd.ExecuteScroll)
}

func (s *Server) handleHover(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "hover", cmd.ExecuteHover)
}

func (s *Server) handleFocus(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.writeActionHandler(request, "focus", cmd.ExecuteFocus)
}

func (s *Server) handleFill(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	// ExecuteStep handles fill dispatch
	result, err := cmd.ExecuteStep(s.provider, "fill", params, app, window)
	if err != nil {
		result.OK = false
		result.Error = err.Error()
		return mcp.NewToolResultError(resultToText(result)), nil
	}
	result.OK = true

	if app != "" {
		s.cache.InvalidateApp(app)
	}

	return mcp.NewToolResultText(resultToText(result)), nil
}

func (s *Server) handleWait(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	result, err := cmd.ExecuteWait(s.provider, params, app, window)
	if err != nil {
		result.OK = false
		result.Error = err.Error()
		return mcp.NewToolResultError(resultToText(result)), nil
	}
	result.OK = true
	return mcp.NewToolResultText(resultToText(result)), nil
}

func (s *Server) handleAssert(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	result, err := cmd.ExecuteAssert(s.provider, params, app, window)
	if err != nil {
		result.OK = false
		result.Error = err.Error()
		return mcp.NewToolResultError(resultToText(result)), nil
	}
	result.OK = true
	return mcp.NewToolResultText(resultToText(result)), nil
}

func (s *Server) handleOpen(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()

	result, err := cmd.ExecuteOpen(params)
	if err != nil {
		result.OK = false
		result.Error = err.Error()
		return mcp.NewToolResultError(resultToText(result)), nil
	}
	result.OK = true
	return mcp.NewToolResultText(resultToText(result)), nil
}

func (s *Server) handleScreenshot(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")
	windowID := cmd.IntParam(params, "window-id", 0)
	pid := cmd.IntParam(params, "pid", 0)
	format := cmd.StringParam(params, "format", "png")
	quality := cmd.IntParam(params, "quality", 80)
	scale := 0.5
	if v, ok := params["scale"]; ok {
		if f, ok := v.(float64); ok {
			scale = f
		}
	}

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	if s.provider.Screenshotter == nil {
		return mcp.NewToolResultError("screenshot not supported on this platform"), nil
	}

	data, err := s.provider.Screenshotter.CaptureWindow(platform.ScreenshotOptions{
		App:      app,
		Window:   window,
		WindowID: windowID,
		PID:      pid,
		Format:   format,
		Quality:  quality,
		Scale:    scale,
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	mimeType := "image/png"
	if format == "jpg" || format == "jpeg" {
		mimeType = "image/jpeg"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.ImageContent{
				Type:     "image",
				Data:     b64,
				MIMEType: mimeType,
			},
		},
	}, nil
}

func (s *Server) handleDo(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := request.GetArguments()
	app := cmd.StringParam(params, "app", "")
	window := cmd.StringParam(params, "window", "")
	stopOnError := cmd.BoolParam(params, "stop-on-error", true)

	stepsRaw, ok := params["steps"]
	if !ok {
		return mcp.NewToolResultError("steps parameter is required"), nil
	}

	arr, ok := stepsRaw.([]interface{})
	if !ok {
		return mcp.NewToolResultError("steps must be an array"), nil
	}

	steps := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("each step must be an object"), nil
		}
		steps = append(steps, m)
	}

	s.providerMu.Lock()
	defer s.providerMu.Unlock()

	doCtx := &cmd.DoContext{
		Provider:      s.provider,
		DefaultApp:    app,
		DefaultWindow: window,
		StopOnError:   stopOnError,
	}
	doCtx.ExecuteSteps(steps, 0)

	s.cache.InvalidateAll()

	doResult := cmd.DoResult{
		OK:      !doCtx.HasFailure,
		Action:  "do",
		Steps:   len(steps),
		Results: doCtx.Results,
	}
	completed := 0
	for _, r := range doCtx.Results {
		if r.OK {
			completed++
		}
	}
	doResult.Completed = completed

	b, _ := yaml.Marshal(doResult)
	return mcp.NewToolResultText(string(b)), nil
}

// findElementByID searches the element tree recursively for an element with the given ID.
func findElementByID(elements []model.Element, id int) *model.Element {
	for i := range elements {
		if elements[i].ID == id {
			return &elements[i]
		}
		if found := findElementByID(elements[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}
