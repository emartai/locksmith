package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestDropObjectRule(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantCount   int
		wantSummary string
		wantRuleID  string
	}{
		{
			name:        "drop table flags",
			sql:         "DROP TABLE users;",
			wantCount:   1,
			wantSummary: "DROP TABLE",
			wantRuleID:  "DROP_OBJECT",
		},
		{
			name:        "drop table if exists flags",
			sql:         "DROP TABLE IF EXISTS users;",
			wantCount:   1,
			wantSummary: "DROP TABLE",
			wantRuleID:  "DROP_OBJECT",
		},
		{
			name:        "drop column flags",
			sql:         "ALTER TABLE users DROP COLUMN email;",
			wantCount:   1,
			wantSummary: "DROP COLUMN",
			wantRuleID:  "DROP_OBJECT",
		},
		{
			name:        "drop column if exists flags",
			sql:         "ALTER TABLE users DROP COLUMN IF EXISTS email;",
			wantCount:   1,
			wantSummary: "DROP COLUMN",
			wantRuleID:  "DROP_OBJECT",
		},
		{
			name:      "add column does not flag",
			sql:       "ALTER TABLE users ADD COLUMN email TEXT;",
			wantCount: 0,
		},
		{
			name:      "drop index does not flag",
			sql:       "DROP INDEX idx_users_email;",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.DropObjectRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}

			if test.wantCount == 0 {
				return
			}

			finding := findings[0]
			if finding.RuleID != test.wantRuleID {
				t.Fatalf("RuleID = %q, want %q", finding.RuleID, test.wantRuleID)
			}

			if finding.Summary != test.wantSummary {
				t.Fatalf("Summary = %q, want %q", finding.Summary, test.wantSummary)
			}
		})
	}
}

func TestDropObjectRuleDropTableFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "DROP TABLE users;", &rules.DropObjectRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.Severity != rules.SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", finding.Severity, rules.SeverityDangerous)
	}

	if finding.Why != "Acquires ACCESS EXCLUSIVE lock and permanently deletes table data. Application code referencing this table will immediately error." {
		t.Fatalf("Why = %q", finding.Why)
	}

	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}

	if finding.Fix != "Ensure all application code has been deployed without references to this table before dropping it." {
		t.Fatalf("Fix = %q", finding.Fix)
	}
}

func TestDropObjectRuleDropColumnFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE users DROP COLUMN email;", &rules.DropObjectRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.Severity != rules.SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", finding.Severity, rules.SeverityDangerous)
	}

	if finding.Why != "Acquires ACCESS EXCLUSIVE lock. If application code still reads this column, it will immediately error." {
		t.Fatalf("Why = %q", finding.Why)
	}

	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}

	if finding.Fix != "Deploy application code that no longer references this column, verify no reads in prod, then drop." {
		t.Fatalf("Fix = %q", finding.Fix)
	}
}
