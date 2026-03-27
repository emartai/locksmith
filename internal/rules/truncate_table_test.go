package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestTruncateTableRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "truncate flags",
			sql:       "TRUNCATE sessions;",
			wantCount: 1,
		},
		{
			name:      "delete does not flag",
			sql:       "DELETE FROM sessions WHERE created_at < NOW() - INTERVAL '30 days';",
			wantCount: 0,
		},
		{
			name:      "ignore comment skips finding",
			sql:       "-- locksmith:ignore TRUNCATE_TABLE\nTRUNCATE sessions;",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.TruncateTableRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestTruncateTableRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "TRUNCATE sessions;", &rules.TruncateTableRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "TRUNCATE_TABLE" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "TRUNCATE_TABLE")
	}
	if finding.Summary != "TRUNCATE TABLE" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "TRUNCATE TABLE")
	}
	if finding.LockType != "ACCESS EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "ACCESS EXCLUSIVE")
	}
}
