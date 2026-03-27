package rules

import "testing"

type overrideSource map[string]string

func (o overrideSource) RuleOverrides() map[string]string {
	return map[string]string(o)
}

func TestDefaultEngineWithOverridesAppliesConfig(t *testing.T) {
	engine := DefaultEngineWithOverrides(overrideSource{
		"MISSING_FK_INDEX":     "error",
		"MISSING_LOCK_TIMEOUT": "ignore",
	})

	result := ParseResult{
		FilePath: "migration.sql",
		Statements: []Statement{
			{
				Raw:  `ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;`,
				Line: 1,
				Node: mustParseNode(t, `ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;`),
			},
		},
	}

	findings := engine.Run(result)
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	if findings[0].RuleID != "MISSING_FK_INDEX" {
		t.Fatalf("findings[0].RuleID = %q, want %q", findings[0].RuleID, "MISSING_FK_INDEX")
	}
	if findings[0].Severity != SeverityDangerous {
		t.Fatalf("findings[0].Severity = %q, want %q", findings[0].Severity, SeverityDangerous)
	}
}
