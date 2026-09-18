package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/m5rcel-vibecodes/why/internal/formatter"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/logscan"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
)

func TestLogScanAggregation(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	logData := `[2026-09-18 10:00:01] INFO Starting worker pool
[2026-09-18 10:00:02] ERROR dial tcp 127.0.0.1:5432: connect: connection refused
[2026-09-18 10:00:03] INFO Retrying database connection
[2026-09-18 10:00:04] ERROR dial tcp 127.0.0.1:5432: connect: connection refused
[2026-09-18 10:00:05] WARN write error: no space left on device
[2026-09-18 10:00:06] ERROR dial tcp 127.0.0.1:5432: connect: connection refused
[2026-09-18 10:00:07] FATAL pod entered CrashLoopBackOff state
[2026-09-18 10:00:08] INFO Shutdown initiated`

	report, err := logscan.Scan(strings.NewReader(logData), "test.log", m)
	if err != nil {
		t.Fatalf("Log scan failed: %v", err)
	}

	if report.TotalLines != 8 {
		t.Errorf("Expected 8 lines scanned, got %d", report.TotalLines)
	}
	if report.TotalErrors != 5 {
		t.Errorf("Expected 5 total error occurrences, got %d", report.TotalErrors)
	}
	if report.DistinctErrors != 3 {
		t.Errorf("Expected 3 distinct error types, got %d", report.DistinctErrors)
	}

	// Verify top error is connection refused with count 3
	if len(report.Findings) == 0 {
		t.Fatal("No findings returned")
	}
	top := report.Findings[0]
	if top.Rule.ID != "net-connection-refused" {
		t.Errorf("Expected top finding to be net-connection-refused, got %s", top.Rule.ID)
	}
	if top.Count != 3 {
		t.Errorf("Expected 3 occurrences of connection refused, got %d", top.Count)
	}
	if len(top.LineNumbers) != 3 {
		t.Errorf("Expected 3 line numbers, got %v", top.LineNumbers)
	}

	// Test Terminal Report Formatting
	styler := formatter.NewStyler(formatter.ColorNever)
	termReport := logscan.FormatTerminalReport(report, styler)
	if !strings.Contains(termReport, "LOG INSPECTION REPORT") {
		t.Errorf("Expected terminal report header, got:\n%s", termReport)
	}
	if !strings.Contains(termReport, "3x") {
		t.Errorf("Expected occurrence count '3x' in report, got:\n%s", termReport)
	}

	// Test JSON Report Formatting
	jsonData, err := logscan.FormatJSONReport(report)
	if err != nil {
		t.Fatalf("JSON format failed: %v", err)
	}
	var parsed logscan.LogReport
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON report: %v", err)
	}
	if parsed.DistinctErrors != 3 {
		t.Errorf("Expected 3 distinct errors in JSON report, got %d", parsed.DistinctErrors)
	}
}

func TestLogScanCleanLog(t *testing.T) {
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
	m := matcher.New(knowledgeBase)

	cleanData := `[2026-09-18 10:00:01] INFO Server listening on :8080
[2026-09-18 10:00:02] INFO Health check passed HTTP 200 OK
[2026-09-18 10:00:03] INFO Processed 42 items successfully`

	report, err := logscan.Scan(strings.NewReader(cleanData), "clean.log", m)
	if err != nil {
		t.Fatalf("Log scan failed: %v", err)
	}

	if report.TotalErrors != 0 {
		t.Errorf("Expected 0 errors on clean log, got %d", report.TotalErrors)
	}
	if report.DistinctErrors != 0 {
		t.Errorf("Expected 0 distinct errors, got %d", report.DistinctErrors)
	}
}
