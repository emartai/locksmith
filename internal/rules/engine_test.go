package rules

import "testing"

type stubRule struct {
	id       string
	severity Severity
	check    func(Statement) *Finding
}

func (r stubRule) ID() string {
	return r.id
}

func (r stubRule) Severity() Severity {
	return r.severity
}

func (r stubRule) Check(stmt Statement) *Finding {
	if r.check == nil {
		return nil
	}

	return r.check(stmt)
}

func TestEngineRunWithNoRulesReturnsNoFindings(t *testing.T) {
	engine := NewEngine(nil)

	findings := engine.Run(ParseResult{
		FilePath: "test.sql",
		Statements: []Statement{
			{Raw: "ALTER TABLE users ADD COLUMN email TEXT;", Line: 1},
		},
	})

	if len(findings) != 0 {
		t.Fatalf("len(findings) = %d, want 0", len(findings))
	}
}

func TestEngineRunReturnsFindingFromMatchingRule(t *testing.T) {
	engine := NewEngine([]Rule{
		stubRule{
			id:       "TEST_RULE",
			severity: SeverityDangerous,
			check: func(stmt Statement) *Finding {
				return &Finding{
					RuleID:   "TEST_RULE",
					Severity: SeverityDangerous,
					Line:     stmt.Line,
					Summary:  "matched",
				}
			},
		},
	})

	findings := engine.Run(ParseResult{
		FilePath: "test.sql",
		Statements: []Statement{
			{Raw: "ALTER TABLE users ADD COLUMN email TEXT;", Line: 7},
		},
	})

	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}

	if findings[0].RuleID != "TEST_RULE" {
		t.Fatalf("findings[0].RuleID = %q, want %q", findings[0].RuleID, "TEST_RULE")
	}

	if findings[0].FilePath != "test.sql" {
		t.Fatalf("findings[0].FilePath = %q, want %q", findings[0].FilePath, "test.sql")
	}
}

func TestEngineRunSkipsIgnoredRules(t *testing.T) {
	engine := NewEngine([]Rule{
		stubRule{
			id:       "SKIP_ME",
			severity: SeverityDangerous,
			check: func(stmt Statement) *Finding {
				return &Finding{
					RuleID:   "SKIP_ME",
					Severity: SeverityDangerous,
					Line:     stmt.Line,
				}
			},
		},
	})

	findings := engine.Run(ParseResult{
		FilePath: "test.sql",
		Statements: []Statement{
			{
				Raw:          "CREATE INDEX idx_users_email ON users(email);",
				Line:         3,
				IgnoredRules: []string{"SKIP_ME"},
			},
		},
	})

	if len(findings) != 0 {
		t.Fatalf("len(findings) = %d, want 0", len(findings))
	}
}

func TestEngineRunSortsFindingsByLineNumber(t *testing.T) {
	engine := NewEngine([]Rule{
		stubRule{
			id:       "FIRST",
			severity: SeverityWarning,
			check: func(stmt Statement) *Finding {
				return &Finding{
					RuleID:   "FIRST",
					Severity: SeverityWarning,
					Line:     stmt.Line,
				}
			},
		},
	})

	findings := engine.Run(ParseResult{
		FilePath: "test.sql",
		Statements: []Statement{
			{Raw: "ALTER TABLE users ADD COLUMN status TEXT;", Line: 20},
			{Raw: "CREATE INDEX idx_users_email ON users(email);", Line: 4},
			{Raw: "DROP TABLE users;", Line: 12},
		},
	})

	if len(findings) != 3 {
		t.Fatalf("len(findings) = %d, want 3", len(findings))
	}

	wantLines := []int{4, 12, 20}
	for i, wantLine := range wantLines {
		if findings[i].Line != wantLine {
			t.Fatalf("findings[%d].Line = %d, want %d", i, findings[i].Line, wantLine)
		}
	}
}
