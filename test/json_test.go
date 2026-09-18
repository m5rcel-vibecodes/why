package test

import (
	"encoding/json"
	"testing"

	"github.com/m5rcel-vibecodes/why/internal/formatter"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
)

func TestJSONFormatter(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	res := m.Match("permission denied")
	if res == nil {
		t.Fatal("Expected match for 'permission denied'")
	}

	data, err := formatter.FormatJSON(res)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	expectedKeys := []string{
		"rule",
		"match_type",
		"confidence_score",
		"confidence_level",
		"matched_input",
		"confirmed_from_input",
		"likely_cause",
		"possible_causes",
		"diagnostic_steps",
		"warnings",
	}

	for _, k := range expectedKeys {
		if _, ok := parsed[k]; !ok {
			t.Errorf("JSON output missing expected key: %s", k)
		}
	}

	// Test nil match result produces valid JSON
	nilData, err := formatter.FormatJSON(nil)
	if err != nil {
		t.Fatalf("FormatJSON(nil) failed: %v", err)
	}
	var nilParsed map[string]any
	if err := json.Unmarshal(nilData, &nilParsed); err != nil {
		t.Fatalf("Failed to parse nil JSON output: %v", err)
	}
	if nilParsed["found"] != false {
		t.Errorf("Expected found: false in nil JSON output, got: %v", nilParsed["found"])
	}
}
