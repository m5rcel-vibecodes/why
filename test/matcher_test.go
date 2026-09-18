package test

import (
	"testing"

	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

func TestMatcherExact(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	cases := []struct {
		input       string
		expectedID  string
		matchType   model.MatchType
		expectLevel string
	}{
		{"permission denied", "linux-permission-denied", model.MatchExact, "HIGH"},
		{"connection refused", "net-connection-refused", model.MatchExact, "HIGH"},
		{"command not found", "linux-command-not-found", model.MatchExact, "HIGH"},
		{"merge conflict", "git-merge-conflict", model.MatchExact, "HIGH"},
		{"cannot find module", "node-module-not-found", model.MatchExact, "HIGH"},
		{"modulenotfounderror", "py-module-not-found", model.MatchExact, "HIGH"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			res := m.Match(tc.input)
			if res == nil {
				t.Fatalf("Expected match for %q, got nil", tc.input)
			}
			if res.Rule.ID != tc.expectedID {
				t.Errorf("Expected rule ID %q, got %q", tc.expectedID, res.Rule.ID)
			}
			if res.MatchType != tc.matchType {
				t.Errorf("Expected match type %q, got %q", tc.matchType, res.MatchType)
			}
			if res.ConfidenceLevel != tc.expectLevel {
				t.Errorf("Expected confidence level %q, got %q", tc.expectLevel, res.ConfidenceLevel)
			}
		})
	}
}

func TestMatcherErrorCodes(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	cases := []struct {
		input      string
		expectedID string
	}{
		{"404", "http-404-not-found"},
		{"502", "http-502-bad-gateway"},
		{"EACCES", "linux-permission-denied"},
		{"CS0246", "dotnet-cs0246-type-not-found"},
		{"exit code 137", "docker-oom-killed"},
		{"errno 111", "net-connection-refused"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			res := m.Match(tc.input)
			if res == nil {
				t.Fatalf("Expected match for code %q, got nil", tc.input)
			}
			if res.Rule.ID != tc.expectedID {
				t.Errorf("Expected rule ID %q, got %q", tc.expectedID, res.Rule.ID)
			}
			if res.MatchType != model.MatchErrorCode {
				t.Errorf("Expected MatchErrorCode, got %s", res.MatchType)
			}
		})
	}
}

func TestMatcherRegexAndParamExtraction(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	// Test 1: connection refused with host & port
	res := m.Match("connect to 10.0.0.1:9000: Connection refused")
	if res == nil {
		t.Fatal("Expected match for connection refused with host:port")
	}
	if res.Rule.ID != "net-connection-refused" {
		t.Errorf("Expected net-connection-refused, got %s", res.Rule.ID)
	}
	if res.ExtractedParams["host"] != "10.0.0.1" {
		t.Errorf("Expected host 10.0.0.1, got %q", res.ExtractedParams["host"])
	}
	if res.ExtractedParams["port"] != "9000" {
		t.Errorf("Expected port 9000, got %q", res.ExtractedParams["port"])
	}

	// Verify parameter substitution into diagnostic step command
	foundSubstituted := false
	for _, step := range res.DiagnosticSteps {
		if step.Command == "nc -zv 10.0.0.1 9000" {
			foundSubstituted = true
			break
		}
	}
	if !foundSubstituted {
		t.Errorf("Expected substituted diagnostic command 'nc -zv 10.0.0.1 9000', got steps: %+v", res.DiagnosticSteps)
	}

	// Test 2: Python ModuleNotFoundError
	resPy := m.Match("ModuleNotFoundError: No module named 'scipy'")
	if resPy == nil {
		t.Fatal("Expected match for ModuleNotFoundError")
	}
	if resPy.ExtractedParams["module"] != "scipy" {
		t.Errorf("Expected module 'scipy', got %q", resPy.ExtractedParams["module"])
	}
}

func TestMatcherFuzzy(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	typoQueries := []struct {
		query      string
		expectedID string
	}{
		{"permissin denyd", "linux-permission-denied"},
		{"conection refused", "net-connection-refused"},
		{"comand not found", "linux-command-not-found"},
	}

	for _, tc := range typoQueries {
		t.Run(tc.query, func(t *testing.T) {
			res := m.Match(tc.query)
			if res == nil {
				t.Fatalf("Expected fuzzy match for %q, got nil", tc.query)
			}
			if res.Rule.ID != tc.expectedID {
				t.Errorf("Expected rule %q, got %q", tc.expectedID, res.Rule.ID)
			}
			if res.MatchType != model.MatchFuzzy {
				t.Errorf("Expected match type fuzzy, got %s", res.MatchType)
			}
			if res.ConfidenceScore <= 0 || res.ConfidenceScore >= 1.0 {
				t.Errorf("Invalid confidence score %f", res.ConfidenceScore)
			}
		})
	}
}
