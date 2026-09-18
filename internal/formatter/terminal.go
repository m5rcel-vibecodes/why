package formatter

import (
	"fmt"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/model"
)

// FormatTerminal renders a MatchResult in human-readable terminal format.
func FormatTerminal(res *model.MatchResult, styler *Styler) string {
	if res == nil {
		return styler.Yellow("NO EXPLANATION FOUND\n\n") +
			"The input did not match any known error patterns in the local knowledge base.\n" +
			styler.Dim("Try checking the command syntax, or pass --ai to query an AI model.\n")
	}

	var sb strings.Builder

	// Header
	sb.WriteString(styler.Bold(styler.Cyan("WHY DID THIS HAPPEN?\n\n")))

	// AI Notice if applicable
	if res.Source == "ai" {
		sb.WriteString(styler.Yellow(styler.Bold("[AI GENERATED EXPLANATION]")) +
			styler.Dim(" (Generated via AI model; verify all suggestions before executing)\n\n"))
	}

	// Error Title
	sb.WriteString(styler.Bold("Error:\n"))
	if res.Rule != nil && res.Rule.Category != "" {
		sb.WriteString(fmt.Sprintf("  %s %s\n\n", res.Rule.Error, styler.Dim("("+res.Rule.Category+")")))
	} else if res.Rule != nil {
		sb.WriteString(fmt.Sprintf("  %s\n\n", res.Rule.Error))
	}

	// Meaning
	if res.Rule != nil && res.Rule.Meaning != "" {
		sb.WriteString(styler.Bold("Usually means:\n"))
		sb.WriteString(styler.Indent(res.Rule.Meaning, 2) + "\n\n")
	}

	// CONFIRMED FROM INPUT
	if len(res.ConfirmedFromInput) > 0 {
		sb.WriteString(styler.Bold(styler.Green("CONFIRMED FROM INPUT:\n")))
		for _, conf := range res.ConfirmedFromInput {
			sb.WriteString(fmt.Sprintf("  • %s\n", conf))
		}
		sb.WriteString("\n")
	}

	// LIKELY CAUSE
	if res.LikelyCause != "" {
		sb.WriteString(styler.Bold(styler.Yellow("LIKELY CAUSE:\n")))
		sb.WriteString(styler.Indent(res.LikelyCause, 2) + "\n\n")
	}

	// POSSIBLE CAUSES
	if len(res.PossibleCauses) > 0 {
		sb.WriteString(styler.Bold("POSSIBLE CAUSES:\n"))
		for _, cause := range res.PossibleCauses {
			sb.WriteString(fmt.Sprintf("  • %s\n", cause))
		}
		sb.WriteString("\n")
	}

	// NEXT DIAGNOSTIC STEP / CHECK
	if len(res.DiagnosticSteps) > 0 {
		sb.WriteString(styler.Bold(styler.Cyan("CHECK / NEXT DIAGNOSTIC STEP:\n\n")))
		for _, step := range res.DiagnosticSteps {
			cmdStr := styler.Bold(step.Command)
			if step.Destructive {
				cmdStr += " " + styler.Red("[DESTRUCTIVE]")
			}
			sb.WriteString(fmt.Sprintf("  %s\n", cmdStr))
			if step.Description != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", styler.Dim(step.Description)))
			}
			sb.WriteString("\n")
		}
	}

	// POTENTIAL SOLUTIONS
	if len(res.PotentialSolutions) > 0 {
		sb.WriteString(styler.Bold("POTENTIAL SOLUTIONS:\n\n"))
		for _, sol := range res.PotentialSolutions {
			sb.WriteString(fmt.Sprintf("  • %s\n", sol.Description))
			if sol.Command != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", styler.Bold(sol.Command)))
			}
			if sol.Warning != "" {
				sb.WriteString(fmt.Sprintf("    %s %s\n", styler.Red("Caution:"), sol.Warning))
			}
			sb.WriteString("\n")
		}
	}

	// WARNINGS
	if len(res.Warnings) > 0 {
		sb.WriteString(styler.Bold(styler.Red("WARNING:\n")))
		for _, warn := range res.Warnings {
			sb.WriteString(fmt.Sprintf("  %s\n", warn))
		}
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}
