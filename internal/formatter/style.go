package formatter

import (
	"os"
	"strings"
)

// Styler manages ANSI color output while respecting NO_COLOR and TTY status.
type Styler struct {
	enabled bool
}

// ColorMode determines color output preference.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// NewStyler creates a Styler based on mode and environment.
func NewStyler(mode ColorMode) *Styler {
	switch mode {
	case ColorNever:
		return &Styler{enabled: false}
	case ColorAlways:
		return &Styler{enabled: true}
	default:
		// Check NO_COLOR environment variable (https://no-color.org)
		if _, exists := os.LookupEnv("NO_COLOR"); exists {
			return &Styler{enabled: false}
		}
		// Check TERM
		term := os.Getenv("TERM")
		if term == "dumb" || term == "" {
			return &Styler{enabled: false}
		}
		// Check if stdout is a character device (terminal)
		fi, err := os.Stdout.Stat()
		if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			return &Styler{enabled: true}
		}
		return &Styler{enabled: false}
	}
}

func (s *Styler) Bold(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[1m" + text + "\033[0m"
}

func (s *Styler) Red(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[31m" + text + "\033[0m"
}

func (s *Styler) Green(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[32m" + text + "\033[0m"
}

func (s *Styler) Yellow(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[33m" + text + "\033[0m"
}

func (s *Styler) Blue(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[34m" + text + "\033[0m"
}

func (s *Styler) Cyan(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[36m" + text + "\033[0m"
}

func (s *Styler) Dim(text string) string {
	if !s.enabled {
		return text
	}
	return "\033[2m" + text + "\033[0m"
}

func (s *Styler) Indent(text string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			lines[i] = indent + l
		}
	}
	return strings.Join(lines, "\n")
}
