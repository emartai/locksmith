package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestRenameObjectRule(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantCount   int
		wantSummary string
	}{
		{
			name:        "rename column flags",
			sql:         "ALTER TABLE users RENAME COLUMN email TO email_address;",
			wantCount:   1,
			wantSummary: "RENAME COLUMN",
		},
		{
			name:        "rename table flags",
			sql:         "ALTER TABLE users RENAME TO accounts;",
			wantCount:   1,
			wantSummary: "RENAME TABLE",
		},
		{
			name:      "rename index does not flag",
			sql:       "ALTER INDEX idx_users_email RENAME TO idx_accounts_email;",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.RenameObjectRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
			if test.wantCount == 1 && findings[0].Summary != test.wantSummary {
				t.Fatalf("Summary = %q, want %q", findings[0].Summary, test.wantSummary)
			}
		})
	}
}

func TestRenameObjectRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE users RENAME TO accounts;", &rules.RenameObjectRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	if findings[0].Severity != rules.SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", findings[0].Severity, rules.SeverityDangerous)
	}
}
