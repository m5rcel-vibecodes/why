package logscan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/formatter"
)

// FormatTerminalReport renders the log scan report for the terminal.
func FormatTerminalReport(report *LogReport, styler *formatter.Styler) string {
	var sb strings.Builder

	sb.WriteString(styler.Bold(styler.Cyan("LOG INSPECTION REPORT\n")))
	sb.WriteString(styler.Dim(fmt.Sprintf("Source: %s\n\n", report.Source)))

	sb.WriteString(styler.Bold("Summary:\n"))
	sb.WriteString(fmt.Sprintf("  • Scanned: %d lines\n", report.TotalLines))
	if report.TotalErrors == 0 {
		sb.WriteString(styler.Green("  • No known error patterns detected in log file.\n\n"))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("  • Total errors detected: %s\n", styler.Red(fmt.Sprintf("%d occurrences", report.TotalErrors))))
	sb.WriteString(fmt.Sprintf("  • Distinct error types:  %s\n\n", styler.Yellow(fmt.Sprintf("%d", report.DistinctErrors))))

	sb.WriteString(styler.Bold(styler.Cyan("ERRORS DETECTED (ORDERED BY FREQUENCY):\n\n")))

	for i, f := range report.Findings {
		sb.WriteString(fmt.Sprintf("%s %s %s\n",
			styler.Bold(fmt.Sprintf("[%d]", i+1)),
			styler.Bold(styler.Red(fmt.Sprintf("%dx", f.Count))),
			styler.Bold(fmt.Sprintf("%s (%s)", f.Error, f.Category)),
		))

		// Line numbers preview
		lineStr := formatLineNumbers(f.LineNumbers)
		sb.WriteString(fmt.Sprintf("    Lines: %s\n", styler.Dim(lineStr)))

		if f.SampleLine != "" {
			trimmedSample := f.SampleLine
			if len(trimmedSample) > 120 {
				trimmedSample = trimmedSample[:117] + "..."
			}
			sb.WriteString(fmt.Sprintf("    Sample: %s\n", styler.Dim(trimmedSample)))
		}

		if f.LikelyCause != "" {
			sb.WriteString(fmt.Sprintf("    %s %s\n", styler.Yellow("Likely Cause:"), f.LikelyCause))
		}

		if len(f.DiagnosticSteps) > 0 {
			sb.WriteString(fmt.Sprintf("    %s\n", styler.Cyan("Suggested Checks:")))
			for _, step := range f.DiagnosticSteps {
				cmdStr := styler.Bold(step.Command)
				if step.Destructive {
					cmdStr += " " + styler.Red("[DESTRUCTIVE]")
				}
				sb.WriteString(fmt.Sprintf("      • %s\n", cmdStr))
			}
		}

		if len(f.Warnings) > 0 {
			sb.WriteString(fmt.Sprintf("    %s %s\n", styler.Red("Caution:"), f.Warnings[0]))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatJSONReport outputs the log report as indented JSON.
func FormatJSONReport(report *LogReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func formatLineNumbers(lines []int) string {
	if len(lines) == 0 {
		return "none"
	}
	var strs []string
	for i, l := range lines {
		if i >= 10 {
			strs = append(strs, fmt.Sprintf("... (+%d more)", len(lines)-10))
			break
		}
		strs = append(strs, fmt.Sprintf("%d", l))
	}
	return strings.Join(strs, ", ")
}
