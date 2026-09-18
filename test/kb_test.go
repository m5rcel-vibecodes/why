package test

import (
	"strings"
	"testing"

	"github.com/m5rcel-vibecodes/why/internal/kb"
)

func TestKnowledgeBaseIntegrity(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load embedded knowledge base: %v", err)
	}

	if len(knowledgeBase.Rules) == 0 {
		t.Fatal("No rules loaded from embedded knowledge base")
	}

	requiredCategories := []string{
		"Linux",
		"Git",
		"Docker",
		"systemd",
		"OpenRC",
		"Networking",
		"DNS",
		"HTTP",
		"TLS",
		"Package Managers",
		"Node.js",
		"Python",
		"Go",
		".NET",
		"SSH",
		"Database",
		"Kubernetes",
		"Rust",
		"Java",
	}

	categoryCounts := make(map[string]int)
	seenIDs := make(map[string]bool)

	for _, rule := range knowledgeBase.Rules {
		if rule.ID == "" {
			t.Errorf("Found rule with empty ID")
		}
		if seenIDs[rule.ID] {
			t.Errorf("Duplicate rule ID found: %s", rule.ID)
		}
		seenIDs[rule.ID] = true

		if rule.Error == "" {
			t.Errorf("Rule %s has empty error name", rule.ID)
		}
		if rule.Category == "" {
			t.Errorf("Rule %s has empty category", rule.ID)
		}
		categoryCounts[strings.ToLower(rule.Category)]++

		if rule.Meaning == "" {
			t.Errorf("Rule %s has empty meaning", rule.ID)
		}
		if rule.LikelyCause == "" {
			t.Errorf("Rule %s has empty likely_cause", rule.ID)
		}
		if len(rule.DiagnosticSteps) == 0 {
			t.Errorf("Rule %s has no diagnostic steps", rule.ID)
		}
		for i, step := range rule.DiagnosticSteps {
			if strings.TrimSpace(step.Command) == "" {
				t.Errorf("Rule %s diagnostic step %d has empty command", rule.ID, i)
			}
		}
	}

	for _, req := range requiredCategories {
		if categoryCounts[strings.ToLower(req)] == 0 {
			t.Errorf("Required category %q has 0 rules loaded", req)
		}
	}

	t.Logf("Successfully verified %d rules across %d categories", len(knowledgeBase.Rules), len(categoryCounts))
}
