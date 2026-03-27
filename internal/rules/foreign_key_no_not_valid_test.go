package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestForeignKeyNoNotValidRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "foreign key without not valid flags",
			sql:       "ALTER TABLE orders ADD CONSTRAINT fk FOREIGN KEY (user_id) REFERENCES users(id);",
			wantCount: 1,
		},
		{
			name:      "foreign key with not valid does not flag",
			sql:       "ALTER TABLE orders ADD CONSTRAINT fk FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;",
			wantCount: 0,
		},
		{
			name:      "check constraint does not flag",
			sql:       "ALTER TABLE orders ADD CONSTRAINT chk CHECK (amount > 0);",
			wantCount: 0,
		},
		{
			name:      "ignore comment skips finding",
			sql:       "-- locksmith:ignore FOREIGN_KEY_NO_NOT_VALID\nALTER TABLE orders ADD CONSTRAINT fk FOREIGN KEY (user_id) REFERENCES users(id);",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.ForeignKeyNoNotValidRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestForeignKeyNoNotValidRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE orders ADD CONSTRAINT fk FOREIGN KEY (user_id) REFERENCES users(id);", &rules.ForeignKeyNoNotValidRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "FOREIGN_KEY_NO_NOT_VALID" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "FOREIGN_KEY_NO_NOT_VALID")
	}

	if finding.Severity != rules.SeverityDangerous {
		t.Fatalf("Severity = %q, want %q", finding.Severity, rules.SeverityDangerous)
	}

	if finding.Summary != "ADD FOREIGN KEY without NOT VALID" {
		t.Fatalf("Summary = %q, want %q", finding.Summary, "ADD FOREIGN KEY without NOT VALID")
	}

	if finding.Why != "Validates every existing row in the table immediately. Holds SHARE ROW EXCLUSIVE lock blocking all writes during validation." {
		t.Fatalf("Why = %q", finding.Why)
	}

	if finding.LockType != "SHARE ROW EXCLUSIVE" {
		t.Fatalf("LockType = %q, want %q", finding.LockType, "SHARE ROW EXCLUSIVE")
	}

	if finding.Fix != "Add the constraint with NOT VALID first, then run VALIDATE CONSTRAINT in a separate migration." {
		t.Fatalf("Fix = %q", finding.Fix)
	}
}
