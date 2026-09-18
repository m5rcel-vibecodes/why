package matcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

var (
	// Matches explicitly prefixed numeric error codes like "exit code 137", "HTTP 404", "errno 111", "status 502"
	prefixedNumericRegex = regexp.MustCompile(`(?i)(?:exit\s+code|status(?:\s+code)?|errno|http|code)\s*[:=]?\s*([0-9]{1,4})\b`)
	// Matches symbolic error codes like EACCES, ECONNREFUSED, CS0246, MSB4019
	symbolicCodeRegex = regexp.MustCompile(`\b([A-Z]{2,}[0-9]+|[A-Z]{3,})\b`)
)

// Matcher performs deterministic local error rule matching.
type Matcher struct {
	kb *kb.KnowledgeBase
}

// New creates a new Matcher instance.
func New(kb *kb.KnowledgeBase) *Matcher {
	return &Matcher{kb: kb}
}

// Match evaluates the input string against all knowledge base rules.
// Deterministic order: Exact Match -> Regex Pattern -> Error Code -> Fuzzy Match.
func (m *Matcher) Match(input string) *model.MatchResult {
	cleaned := strings.TrimSpace(input)
	if cleaned == "" {
		return nil
	}

	lower := strings.ToLower(cleaned)

	// 1. Error Code Matching (for single-token codes like 404, EACCES or prefixed like 'exit code 137')
	if res := m.matchErrorCode(cleaned, lower); res != nil {
		return res
	}

	// 2. Exact Matching
	if res := m.matchExact(cleaned, lower); res != nil {
		return res
	}

	// 3. Regex Pattern Matching
	if res := m.matchRegex(cleaned); res != nil {
		return res
	}

	// 4. Fuzzy Matching
	if res := m.matchFuzzy(cleaned); res != nil {
		return res
	}

	return nil
}

