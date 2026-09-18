package model

// DiagnosticStep represents an actionable, safe command to inspect the issue.
type DiagnosticStep struct {
	Command     string `yaml:"command" json:"command"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Destructive bool   `yaml:"destructive,omitempty" json:"destructive,omitempty"`
}

// Solution represents a potential fix or resolution.
type Solution struct {
	Description string `yaml:"description" json:"description"`
	Command     string `yaml:"command,omitempty" json:"command,omitempty"`
	Warning     string `yaml:"warning,omitempty" json:"warning,omitempty"`
}

// ErrorRule is a knowledge base entry for a specific error.
type ErrorRule struct {
	ID                 string           `yaml:"id" json:"id"`
	Error              string           `yaml:"error" json:"error"`
	Category           string           `yaml:"category" json:"category"`
	Meaning            string           `yaml:"meaning" json:"meaning"`
	ExactMatches       []string         `yaml:"exact_matches,omitempty" json:"exact_matches,omitempty"`
	Patterns           []string         `yaml:"patterns,omitempty" json:"patterns,omitempty"`
	ErrorCodes         []string         `yaml:"error_codes,omitempty" json:"error_codes,omitempty"`
	ConfirmedTemplates []string         `yaml:"confirmed_templates,omitempty" json:"confirmed_templates,omitempty"`
	LikelyCause        string           `yaml:"likely_cause" json:"likely_cause"`
	PossibleCauses     []string         `yaml:"possible_causes" json:"possible_causes"`
	DiagnosticSteps    []DiagnosticStep `yaml:"diagnostic_steps" json:"diagnostic_steps"`
	PotentialSolutions []Solution       `yaml:"potential_solutions,omitempty" json:"potential_solutions,omitempty"`
	Warnings           []string         `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

// MatchType defines how an input matched a rule.
type MatchType string

const (
	MatchExact     MatchType = "exact"
	MatchErrorCode MatchType = "error_code"
	MatchRegex     MatchType = "regex"
	MatchFuzzy     MatchType = "fuzzy"
)

// MatchResult is the result returned by the matcher engine.
type MatchResult struct {
	Rule               *ErrorRule        `json:"rule"`
	MatchType          MatchType         `json:"match_type"`
	ConfidenceScore    float64           `json:"confidence_score"`
	ConfidenceLevel    string            `json:"confidence_level"` // HIGH, MEDIUM, LOW
	MatchedInput       string            `json:"matched_input"`
	ExtractedParams    map[string]string `json:"extracted_params,omitempty"`
	ConfirmedFromInput []string          `json:"confirmed_from_input"`
	LikelyCause        string            `json:"likely_cause"`
	PossibleCauses     []string          `json:"possible_causes"`
	DiagnosticSteps    []DiagnosticStep  `json:"diagnostic_steps"`
	PotentialSolutions []Solution        `json:"potential_solutions,omitempty"`
	Warnings           []string          `json:"warnings,omitempty"`
	Source             string            `json:"source"` // "knowledge_base" or "ai"
}
