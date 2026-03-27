package rules_test

import (
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestMissingFKIndexRule(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "fk without matching index flags",
			sql:       `ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;`,
			wantCount: 1,
		},
		{
			name: "fk with matching index does not flag",
			sql: `CREATE INDEX CONCURRENTLY idx_orders_user_id ON orders(user_id);
ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;`,
			wantCount: 0,
		},
		{
			name:      "non fk statement does not flag",
			sql:       `ALTER TABLE orders ADD CONSTRAINT chk_amount CHECK (amount > 0);`,
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := runSingleRuleOnSQL(t, test.sql, &rules.MissingFKIndexRule{})
			if len(findings) != test.wantCount {
				t.Fatalf("len(findings) = %d, want %d", len(findings), test.wantCount)
			}
		})
	}
}

func TestMissingFKIndexRuleFindingDetails(t *testing.T) {
	findings := runSingleRuleOnSQL(t, "ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;", &rules.MissingFKIndexRule{})
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	finding := findings[0]
	if finding.RuleID != "MISSING_FK_INDEX" {
		t.Fatalf("RuleID = %q, want %q", finding.RuleID, "MISSING_FK_INDEX")
	}
	if finding.Severity != rules.SeverityWarning {
		t.Fatalf("Severity = %q, want %q", finding.Severity, rules.SeverityWarning)
	}
	if finding.Summary != "Foreign key column has no index" {
		t.Fatalf("Summary = %q", finding.Summary)
	}
}
