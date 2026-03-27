package reporter

import (
	"bytes"
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestReporterPrintDangerousAndWarningFormat(t *testing.T) {
	var out bytes.Buffer
	reporter := New(&out, false)

	reporter.Print([]rules.Finding{
		{
			RuleID:   "ADD_COLUMN_DEFAULT",
			Severity: rules.SeverityDangerous,
			Line:     4,
			Summary:  "ADD COLUMN with NOT NULL and DEFAULT",
			Why:      "Rewrites entire table on Postgres < 11",
			LockType: "ACCESS EXCLUSIVE - blocks all reads and writes",
			Fix:      "Add column nullable -> backfill -> add NOT NULL",
		},
		{
			RuleID:   "MISSING_FK_INDEX",
			Severity: rules.SeverityWarning,
			Line:     9,
			Summary:  "Foreign key column has no index",
			Why:      "ON DELETE CASCADE causes full table scans",
			Fix:      "Add CREATE INDEX CONCURRENTLY on the FK column",
		},
	}, "migration.sql")

	want := "" +
		"❌ DANGEROUS Line 4\n" +
		"   ADD COLUMN with NOT NULL and DEFAULT\n" +
		"   \n" +
		"   Why:   Rewrites entire table on Postgres < 11\n" +
		"   Lock:  ACCESS EXCLUSIVE - blocks all reads and writes\n" +
		"   Fix:   Add column nullable -> backfill -> add NOT NULL\n" +
		"\n" +
		"⚠️ WARNING   Line 9\n" +
		"   Foreign key column has no index\n" +
		"   \n" +
		"   Why:   ON DELETE CASCADE causes full table scans\n" +
		"   Fix:   Add CREATE INDEX CONCURRENTLY on the FK column\n" +
		"\n" +
		"\n" +
		"2 issues found. Migration blocked.\n"

	if out.String() != want {
		t.Fatalf("Print() output = %q, want %q", out.String(), want)
	}
}

func TestReporterPrintPassedFormat(t *testing.T) {
	var out bytes.Buffer
	reporter := New(&out, false)

	reporter.Print(nil, "migration.sql")

	want := "✅ PASSED - no issues found\n"
	if out.String() != want {
		t.Fatalf("Print() output = %q, want %q", out.String(), want)
	}
}

func TestReporterPrintSummary(t *testing.T) {
	var out bytes.Buffer
	reporter := New(&out, false)

	reporter.PrintSummary(map[string][]rules.Finding{
		"a.sql": {
			{Severity: rules.SeverityDangerous},
			{Severity: rules.SeverityWarning},
		},
		"b.sql": nil,
		"c.sql": {
			{Severity: rules.SeverityDangerous},
		},
	})

	want := "Checked 3 files. 2 files with issues (2 dangerous, 1 warning).\n"
	if out.String() != want {
		t.Fatalf("PrintSummary() output = %q, want %q", out.String(), want)
	}
}

func TestSummaryLineWarningOnly(t *testing.T) {
	got := summaryLine([]rules.Finding{
		{Severity: rules.SeverityWarning},
		{Severity: rules.SeverityWarning},
	})

	want := "2 warnings found."
	if got != want {
		t.Fatalf("summaryLine() = %q, want %q", got, want)
	}
}
