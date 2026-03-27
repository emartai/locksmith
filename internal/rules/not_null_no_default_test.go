package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestNotNullNoDefaultRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "set not null flags",
			sql:       "ALTER TABLE users ALTER COLUMN email SET NOT NULL;",
			wantCount: 1,
		},
		{
			name:      "drop not null does not flag",
			sql:       "ALTER TABLE users ALTER COLUMN email DROP NOT NULL;",
			wantCount: 0,
		},
		{
			name:      "ignore comment skips finding",
			sql:       "-- locksmith:ignore NOT_NULL_NO_DEFAULT\nALTER TABLE users ALTER COLUMN email SET NOT NULL;",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.NotNullNoDefaultRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestNotNullNoDefaultRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE users ALTER COLUMN email SET NOT NULL;", &rules.NotNullNoDefaultRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "NOT_NULL_NO_DEFAULT" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "NOT_NULL_NO_DEFAULT")
	}
	if finding.Summary != "SET NOT NULL on existing column" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "SET NOT NULL on existing column")
	}
	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}
}
