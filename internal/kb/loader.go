package kb

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/model"
	"gopkg.in/yaml.v3"
)

//go:embed data/*.yaml
var embeddedFS embed.FS

// KnowledgeBase holds all registered error rules.
type KnowledgeBase struct {
	Rules []*model.ErrorRule
}

// RuleFile represents the YAML file structure.
type RuleFile struct {
	Rules []*model.ErrorRule `yaml:"rules"`
}

// LoadDefault loads the embedded knowledge database rules.
func LoadDefault() (*KnowledgeBase, error) {
	kb := &KnowledgeBase{Rules: make([]*model.ErrorRule, 0)}

	entries, err := embeddedFS.ReadDir("data")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded kb directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		content, err := embeddedFS.ReadFile("data/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded file %s: %w", entry.Name(), err)
		}

		var file RuleFile
		if err := yaml.Unmarshal(content, &file); err != nil {
			return nil, fmt.Errorf("failed to parse YAML %s: %w", entry.Name(), err)
		}

		for _, rule := range file.Rules {
			if err := validateRule(rule); err != nil {
				return nil, fmt.Errorf("invalid rule %q in %s: %w", rule.ID, entry.Name(), err)
			}
			kb.Rules = append(kb.Rules, rule)
		}
	}

	return kb, nil
}

// LoadCustomDir loads additional rules from a custom directory on disk.
func (kb *KnowledgeBase) LoadCustomDir(dir string) error {
	if dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) || !info.IsDir() {
		return nil // directory does not exist, ignore silently
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read custom rules directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read custom rule file %s: %w", filePath, err)
		}

		var file RuleFile
		if err := yaml.Unmarshal(content, &file); err != nil {
			return fmt.Errorf("failed to parse custom YAML %s: %w", filePath, err)
		}

		for _, rule := range file.Rules {
			if err := validateRule(rule); err != nil {
				return fmt.Errorf("invalid custom rule %q in %s: %w", rule.ID, filePath, err)
			}
			// Prepend custom rules so they take priority
			kb.Rules = append([]*model.ErrorRule{rule}, kb.Rules...)
		}
	}

	return nil
}

func validateRule(r *model.ErrorRule) error {
	if r.ID == "" {
		return fmt.Errorf("rule missing id")
	}
	if r.Error == "" {
		return fmt.Errorf("rule %q missing error name", r.ID)
	}
	if r.Category == "" {
		return fmt.Errorf("rule %q missing category", r.ID)
	}
	if r.Meaning == "" {
		return fmt.Errorf("rule %q missing meaning", r.ID)
	}
	for _, p := range r.Patterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid regex pattern %q in rule %s: %w", p, r.ID, err)
		}
	}
	return nil
}
