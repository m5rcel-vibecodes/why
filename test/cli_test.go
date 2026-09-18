package test

import (
	"strings"
	"testing"

	"github.com/m5rcel-vibecodes/why/internal/extractor"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
)

func TestExtractorMultilineLog(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	logOutput := `[2026-09-18 10:00:01] Starting application build...
[2026-09-18 10:00:03] Compiling assets
[2026-09-18 10:00:05] Connecting to database on port 5432
dial tcp 127.0.0.1:5432: connect: connection refused
[2026-09-18 10:00:06] Fatal exit with error code 1`

	res, line := extractor.ExtractAndMatch(strings.NewReader(logOutput), m)
	if res == nil {
		t.Fatal("Expected match from multiline log output")
	}

	if res.Rule.ID != "net-connection-refused" {
		t.Errorf("Expected net-connection-refused, got %s", res.Rule.ID)
	}

	if !strings.Contains(line, "connection refused") {
		t.Errorf("Expected matched line to contain 'connection refused', got %q", line)
	}
}

func TestExtractorGitMultiline(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	gitOutput := `To github.com:user/repo.git
 ! [rejected]        main -> main (non-fast-forward)
error: failed to push some refs to 'github.com:user/repo.git'
hint: Updates were rejected because the tip of your current branch is behind`

	res, _ := extractor.ExtractAndMatch(strings.NewReader(gitOutput), m)
	if res == nil {
		t.Fatal("Expected match from git push log")
	}

	if res.Rule.ID != "git-non-fast-forward" {
		t.Errorf("Expected git-non-fast-forward, got %s", res.Rule.ID)
	}
}
