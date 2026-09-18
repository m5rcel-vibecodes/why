package extractor

import (
	"bufio"
	"io"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

const maxInputBytes = 1024 * 1024 // 1 MB limit for stdin processing

// ExtractAndMatch processes text (which may be a multi-line log from stdin or command output)
// and returns the best MatchResult found across the lines.
func ExtractAndMatch(r io.Reader, m *matcher.Matcher) (*model.MatchResult, string) {
	lr := io.LimitReader(r, maxInputBytes)
	scanner := bufio.NewScanner(lr)

	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return nil, ""
	}

	// If there is only one line, test directly
	if len(lines) == 1 {
		res := m.Match(lines[0])
		return res, lines[0]
	}

	// Multi-line scan: first test lines with common error keywords backwards (most recent first)
	var bestResult *model.MatchResult
	var bestMatchedLine string

	// Prioritize lines with error prefixes
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		res := m.Match(line)
		if res != nil {
			if bestResult == nil || res.ConfidenceScore > bestResult.ConfidenceScore {
				bestResult = res
				bestMatchedLine = line
			}
			// If high confidence exact/regex match, stop early
			if res.ConfidenceScore >= 0.95 {
				return res, line
			}
		}
	}

	if bestResult != nil {
		return bestResult, bestMatchedLine
	}

	// Fallback: try matching the concatenated last 3 lines
	joined := strings.Join(lines[max(0, len(lines)-3):], " ")
	res := m.Match(joined)
	return res, joined
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
