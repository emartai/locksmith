package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestAddColumnDefaultRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "nullable without default does not flag",
			sql:       "ALTER TABLE t ADD COLUMN x TEXT;",
			wantCount: 0,
		},
		{
			name:      "default but nullable does not flag",
			sql:       "ALTER TABLE t ADD COLUMN x TEXT DEFAULT 'val';",
			wantCount: 0,
		},
		{
			name:      "not null with default flags",
			sql:       "ALTER TABLE t ADD COLUMN x TEXT NOT NULL DEFAULT 'val';",
			wantCount: 1,
		},
		{
			name:      "not null without default does not flag",
			sql:       "ALTER TABLE t ADD COLUMN x TEXT NOT NULL;",
			wantCount: 0,
		},
		{
			name:      "ignore comment skips finding",
			sql:       "-- locksmith:ignore ADD_COLUMN_DEFAULT\nALTER TABLE t ADD COLUMN x TEXT NOT NULL DEFAULT 'val';",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.AddColumnDefaultRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestAddColumnDefaultRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE t ADD COLUMN x TEXT NOT NULL DEFAULT 'val';", &rules.AddColumnDefaultRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "ADD_COLUMN_DEFAULT" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "ADD_COLUMN_DEFAULT")
	}

	if finding.Severity != rules.SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", finding.Severity, rules.SeverityDangerous)
	}

	if finding.Summary != "ADD COLUMN with NOT NULL and DEFAULT" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "ADD COLUMN with NOT NULL and DEFAULT")
	}

	if finding.Why != "On Postgres 10 and earlier, this rewrites the entire table. An ACCESS EXCLUSIVE lock is held for the full duration." {
		t.Fatalf("Why = %q", finding.Why)
	}

	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}

	if finding.Fix != "Add column as nullable first, backfill in batches, then add NOT NULL constraint separately." {
		t.Fatalf("Fix = %q", finding.Fix)
	}
}
