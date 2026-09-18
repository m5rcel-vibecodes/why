package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/formatter"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/logscan"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 message.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 message.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ToolDefinition defines an MCP tool.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema defines the JSON schema for tool arguments.
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]PropertyDef `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// PropertyDef describes a parameter in InputSchema.
type PropertyDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Server implements a Model Context Protocol (MCP) server over stdio.
type Server struct {
	kb      *kb.KnowledgeBase
	matcher *matcher.Matcher
	version string
}

// NewServer creates a new MCP server.
func NewServer(k *kb.KnowledgeBase, m *matcher.Matcher, ver string) *Server {
	return &Server{
		kb:      k,
		matcher: m,
		version: ver,
	}
}

// Serve reads JSON-RPC requests from in and writes responses to out.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB line buffer

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(out, nil, -32700, "Parse error: "+err.Error())
			continue
		}

		s.handleRequest(out, &req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(out io.Writer, req *JSONRPCRequest) {
	// Notifications (no ID) do not receive responses
	isNotification := len(req.ID) == 0 || string(req.ID) == "null"

	switch req.Method {
	case "initialize":
		result := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "why-mcp-server",
				"version": s.version,
			},
		}
		s.sendResult(out, req.ID, result)

	case "notifications/initialized":
		// Standard MCP client notification; no response required
		return

	case "ping":
		if !isNotification {
			s.sendResult(out, req.ID, map[string]any{})
		}

	case "tools/list":
		s.sendResult(out, req.ID, map[string]any{
			"tools": s.getToolDefinitions(),
		})

	case "tools/call":
		s.handleToolCall(out, req)

	default:
		if !isNotification {
			s.sendError(out, req.ID, -32601, fmt.Sprintf("Method %q not found", req.Method))
		}
	}
}

func (s *Server) getToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name: "why_explain",
			Description: "Explains a terminal error, command failure, or exit code in plain language. " +
				"Provides verified findings, most likely cause, possible causes, safe next diagnostic steps, " +
				"actionable solutions, and safety warnings. 100% deterministic and runs offline.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"error": {
						Type:        "string",
						Description: "The exact error string, stderr output, or exit code (e.g. 'permission denied', 'connect: connection refused', 'CrashLoopBackOff', '404', 'exit code 137')",
					},
					"format": {
						Type:        "string",
						Description: "Output format: 'text' (human-readable terminal format) or 'json' (structured JSON). Defaults to 'text'.",
					},
				},
				Required: []string{"error"},
			},
		},
		{
			Name: "why_inspect_log",
			Description: "Analyzes multi-line log output or a log file, detects and groups distinct error patterns, " +
				"reports occurrence counts and line numbers, and aggregates recommended diagnostic checks.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"log_content": {
						Type:        "string",
						Description: "The raw log text to inspect.",
					},
					"file_path": {
						Type:        "string",
						Description: "Optional filesystem path to a log file to inspect.",
					},
					"format": {
						Type:        "string",
						Description: "Output format: 'text' or 'json'. Defaults to 'text'.",
					},
				},
			},
		},
		{
			Name: "why_list_categories",
			Description: "Lists all 19 supported systems domains and knowledge base categories available in why.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]PropertyDef{},
			},
		},
	}
}

func (s *Server) handleToolCall(out io.Writer, req *JSONRPCRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(out, req.ID, -32602, "Invalid params: "+err.Error())
		return
	}

	switch params.Name {
	case "why_explain":
		var args struct {
			Error  string `json:"error"`
			Format string `json:"format"`
		}
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &args)
		}
		if strings.TrimSpace(args.Error) == "" {
			s.sendToolError(out, req.ID, "Missing required argument 'error'")
			return
		}

		res := s.matcher.Match(args.Error)
		if res == nil {
			s.sendToolSuccess(out, req.ID, fmt.Sprintf("No matching error explanation found in local knowledge base for %q.", args.Error))
			return
		}

		if strings.ToLower(args.Format) == "json" {
			data, err := formatter.FormatJSON(res)
			if err != nil {
				s.sendToolError(out, req.ID, "JSON formatting failed: "+err.Error())
				return
			}
			s.sendToolSuccess(out, req.ID, string(data))
		} else {
			styler := formatter.NewStyler(formatter.ColorNever)
			text := formatter.FormatTerminal(res, styler)
			s.sendToolSuccess(out, req.ID, text)
		}

	case "why_inspect_log":
		var args struct {
			LogContent string `json:"log_content"`
			FilePath   string `json:"file_path"`
			Format     string `json:"format"`
		}
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &args)
		}

		var report *logscan.LogReport
		var err error

		if args.FilePath != "" {
			report, err = logscan.ScanFile(args.FilePath, s.matcher)
		} else if args.LogContent != "" {
			report, err = logscan.Scan(strings.NewReader(args.LogContent), "inline_log", s.matcher)
		} else {
			s.sendToolError(out, req.ID, "Must provide either 'log_content' or 'file_path'")
			return
		}

		if err != nil {
			s.sendToolError(out, req.ID, "Log inspection failed: "+err.Error())
			return
		}

		if strings.ToLower(args.Format) == "json" {
			data, err := logscan.FormatJSONReport(report)
			if err != nil {
				s.sendToolError(out, req.ID, "JSON report failed: "+err.Error())
				return
			}
			s.sendToolSuccess(out, req.ID, string(data))
		} else {
			styler := formatter.NewStyler(formatter.ColorNever)
			text := logscan.FormatTerminalReport(report, styler)
			s.sendToolSuccess(out, req.ID, text)
		}

	case "why_list_categories":
		categoriesMap := make(map[string]int)
		for _, r := range s.kb.Rules {
			categoriesMap[r.Category]++
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("why local knowledge base contains %d rules across %d domains:\n\n", len(s.kb.Rules), len(categoriesMap)))
		for cat, count := range categoriesMap {
			sb.WriteString(fmt.Sprintf("• %s (%d rules)\n", cat, count))
		}
		s.sendToolSuccess(out, req.ID, sb.String())

	default:
		s.sendToolError(out, req.ID, fmt.Sprintf("Unknown tool %q", params.Name))
	}
}

func (s *Server) sendToolSuccess(out io.Writer, id json.RawMessage, text string) {
	result := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": false,
	}
	s.sendResult(out, id, result)
}

func (s *Server) sendToolError(out io.Writer, id json.RawMessage, errMsg string) {
	result := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": errMsg,
			},
		},
		"isError": true,
	}
	s.sendResult(out, id, result)
}

func (s *Server) sendResult(out io.Writer, id json.RawMessage, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeJSON(out, resp)
}

func (s *Server) sendError(out io.Writer, id json.RawMessage, code int, msg string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: msg,
		},
	}
	s.writeJSON(out, resp)
}

func (s *Server) writeJSON(out io.Writer, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal response: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(out, "%s\n", data)
}
