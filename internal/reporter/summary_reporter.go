package reporter

import (
	"fmt"
	"os"
	"strings"

	"github.com/emartai/locksmith/internal/rules"
)

const gitHubStepSummaryEnv = "GITHUB_STEP_SUMMARY"

// WriteGitHubSummary appends a Markdown report to the GitHub Actions job summary.
// It does nothing when GITHUB_STEP_SUMMARY is not set.
func WriteGitHubSummary(outputs []JSONOutput) error {
	summaryPath := strings.TrimSpace(os.Getenv(gitHubStepSummaryEnv))
	if summaryPath == "" {
		return nil
	}

	file, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open github step summary: %w", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, "## Locksmith Migration Safety Report"); err != nil {
		return fmt.Errorf("write github step summary header: %w", err)
	}
	if len(flattenFindings(outputs)) == 0 {
		if _, err := fmt.Fprintln(file, "✅ All migration files passed. No dangerous operations detected."); err != nil {
			return fmt.Errorf("write github step summary clean message: %w", err)
		}
		return nil
	}

	if _, err := fmt.Fprintln(file); err != nil {
		return fmt.Errorf("write github step summary spacing: %w", err)
	}
	if _, err := fmt.Fprintln(file, "| File | Rule | Severity | Line | Summary |"); err != nil {
		return fmt.Errorf("write github step summary table header: %w", err)
	}
	if _, err := fmt.Fprintln(file, "|------|------|----------|------|---------|"); err != nil {
		return fmt.Errorf("write github step summary table divider: %w", err)
	}

	for _, row := range flattenFindings(outputs) {
		if _, err := fmt.Fprintf(
			file,
			"| %s | %s | %s %s | %d | %s |\n",
			escapeMarkdownCell(row.file),
			escapeMarkdownCell(row.finding.RuleID),
			emojiFor(row.finding.Severity),
			row.finding.Severity,
			row.finding.Line,
			escapeMarkdownCell(row.finding.Summary),
		); err != nil {
			return fmt.Errorf("write github step summary row: %w", err)
		}
	}

	dangerousCount, warningCount, filesWithIssues := summarizeOutputs(outputs)
	if _, err := fmt.Fprintln(file); err != nil {
		return fmt.Errorf("write github step summary spacing: %w", err)
	}
	if dangerousCount > 0 {
		_, err = fmt.Fprintf(
			file,
			"**%d dangerous issues found across %d files. Merge blocked.**\n",
			dangerousCount,
			filesWithIssues,
		)
	} else {
		_, err = fmt.Fprintf(
			file,
			"**%d warning issues found across %d files. Review recommended.**\n",
			warningCount,
			filesWithIssues,
		)
	}
	if err != nil {
		return fmt.Errorf("write github step summary footer: %w", err)
	}

	return nil
}

type summaryRow struct {
	file    string
	finding JSONFinding
}

func flattenFindings(outputs []JSONOutput) []summaryRow {
	rows := make([]summaryRow, 0)
	for _, output := range outputs {
		for _, finding := range output.Findings {
			rows = append(rows, summaryRow{
				file:    output.File,
				finding: finding,
			})
		}
	}
	return rows
}

func summarizeOutputs(outputs []JSONOutput) (dangerousCount int, warningCount int, filesWithIssues int) {
	for _, output := range outputs {
		if len(output.Findings) > 0 {
			filesWithIssues++
		}
		for _, finding := range output.Findings {
			switch finding.Severity {
			case rules.SeverityDangerous:
				dangerousCount++
			case rules.SeverityWarning:
				warningCount++
			}
		}
	}
	return dangerousCount, warningCount, filesWithIssues
}

func escapeMarkdownCell(value string) string {
	replacer := strings.NewReplacer("|", "\\|", "\n", " ", "\r", " ")
	return replacer.Replace(value)
}
