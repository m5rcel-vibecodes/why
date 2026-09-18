package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/m5rcel-vibecodes/why/internal/config"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

// Client handles optional, opt-in AI queries.
type Client struct {
	cfg        *config.AIConfig
	httpClient *http.Client
}

// NewClient creates a new AI client.
func NewClient(cfg *config.AIConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type aiResponseSchema struct {
	Error              string                 `json:"error"`
	Meaning            string                 `json:"meaning"`
	LikelyCause        string                 `json:"likely_cause"`
	PossibleCauses     []string               `json:"possible_causes"`
	DiagnosticSteps    []model.DiagnosticStep `json:"diagnostic_steps"`
	PotentialSolutions []model.Solution       `json:"potential_solutions"`
	Warnings           []string               `json:"warnings"`
}

// Query sends the error string to the configured AI provider.
func (c *Client) Query(ctx context.Context, input string) (*model.MatchResult, error) {
	prompt := fmt.Sprintf(`You are a systems diagnostic assistant. Explain this command-line error safely and concisely in valid JSON.
Error input: %s

You MUST respond strictly with a valid JSON object matching this schema:
{
  "error": "short error name",
  "meaning": "plain language explanation of what happened",
  "likely_cause": "most probable direct cause",
  "possible_causes": ["bullet 1", "bullet 2"],
  "diagnostic_steps": [
    {"command": "safe inspection command", "description": "what this checks", "destructive": false}
  ],
  "potential_solutions": [
    {"description": "how to safely solve", "command": "command to fix", "warning": "any caveat"}
  ],
  "warnings": [
    "Safety warnings, e.g. avoiding blind sudo or destructive actions"
  ]
}
Do NOT wrap in markdown backticks if possible, return raw JSON. Never suggest destructive commands as simple diagnostic steps.`, input)

	apiKey := c.cfg.APIKey
	provider := strings.ToLower(c.cfg.Provider)

	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		apiKey = geminiKey
		provider = "gemini"
	} else if openAIKey := os.Getenv("OPENAI_API_KEY"); openAIKey != "" {
		apiKey = openAIKey
		provider = "openai"
	}

	var jsonBytes []byte
	var err error

	switch provider {
	case "gemini":
		if apiKey == "" {
			return nil, fmt.Errorf("gemini provider selected but GEMINI_API_KEY environment variable is not set")
		}
		jsonBytes, err = c.callGemini(ctx, apiKey, prompt)
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("openai provider selected but OPENAI_API_KEY environment variable is not set")
		}
		jsonBytes, err = c.callOpenAI(ctx, apiKey, prompt)
	default:
		// Default to local Ollama
		jsonBytes, err = c.callOllama(ctx, prompt)
	}

	if err != nil {
		return nil, err
	}

	var resp aiResponseSchema
	cleanJSON := extractJSON(string(jsonBytes))
	if err := json.Unmarshal([]byte(cleanJSON), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w (raw response: %s)", err, string(jsonBytes))
	}

	rule := &model.ErrorRule{
		ID:                 "ai-generated",
		Error:              resp.Error,
		Category:           "AI-Analysis",
		Meaning:            resp.Meaning,
		LikelyCause:        resp.LikelyCause,
		PossibleCauses:     resp.PossibleCauses,
		DiagnosticSteps:    resp.DiagnosticSteps,
		PotentialSolutions: resp.PotentialSolutions,
		Warnings:           resp.Warnings,
	}

	return &model.MatchResult{
		Rule:               rule,
		MatchType:          model.MatchAI,
		ConfidenceScore:    0.80,
		ConfidenceLevel:    "MEDIUM",
		MatchedInput:       input,
		ConfirmedFromInput: []string{"Processed via optional AI diagnostic query"},
		LikelyCause:        resp.LikelyCause,
		PossibleCauses:     resp.PossibleCauses,
		DiagnosticSteps:    resp.DiagnosticSteps,
		PotentialSolutions: resp.PotentialSolutions,
		Warnings:           resp.Warnings,
		Source:             "ai",
	}, nil
}

func (c *Client) callOllama(ctx context.Context, prompt string) ([]byte, error) {
	baseURL := c.cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	modelName := c.cfg.Model
	if modelName == "" {
		modelName = "llama3"
	}

	payload := map[string]any{
		"model":  modelName,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to Ollama at %s: %w. Ensure Ollama is running or configure GEMINI_API_KEY", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}
	return []byte(ollamaResp.Response), nil
}

func (c *Client) callGemini(ctx context.Context, apiKey, prompt string) ([]byte, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
	payload := map[string]any{
		"contents": []any{
			map[string]any{
				"parts": []any{
					map[string]string{"text": prompt},
				},
			},
		},
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini API returned empty response")
	}
	return []byte(geminiResp.Candidates[0].Content.Parts[0].Text), nil
}

func (c *Client) callOpenAI(ctx context.Context, apiKey, prompt string) ([]byte, error) {
	url := "https://api.openai.com/v1/chat/completions"
	modelName := c.cfg.Model
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}
	payload := map[string]any{
		"model": modelName,
		"messages": []any{
			map[string]string{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, err
	}
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("openai API returned no choices")
	}
	return []byte(openAIResp.Choices[0].Message.Content), nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip markdown ```json ... ```
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			s = strings.Join(lines, "\n")
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}
