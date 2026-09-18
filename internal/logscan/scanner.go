package logscan

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

// LogFinding aggregates occurrences of a specific matched error in a log file.
type LogFinding struct {
	Rule               *model.ErrorRule       `json:"rule"`
	Error              string                 `json:"error"`
	Category           string                 `json:"category"`
	Count              int                    `json:"count"`
	LineNumbers        []int                  `json:"line_numbers"`
	SampleLine         string                 `json:"sample_line"`
	ExtractedParams    map[string]string      `json:"extracted_params,omitempty"`
	LikelyCause        string                 `json:"likely_cause"`
	PossibleCauses     []string               `json:"possible_causes"`
	DiagnosticSteps    []model.DiagnosticStep `json:"diagnostic_steps"`
	PotentialSolutions []model.Solution       `json:"potential_solutions,omitempty"`
	Warnings           []string               `json:"warnings,omitempty"`
}

// LogReport contains the complete scan report of a log file.
type LogReport struct {
	Source         string        `json:"source"`
	TotalLines     int           `json:"total_lines"`
	TotalErrors    int           `json:"total_errors"`
	DistinctErrors int           `json:"distinct_errors"`
	Findings       []*LogFinding `json:"findings"`
}

// ScanFile opens and analyzes a log file from path (or stdin if source is "-").
func ScanFile(source string, m *matcher.Matcher) (*LogReport, error) {
	var r io.Reader
	displayName := source

	if source == "-" || source == "stdin" {
		r = os.Stdin
		displayName = "standard input (stdin)"
	} else {
		f, err := os.Open(source)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", source, err)
		}
		defer f.Close()
		r = f
	}

	return Scan(r, displayName, m)
}

// Scan reads through a log stream and identifies and aggregates error patterns.
func Scan(r io.Reader, sourceName string, m *matcher.Matcher) (*LogReport, error) {
	scanner := bufio.NewScanner(r)
	// Allow large log lines (up to 1MB per line)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	findingsMap := make(map[string]*LogFinding)
	lineNum := 0
	totalErrors := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Quick heuristic check: avoid scoring pure informational/debug lines without error indicators
		lower := strings.ToLower(line)
		hasErrorKeyword := strings.Contains(lower, "error") ||
			strings.Contains(lower, "fatal") ||
			strings.Contains(lower, "fail") ||
			strings.Contains(lower, "panic") ||
			strings.Contains(lower, "exception") ||
			strings.Contains(lower, "denied") ||
			strings.Contains(lower, "refused") ||
			strings.Contains(lower, "timeout") ||
			strings.Contains(lower, "warn") ||
			strings.Contains(lower, "broken") ||
			strings.Contains(lower, "cannot") ||
			strings.Contains(lower, "could not") ||
			strings.Contains(lower, "crash") ||
			strings.Contains(lower, "errno") ||
			strings.Contains(lower, "status code") ||
			strings.Contains(lower, "exit code") ||
			strings.Contains(lower, "backoff") ||
			strings.Contains(lower, "oom")

		if !hasErrorKeyword {
			continue
		}

		res := m.Match(line)
		if res != nil && res.Rule != nil && res.ConfidenceScore >= 0.75 {
			totalErrors++
			finding, exists := findingsMap[res.Rule.ID]
			if !exists {
				finding = &LogFinding{
					Rule:               res.Rule,
					Error:              res.Rule.Error,
					Category:           res.Rule.Category,
					Count:              0,
					LineNumbers:        make([]int, 0),
					SampleLine:         line,
					ExtractedParams:    res.ExtractedParams,
					LikelyCause:        res.LikelyCause,
					PossibleCauses:     res.PossibleCauses,
					DiagnosticSteps:    res.DiagnosticSteps,
					PotentialSolutions: res.PotentialSolutions,
					Warnings:           res.Warnings,
				}
				findingsMap[res.Rule.ID] = finding
			}
			finding.Count++
			if len(finding.LineNumbers) < 50 { // cap stored line numbers to avoid unbounded memory
				finding.LineNumbers = append(finding.LineNumbers, lineNum)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log stream: %w", err)
	}

	// Sort findings by count descending (most frequent error first)
	findingsList := make([]*LogFinding, 0, len(findingsMap))
	for _, f := range findingsMap {
		findingsList = append(findingsList, f)
	}
	sort.Slice(findingsList, func(i, j int) bool {
		return findingsList[i].Count > findingsList[j].Count
	})

	return &LogReport{
		Source:         sourceName,
		TotalLines:     lineNum,
		TotalErrors:    totalErrors,
		DistinctErrors: len(findingsList),
		Findings:       findingsList,
	}, nil
}
