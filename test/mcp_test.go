package test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/mcp"
)

func setupMCPServer(t *testing.T) *mcp.Server {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)
	return mcp.NewServer(knowledgeBase, m, "1.2.0")
}

func sendMCPMessage(t *testing.T, s *mcp.Server, reqJSON string) map[string]any {
	in := strings.NewReader(reqJSON + "\n")
	var out bytes.Buffer

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Server.Serve failed: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON-RPC response: %v, raw: %s", err, out.String())
	}
	return resp
}

func TestMCPInitialize(t *testing.T) {
	server := setupMCPServer(t)

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	resp := sendMCPMessage(t, server, req)

	if resp["id"] != float64(1) {
		t.Errorf("Expected id 1, got %v", resp["id"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result map, got %v", resp["result"])
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion 2024-11-05, got %v", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok || serverInfo["name"] != "why-mcp-server" {
		t.Errorf("Invalid serverInfo: %v", serverInfo)
	}
}

func TestMCPToolsList(t *testing.T) {
	server := setupMCPServer(t)

	req := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	resp := sendMCPMessage(t, server, req)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result map, got %v", resp["result"])
	}

	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 3 {
		t.Fatalf("Expected 3 tools, got %v", tools)
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		tmap := tool.(map[string]any)
		toolNames[tmap["name"].(string)] = true
	}

	if !toolNames["why_explain"] || !toolNames["why_inspect_log"] || !toolNames["why_list_categories"] {
		t.Errorf("Missing required tools: %v", toolNames)
	}
}

func TestMCPToolCallExplain(t *testing.T) {
	server := setupMCPServer(t)

	req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"why_explain","arguments":{"error":"permission denied"}}}`
	resp := sendMCPMessage(t, server, req)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result map, got %v", resp["result"])
	}

	content := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("Expected content array in tool result")
	}

	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "WHY DID THIS HAPPEN?") || !strings.Contains(text, "permission denied") {
		t.Errorf("Unexpected tool output:\n%s", text)
	}
}

func TestMCPToolCallExplainJSON(t *testing.T) {
	server := setupMCPServer(t)

	req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"why_explain","arguments":{"error":"connection refused","format":"json"}}}`
	resp := sendMCPMessage(t, server, req)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result map, got %v", resp["result"])
	}

	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("Expected valid JSON in text field, got: %s (err: %v)", text, err)
	}
	if parsed["match_type"] != "exact" {
		t.Errorf("Expected match_type exact, got %v", parsed["match_type"])
	}
}

func TestMCPToolCallInspectLog(t *testing.T) {
	server := setupMCPServer(t)

	logText := `line 1: initializing
line 2: dial tcp 127.0.0.1:5432: connect: connection refused
line 3: retry failed: connection refused`

	callPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "why_inspect_log",
			"arguments": map[string]any{
				"log_content": logText,
			},
		},
	}
	data, _ := json.Marshal(callPayload)

	resp := sendMCPMessage(t, server, string(data))
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result map, got %v", resp["result"])
	}

	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "LOG INSPECTION REPORT") || !strings.Contains(text, "connection refused") {
		t.Errorf("Unexpected log inspection report:\n%s", text)
	}
}
