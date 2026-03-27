package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestAlterColumnTypeRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "alter column type flags",
			sql:       "ALTER TABLE users ALTER COLUMN age TYPE BIGINT;",
			wantCount: 1,
		},
		{
			name:      "add column does not flag",
			sql:       "ALTER TABLE users ADD COLUMN age BIGINT;",
			wantCount: 0,
		},
		{
			name:      "ignore comment skips finding",
			sql:       "-- locksmith:ignore ALTER_COLUMN_TYPE\nALTER TABLE users ALTER COLUMN age TYPE BIGINT;",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.AlterColumnTypeRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestAlterColumnTypeRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE users ALTER COLUMN age TYPE BIGINT;", &rules.AlterColumnTypeRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "ALTER_COLUMN_TYPE" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "ALTER_COLUMN_TYPE")
	}
	if finding.Summary != "ALTER COLUMN TYPE" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "ALTER COLUMN TYPE")
	}
	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}
}
