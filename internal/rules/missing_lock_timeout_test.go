package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestMissingLockTimeoutRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "dangerous migration without timeout flags",
			sql:       "ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';",
			wantCount: 2,
		},
		{
			name: "dangerous migration with lock timeout does not add warning",
			sql: `SET lock_timeout = '2s';
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';`,
			wantCount: 1,
		},
		{
			name: "timeout after dangerous statement still flags",
			sql: `ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
SET lock_timeout = '2s';`,
			wantCount: 2,
		},
		{
			name: "timeout inside transaction before dangerous statement does not flag",
			sql: `BEGIN;
SET lock_timeout = '2s';
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
COMMIT;`,
			wantCount: 1,
		},
		{
			name:      "warning only migration does not add timeout finding",
			sql:       "ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;",
			wantCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runEngineOnSQL(t, test.sql, rules.DefaultEngine())
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestMissingLockTimeoutRuleFindingDetails(t *testing.T) {
	findings := runEngineOnSQL(t, "ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';", rules.DefaultEngine())
	if len(findings) != 2 {
		t.Fatalf("len(findings) = %d, want 2", len(findings))
	}

	last := findings[1]
	if last.RuleID != "MISSING_LOCK_TIMEOUT" {
		t.Fatalf("RuleID = %q, want %q", last.RuleID, "MISSING_LOCK_TIMEOUT")
	}
	if last.Severity != rules.SeverityWarning {
		t.Fatalf("Severity = %q, want %q", last.Severity, rules.SeverityWarning)
	}
	if last.Summary != "No lock timeout set before dangerous operation" {
		t.Fatalf("Summary = %q", last.Summary)
	}
}
