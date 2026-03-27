package reporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestWriteGitHubSummaryWithFindings(t *testing.T) {
	summaryPath := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv(gitHubStepSummaryEnv, summaryPath)

	err := WriteGitHubSummary([]JSONOutput{
		{
			File: "migrations/001.sql",
			Findings: []JSONFinding{
				{
					RuleID:   "ADD_COLUMN_DEFAULT",
					Severity: rules.SeverityDangerous,
					Line:     4,
					Summary:  "ADD COLUMN with NOT NULL and DEFAULT",
				},
			},
		},
		{
			File: "migrations/002.sql",
			Findings: []JSONFinding{
				{
					RuleID:   "INDEX_WITHOUT_CONCURRENTLY",
					Severity: rules.SeverityDangerous,
					Line:     1,
					Summary:  "CREATE INDEX without CONCURRENTLY",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteGitHubSummary() error = %v", err)
	}

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	got := string(data)
	wantParts := []string{
		"## Locksmith Migration Safety Report\n",
		"| File | Rule | Severity | Line | Summary |\n",
		"| migrations/001.sql | ADD_COLUMN_DEFAULT | ❌ DANGEROUS | 4 | ADD COLUMN with NOT NULL and DEFAULT |\n",
		"| migrations/002.sql | INDEX_WITHOUT_CONCURRENTLY | ❌ DANGEROUS | 1 | CREATE INDEX without CONCURRENTLY |\n",
		"**2 dangerous issues found across 2 files. Merge blocked.**\n",
	}
	for _, want := range wantParts {
		if !strings.Contains(got, want) {
			t.Fatalf("summary output missing %q in %q", want, got)
		}
	}
}

func TestWriteGitHubSummaryClean(t *testing.T) {
	summaryPath := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv(gitHubStepSummaryEnv, summaryPath)

	err := WriteGitHubSummary([]JSONOutput{
		{
			File:     "migrations/001.sql",
			Passed:   true,
			Findings: nil,
		},
	})
	if err != nil {
		t.Fatalf("WriteGitHubSummary() error = %v", err)
	}

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	got := string(data)
	want := "## Locksmith Migration Safety Report\n✅ All migration files passed. No dangerous operations detected.\n"
	if got != want {
		t.Fatalf("summary output = %q, want %q", got, want)
	}
}
