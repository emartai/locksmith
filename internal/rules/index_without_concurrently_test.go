package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestIndexWithoutConcurrentlyRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "create index flags",
			sql:       "CREATE INDEX idx ON t(col);",
			wantCount: 1,
		},
		{
			name:      "create unique index flags",
			sql:       "CREATE UNIQUE INDEX idx ON t(col);",
			wantCount: 1,
		},
		{
			name:      "create index concurrently does not flag",
			sql:       "CREATE INDEX CONCURRENTLY idx ON t(col);",
			wantCount: 0,
		},
		{
			name:      "create unique index concurrently does not flag",
			sql:       "CREATE UNIQUE INDEX CONCURRENTLY idx ON t(col);",
			wantCount: 0,
		},
		{
			name:      "ignore comment skips finding",
			sql:       "-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY\nCREATE INDEX idx ON t(col);",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.IndexWithoutConcurrentlyRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestIndexWithoutConcurrentlyRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "CREATE INDEX idx_orders_user_id ON orders(user_id);", &rules.IndexWithoutConcurrentlyRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "INDEX_WITHOUT_CONCURRENTLY" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "INDEX_WITHOUT_CONCURRENTLY")
	}

	if finding.Severity != rules.SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", finding.Severity, rules.SeverityDangerous)
	}

	if finding.Summary != "CREATE INDEX without CONCURRENTLY" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "CREATE INDEX without CONCURRENTLY")
	}

	if finding.Why != "Holds a SHARE lock on the table for the full duration of the index build, blocking all writes." {
		t.Fatalf("Why = %q", finding.Why)
	}

	if finding.LockType != "SHARE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "SHARE")
	}

	if finding.Fix != "Use CREATE INDEX CONCURRENTLY. Note: CONCURRENTLY cannot run inside a transaction block." {
		t.Fatalf("Fix = %q", finding.Fix)
	}
}