func (m *Matcher) matchErrorCode(original, lower string) *model.MatchResult {
	var candidates []string

	// Single token input (e.g. "404", "137", "EACCES", "CS0246")
	tokens := strings.Fields(original)
	if len(tokens) == 1 {
		candidates = append(candidates, strings.ToUpper(tokens[0]))
	} else {
		// In multi-token phrases, look for explicitly prefixed numeric codes (e.g. "exit code 137")
		for _, m := range prefixedNumericRegex.FindAllStringSubmatch(original, -1) {
			if len(m) > 1 {
				candidates = append(candidates, m[1])
			}
		}
		// Or standalone symbolic codes (e.g. "failed with EACCES")
		for _, m := range symbolicCodeRegex.FindAllStringSubmatch(original, -1) {
			if len(m) > 1 {
				candidates = append(candidates, strings.ToUpper(m[1]))
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	for _, cand := range candidates {
		for _, rule := range m.kb.Rules {
			for _, code := range rule.ErrorCodes {
				if strings.EqualFold(code, cand) {
					params := map[string]string{"code": cand}
					return m.buildResult(rule, model.MatchErrorCode, 0.95, original, params, []string{
						fmt.Sprintf("Matched error code: %s", cand),
					})
				}
			}
		}
	}
	return nil
}

func (m *Matcher) matchExact(original, lower string) *model.MatchResult {
	for _, rule := range m.kb.Rules {
		if strings.ToLower(rule.Error) == lower || strings.ToLower(rule.ID) == lower {
			return m.buildResult(rule, model.MatchExact, 1.0, original, nil, []string{
				fmt.Sprintf("Direct error match: %s", rule.Error),
			})
		}
		for _, exact := range rule.ExactMatches {
			if strings.ToLower(exact) == lower {
				return m.buildResult(rule, model.MatchExact, 1.0, original, nil, []string{
					fmt.Sprintf("Exact error match: %s", rule.Error),
				})
			}
		}
	}
	return nil
}

func (m *Matcher) matchRegex(original string) *model.MatchResult {
	for _, rule := range m.kb.Rules {
		for _, pattern := range rule.Patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}

			if !re.MatchString(original) {
				continue
			}

			submatches := re.FindStringSubmatch(original)
			params := make(map[string]string)
			for i, name := range re.SubexpNames() {
				if i > 0 && i < len(submatches) && name != "" && submatches[i] != "" {
					params[name] = submatches[i]
				}
			}

			confidence := 0.90
			if len(params) > 0 {
				confidence = 0.95
			}

			var confirmed []string
			for _, tmpl := range rule.ConfirmedTemplates {
				rendered := interpolateTemplate(tmpl, params)
				if rendered != "" {
					confirmed = append(confirmed, rendered)
				}
			}
			if len(confirmed) == 0 {
				confirmed = append(confirmed, fmt.Sprintf("Matched error pattern: %s", rule.Error))
			}

			return m.buildResult(rule, model.MatchRegex, confidence, original, params, confirmed)
		}
	}
	return nil
}

func (m *Matcher) matchFuzzy(original string) *model.MatchResult {
	var bestRule *model.ErrorRule
	var bestScore float64

	for _, rule := range m.kb.Rules {
		// Test against error title
		s := FuzzyScore(original, rule.Error)
		if s > bestScore {
			bestScore = s
			bestRule = rule
		}

		// Test against exact match synonyms
		for _, exact := range rule.ExactMatches {
			s := FuzzyScore(original, exact)
			if s > bestScore {
				bestScore = s
				bestRule = rule
			}
		}
	}

	// Threshold for fuzzy matching: 0.45 to prevent random false positives
	if bestScore >= 0.45 && bestRule != nil {
		confirmed := []string{
			fmt.Sprintf("Approximate match (%.0f%% similarity): %s", bestScore*100, bestRule.Error),
			"Notice: Match determined via fuzzy similarity; check if error matches your scenario.",
		}
		return m.buildResult(bestRule, model.MatchFuzzy, bestScore, original, nil, confirmed)
	}

	return nil
}

func (m *Matcher) buildResult(
	rule *model.ErrorRule,
	matchType model.MatchType,
	score float64,
	input string,
	params map[string]string,
	confirmed []string,
) *model.MatchResult {
	if params == nil {
		params = make(map[string]string)
	}

	// Calculate confidence level
	level := "HIGH"
	if score < 0.65 {
		level = "LOW"
	} else if score < 0.85 {
		level = "MEDIUM"
	}

	// Interpolate Likely Cause
	likely := interpolateTemplate(rule.LikelyCause, params)

	// Interpolate Possible Causes
	possible := make([]string, len(rule.PossibleCauses))
	for i, c := range rule.PossibleCauses {
		possible[i] = interpolateTemplate(c, params)
	}

	// Interpolate Diagnostic Steps
	steps := make([]model.DiagnosticStep, len(rule.DiagnosticSteps))
	for i, s := range rule.DiagnosticSteps {
		steps[i] = model.DiagnosticStep{
			Command:     interpolateTemplate(s.Command, params),
			Description: interpolateTemplate(s.Description, params),
			Destructive: s.Destructive,
		}
	}

	// Interpolate Potential Solutions
	solutions := make([]model.Solution, len(rule.PotentialSolutions))
	for i, sol := range rule.PotentialSolutions {
		solutions[i] = model.Solution{
			Description: interpolateTemplate(sol.Description, params),
			Command:     interpolateTemplate(sol.Command, params),
			Warning:     interpolateTemplate(sol.Warning, params),
		}
	}

	// Interpolate Warnings
	warnings := make([]string, len(rule.Warnings))
	for i, w := range rule.Warnings {
		warnings[i] = interpolateTemplate(w, params)
	}

	return &model.MatchResult{
		Rule:               rule,
		MatchType:          matchType,
		ConfidenceScore:    score,
		ConfidenceLevel:    level,
		MatchedInput:       input,
		ExtractedParams:    params,
		ConfirmedFromInput: confirmed,
		LikelyCause:        likely,
		PossibleCauses:     possible,
		DiagnosticSteps:    steps,
		PotentialSolutions: solutions,
		Warnings:           warnings,
		Source:             "knowledge_base",
	}
}
