package widget

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"strings"

	mcpgw "github.com/memohai/memoh/internal/mcp"
)

//go:embed specs/*.md
var specsFS embed.FS

const (
	toolShowWidget = "show_widget"
	toolReadMe     = "read_me"
)

var validModules = map[string]string{
	"chart":       "specs/chart.md",
	"interactive": "specs/interactive.md",
	"diagram":     "specs/diagram.md",
	"mockup":      "specs/mockup.md",
	"art":         "specs/art.md",
}

// Executor provides generative UI tools for rendering interactive widgets in chat.
type Executor struct {
	logger *slog.Logger
}

// NewExecutor returns a widget tool executor.
func NewExecutor(log *slog.Logger) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		logger: log.With(slog.String("provider", "widget_tool")),
	}
}

// ListTools returns the show_widget and read_me tool descriptors.
func (*Executor) ListTools(_ context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if session.IsSubagent {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return []mcpgw.ToolDescriptor{
		{
			Name: toolShowWidget,
			Description: `Render an interactive HTML widget directly in the chat UI. The HTML is displayed inside a Shadow DOM with theme-aware CSS variables. Use this for charts (Chart.js), interactive cards, diagrams (SVG), mockups, and data visualizations.

IMPORTANT RULES:
- Output a SINGLE self-contained HTML fragment: <style>...</style> then HTML then <script>...</script>
- For interactivity, use onclick="sendPrompt('...')" — this is the ONLY way to trigger follow-up actions
- External scripts: only cdn.jsdelivr.net is allowed (Chart.js, D3.js)
- NO iframes, NO external stylesheets, NO dynamic code generation
- Before your first use, call read_me to load the design spec for the module you need`,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"html": map[string]any{
						"type":        "string",
						"description": "Complete HTML fragment with inline <style>, HTML body, and optional <script>. Must be self-contained.",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Optional title displayed above the widget for collapse/expand.",
					},
					"height": map[string]any{
						"type":        "number",
						"description": "Suggested height in pixels. If omitted, the widget auto-sizes.",
					},
				},
				"required": []string{"html"},
			},
		},
		{
			Name: toolReadMe,
			Description: `Load design specification for a widget module. Call this BEFORE using show_widget to understand the design rules for the type of widget you want to create.

Available modules:
- chart: Chart.js configuration rules, color palettes, responsive sizing
- interactive: Card, button, metric, form components
- diagram: SVG flowchart, tree, and graph rules
- mockup: UI prototype guidelines and layout patterns
- art: Illustration and decorative element rules`,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"module": map[string]any{
						"type":        "string",
						"description": "The design spec module to load.",
						"enum":        []string{"chart", "interactive", "diagram", "mockup", "art"},
					},
				},
				"required": []string{"module"},
			},
		},
	}, nil
}

// CallTool handles show_widget and read_me tool calls.
func (e *Executor) CallTool(_ context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	switch toolName {
	case toolShowWidget:
		return e.callShowWidget(session, arguments)
	case toolReadMe:
		return e.callReadMe(arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

func (e *Executor) callShowWidget(session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	if session.IsSubagent {
		return mcpgw.BuildToolErrorResult("show_widget is not available in subagent context"), nil
	}

	html := mcpgw.StringArg(arguments, "html")
	if html == "" {
		return mcpgw.BuildToolErrorResult("html is required"), nil
	}

	title := mcpgw.StringArg(arguments, "title")
	height, _, _ := mcpgw.IntArg(arguments, "height")

	result := map[string]any{
		"ok":   true,
		"html": html,
	}
	if title != "" {
		result["title"] = title
	}
	if height > 0 {
		result["height"] = height
	}

	e.logger.Debug("show_widget called",
		slog.String("bot_id", session.BotID),
		slog.String("title", title),
		slog.Int("html_len", len(html)),
	)

	return mcpgw.BuildToolSuccessResult(result), nil
}

func (e *Executor) callReadMe(arguments map[string]any) (map[string]any, error) {
	module := mcpgw.StringArg(arguments, "module")
	if module == "" {
		return mcpgw.BuildToolErrorResult("module is required"), nil
	}

	path, ok := validModules[module]
	if !ok {
		modules := make([]string, 0, len(validModules))
		for k := range validModules {
			modules = append(modules, k)
		}
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("unknown module %q, valid modules: %s", module, strings.Join(modules, ", "))), nil
	}

	// Read shared spec first, then module-specific spec
	shared, err := specsFS.ReadFile("specs/shared.md")
	if err != nil {
		e.logger.Error("failed to read shared spec", slog.String("error", err.Error()))
		return mcpgw.BuildToolErrorResult("failed to read shared design spec"), nil
	}

	content, err := specsFS.ReadFile(path)
	if err != nil {
		e.logger.Error("failed to read module spec", slog.String("module", module), slog.String("error", err.Error()))
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("failed to read spec for module %q", module)), nil
	}

	combined := string(shared) + "\n\n---\n\n" + string(content)

	return mcpgw.BuildToolSuccessResult(map[string]any{
		"ok":     true,
		"module": module,
		"spec":   combined,
	}), nil
}
