package rules

import "testing"

func TestSeverityConstants(t *testing.T) {
	if SeverityDangerous != "DANGEROUS" {
		t.Fatalf("SeverityDangerous = %q, want %q", SeverityDangerous, "DANGEROUS")
	}

	if SeverityWarning != "WARNING" {
		t.Fatalf("SeverityWarning = %q, want %q", SeverityWarning, "WARNING")
	}

	if SeverityInfo != "INFO" {
		t.Fatalf("SeverityInfo = %q, want %q", SeverityInfo, "INFO")
	}
}

func TestFindingConstruction(t *testing.T) {
	finding := Finding{
		RuleID:   "ADD_COLUMN_DEFAULT",
		Severity: SeverityDangerous,
		Line:     4,
		FilePath: "testdata/migrations/dangerous/001_add_column_with_default.sql",
		Summary:  "ADD COLUMN with NOT NULL and DEFAULT",
		Why:      "On Postgres 10 and earlier, this rewrites the entire table.",
		LockType: "ACCESS EXCLUSIVE",
		Fix:      "Add column as nullable first, backfill in batches, then add NOT NULL constraint separately.",
	}

	if finding.RuleID != "ADD_COLUMN_DEFAULT" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "ADD_COLUMN_DEFAULT")
	}

	if finding.Severity != SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", finding.Severity, SeverityDangerous)
	}

	if finding.Line != 4 {
		t.Fatalf("Line = %d, want %d", finding.Line, 4)
	}

	if finding.FilePath != "testdata/migrations/dangerous/001_add_column_with_default.sql" {
		t.Fatalf("FilePath = %q, want %q", finding.FilePath, "testdata/migrations/dangerous/001_add_column_with_default.sql")
	}

	if finding.Summary != "ADD COLUMN with NOT NULL and DEFAULT" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "ADD COLUMN with NOT NULL and DEFAULT")
	}

	if finding.Why != "On Postgres 10 and earlier, this rewrites the entire table." {
		t.Fatalf("Why = %q, want %q", finding.Why, "On Postgres 10 and earlier, this rewrites the entire table.")
	}

	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}

	if finding.Fix != "Add column as nullable first, backfill in batches, then add NOT NULL constraint separately." {
		t.Fatalf("Fix = %q, want %q", finding.Fix, "Add column as nullable first, backfill in batches, then add NOT NULL constraint separately.")
	}
}
