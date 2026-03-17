package remotefs

import (
	"context"
	"log/slog"

	mcpgw "github.com/memohai/memoh/internal/mcp"
)

const (
	toolListFiles   = "remote_list_files"
	toolReadFile    = "remote_read_file"
	toolSearchFiles = "remote_search_files"
)

// DeviceInfo holds resolved device connection details.
type DeviceInfo struct {
	DeviceID string
	UserID   string
	TsIP     string
	Token    string
	Hostname string
}

// DeviceResolver resolves a bot's owner to their online remote devices.
type DeviceResolver interface {
	ResolveDevice(ctx context.Context, botID string, session mcpgw.ToolSessionContext) (DeviceInfo, error)
}

// AuditLogger records file access operations.
type AuditLogger interface {
	LogAccess(ctx context.Context, entry AuditEntry)
}

// AuditEntry represents a single file access audit record.
type AuditEntry struct {
	DeviceID   string
	BotID      string
	UserID     string
	Action     string
	FilePath   string
	FileSize   int64
	Result     string
	ErrorMsg   string
	DurationMs int
}

// Executor provides remote filesystem tools that access files on desktop clients via Tailscale.
type Executor struct {
	resolver DeviceResolver
	client   *Client
	audit    AuditLogger
	logger   *slog.Logger
}

// NewExecutor returns a remotefs tool executor.
func NewExecutor(log *slog.Logger, resolver DeviceResolver, audit AuditLogger) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		resolver: resolver,
		client:   NewClient(log),
		audit:    audit,
		logger:   log.With(slog.String("provider", "remotefs_tool")),
	}
}

// ListTools returns the remote filesystem tool descriptors.
func (*Executor) ListTools(_ context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if session.IsSubagent {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolListFiles,
			Description: "List files and directories on the user's desktop computer. The files are accessed via a secure Tailscale connection to their local machine.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path within the shared directory. Use empty string or '.' for root.",
					},
					"recursive": map[string]any{
						"type":        "boolean",
						"description": "Whether to list recursively. Default: false.",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "Optional glob pattern to filter files (e.g. '*.md', '*.go').",
					},
				},
			},
		},
		{
			Name:        toolReadFile,
			Description: "Read the content of a file on the user's desktop computer. Supports pagination for large files.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path to the file within the shared directory.",
					},
					"offset": map[string]any{
						"type":        "number",
						"description": "Line number to start reading from (0-based). Default: 0.",
					},
					"limit": map[string]any{
						"type":        "number",
						"description": "Maximum number of lines to read. Default: 500.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        toolSearchFiles,
			Description: "Search for text content across files on the user's desktop computer.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query string.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Optional directory to limit the search scope.",
					},
					"glob": map[string]any{
						"type":        "string",
						"description": "Optional file type filter (e.g. '*.md').",
					},
					"max_results": map[string]any{
						"type":        "number",
						"description": "Maximum number of results. Default: 20.",
					},
				},
				"required": []string{"query"},
			},
		},
	}, nil
}

// CallTool handles remote filesystem tool calls.
func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	switch toolName {
	case toolListFiles:
		return e.callListFiles(ctx, session, arguments)
	case toolReadFile:
		return e.callReadFile(ctx, session, arguments)
	case toolSearchFiles:
		return e.callSearchFiles(ctx, session, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

func (e *Executor) resolveOrError(ctx context.Context, session mcpgw.ToolSessionContext) (DeviceInfo, error) {
	device, err := e.resolver.ResolveDevice(ctx, session.BotID, session)
	if err != nil {
		return DeviceInfo{}, err
	}
	return device, nil
}

func (e *Executor) callListFiles(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	device, err := e.resolveOrError(ctx, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult("Failed to connect to your desktop: " + err.Error()), nil
	}

	path := mcpgw.StringArg(arguments, "path")
	recursive, _, _ := mcpgw.BoolArg(arguments, "recursive")
	pattern := mcpgw.StringArg(arguments, "pattern")

	result, duration, err := e.client.ListFiles(ctx, device, ListFilesRequest{
		Path:      path,
		Recursive: recursive,
		Pattern:   pattern,
	})

	action := "list"
	if e.audit != nil {
		entry := AuditEntry{
			DeviceID:   device.DeviceID,
			BotID:      session.BotID,
			UserID:     device.UserID,
			Action:     action,
			FilePath:   path,
			DurationMs: int(duration.Milliseconds()),
		}
		if err != nil {
			entry.Result = "error"
			entry.ErrorMsg = err.Error()
		} else {
			entry.Result = "success"
		}
		e.audit.LogAccess(ctx, entry)
	}

	if err != nil {
		return mcpgw.BuildToolErrorResult("Failed to list files: " + err.Error()), nil
	}

	return mcpgw.BuildToolSuccessResult(result), nil
}

func (e *Executor) callReadFile(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	device, err := e.resolveOrError(ctx, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult("Failed to connect to your desktop: " + err.Error()), nil
	}

	path := mcpgw.StringArg(arguments, "path")
	if path == "" {
		return mcpgw.BuildToolErrorResult("path is required"), nil
	}

	offset, _, _ := mcpgw.IntArg(arguments, "offset")
	limit, _, _ := mcpgw.IntArg(arguments, "limit")

	result, duration, err := e.client.ReadFile(ctx, device, ReadFileRequest{
		Path:   path,
		Offset: offset,
		Limit:  limit,
	})

	if e.audit != nil {
		entry := AuditEntry{
			DeviceID:   device.DeviceID,
			BotID:      session.BotID,
			UserID:     device.UserID,
			Action:     "read",
			FilePath:   path,
			DurationMs: int(duration.Milliseconds()),
		}
		if err != nil {
			entry.Result = "error"
			entry.ErrorMsg = err.Error()
		} else {
			entry.Result = "success"
			if size, ok := result["size"].(int64); ok {
				entry.FileSize = size
			}
		}
		e.audit.LogAccess(ctx, entry)
	}

	if err != nil {
		return mcpgw.BuildToolErrorResult("Failed to read file: " + err.Error()), nil
	}

	return mcpgw.BuildToolSuccessResult(result), nil
}

func (e *Executor) callSearchFiles(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	device, err := e.resolveOrError(ctx, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult("Failed to connect to your desktop: " + err.Error()), nil
	}

	query := mcpgw.StringArg(arguments, "query")
	if query == "" {
		return mcpgw.BuildToolErrorResult("query is required"), nil
	}

	path := mcpgw.StringArg(arguments, "path")
	glob := mcpgw.StringArg(arguments, "glob")
	maxResults, _, _ := mcpgw.IntArg(arguments, "max_results")

	result, duration, err := e.client.SearchFiles(ctx, device, SearchFilesRequest{
		Query:      query,
		Path:       path,
		Glob:       glob,
		MaxResults: maxResults,
	})

	if e.audit != nil {
		entry := AuditEntry{
			DeviceID:   device.DeviceID,
			BotID:      session.BotID,
			UserID:     device.UserID,
			Action:     "search",
			FilePath:   path,
			DurationMs: int(duration.Milliseconds()),
		}
		if err != nil {
			entry.Result = "error"
			entry.ErrorMsg = err.Error()
		} else {
			entry.Result = "success"
		}
		e.audit.LogAccess(ctx, entry)
	}

	if err != nil {
		return mcpgw.BuildToolErrorResult("Failed to search files: " + err.Error()), nil
	}

	return mcpgw.BuildToolSuccessResult(result), nil
}
