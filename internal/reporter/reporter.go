package reporter

import (
	"fmt"
	"io"
	"sort"

	"github.com/fatih/color"

	"github.com/emartai/locksmith/internal/rules"
)

const severityWidth = 9

// Reporter formats findings for terminal output.
type Reporter struct {
	writer    io.Writer
	dangerous *color.Color
	warning   *color.Color
	success   *color.Color
}

// New creates a reporter that writes to the provided writer.
func New(writer io.Writer, useColor bool) *Reporter {
	plain := color.New()
	if writer == nil {
		writer = io.Discard
	}

	reporter := &Reporter{
		writer:    writer,
		dangerous: plain,
		warning:   plain,
		success:   plain,
	}

	if useColor {
		reporter.dangerous = color.New(color.FgRed, color.Bold)
		reporter.warning = color.New(color.FgYellow, color.Bold)
		reporter.success = color.New(color.FgGreen, color.Bold)
	}

	return reporter
}

// Print writes finding details for a single file.
func (r *Reporter) Print(findings []rules.Finding, filePath string) {
	if len(findings) == 0 {
		fmt.Fprintln(r.writer, r.success.Sprintf("✅ PASSED - no issues found"))
		return
	}

	for i, finding := range findings {
		if i > 0 {
			fmt.Fprintln(r.writer)
		}

		label := fmt.Sprintf("%-*s", severityWidth, finding.Severity)
		header := fmt.Sprintf("%s %s Line %d", emojiFor(finding.Severity), label, finding.Line)
		fmt.Fprintln(r.writer, colorizeHeader(r, finding.Severity, header))
		fmt.Fprintf(r.writer, "   %s\n", finding.Summary)
		fmt.Fprintln(r.writer, "   ")
		fmt.Fprintf(r.writer, "   Why:   %s\n", finding.Why)
		if finding.LockType != "" {
			fmt.Fprintf(r.writer, "   Lock:  %s\n", finding.LockType)
		}
		fmt.Fprintf(r.writer, "   Fix:   %s\n", finding.Fix)
	}

	fmt.Fprintln(r.writer)
	fmt.Fprintln(r.writer)
	fmt.Fprintln(r.writer, summaryLine(findings))
}

// PrintSummary writes an aggregate directory-level summary.
func (r *Reporter) PrintSummary(allFindings map[string][]rules.Finding) {
	files := make([]string, 0, len(allFindings))
	for file := range allFindings {
		files = append(files, file)
	}
	sort.Strings(files)

	filesWithIssues := 0
	dangerousCount := 0
	warningCount := 0
	for _, file := range files {
		if len(allFindings[file]) > 0 {
			filesWithIssues++
		}
		for _, finding := range allFindings[file] {
			switch finding.Severity {
			case rules.SeverityDangerous:
				dangerousCount++
			case rules.SeverityWarning:
				warningCount++
			}
		}
	}

	fmt.Fprintf(
		r.writer,
		"Checked %d files. %d files with issues (%d dangerous, %d warning).\n",
		len(files),
		filesWithIssues,
		dangerousCount,
		warningCount,
	)
}

func colorizeHeader(r *Reporter, severity rules.Severity, header string) string {
	switch severity {
	case rules.SeverityDangerous:
		return r.dangerous.Sprint(header)
	case rules.SeverityWarning:
		return r.warning.Sprint(header)
	default:
		return header
	}
}

func emojiFor(severity rules.Severity) string {
	switch severity {
	case rules.SeverityDangerous:
		return "❌"
	case rules.SeverityWarning:
		return "⚠️"
	default:
		return "✅"
	}
}

func summaryLine(findings []rules.Finding) string {
	dangerousCount := 0
	warningCount := 0
	for _, finding := range findings {
		switch finding.Severity {
		case rules.SeverityDangerous:
			dangerousCount++
		case rules.SeverityWarning:
			warningCount++
		}
	}

	if dangerousCount > 0 {
		return fmt.Sprintf("%d issues found. Migration blocked.", len(findings))
	}

	return fmt.Sprintf("%d warnings found.", warningCount)
}
