# Locksmith — Testing Strategy

## Philosophy

For a safety tool like Locksmith, **false positives are worse than false negatives**. An engineer who gets a wrong flag will turn the tool off. An engineer who gets a missed flag will open a GitHub issue. Test accordingly:

- Every rule must have a test that confirms it flags the right pattern
- Every rule must have a test that confirms it does NOT flag safe alternatives
- Every rule must have a test that confirms ignore comments work
- No test may rely on external state (no live database for MVP tests)

---

## Test Structure

```
locksmith/
├── internal/
│   ├── parser/
│   │   └── parser_test.go        # Unit tests for parser
│   ├── rules/
│   │   ├── types_test.go
│   │   ├── engine_test.go
│   │   ├── add_column_default_test.go
│   │   ├── index_without_concurrently_test.go
│   │   ├── foreign_key_no_not_valid_test.go
│   │   ├── drop_object_test.go
│   │   ├── alter_column_type_test.go
│   │   ├── not_null_no_default_test.go
│   │   ├── truncate_table_test.go
│   │   ├── missing_fk_index_test.go
│   │   ├── rename_object_test.go
│   │   └── missing_lock_timeout_test.go
│   ├── reporter/
│   │   └── reporter_test.go
│   ├── config/
│   │   └── config_test.go
│   └── integration_test.go       # End-to-end tests using testdata/
└── testdata/
    └── migrations/
        ├── dangerous/             # Should trigger DANGEROUS
        ├── warnings/              # Should trigger WARNING
        ├── clean/                 # Should pass all rules
        └── ignored/              # Should pass via ignore comments
```

---

## Unit Test Pattern

All rule tests use table-driven tests. This is the standard template:

```go
package rules_test

import (
    "testing"
    "github.com/emartai/locksmith/internal/parser"
    "github.com/emartai/locksmith/internal/rules"
)

func TestAddColumnDefault(t *testing.T) {
    rule := &rules.AddColumnDefaultRule{}

    tests := []struct {
        name     string
        sql      string
        wantFlag bool
        wantRule string // Expected rule ID if flagged
    }{
        {
            name:     "nullable column — safe",
            sql:      "ALTER TABLE users ADD COLUMN status TEXT",
            wantFlag: false,
        },
        {
            name:     "column with default but nullable — safe",
            sql:      "ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active'",
            wantFlag: false,
        },
        {
            name:     "NOT NULL without default — safe (different rule)",
            sql:      "ALTER TABLE users ADD COLUMN status TEXT NOT NULL",
            wantFlag: false,
        },
        {
            name:     "NOT NULL with DEFAULT — DANGEROUS",
            sql:      "ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'",
            wantFlag: true,
            wantRule: "ADD_COLUMN_DEFAULT",
        },
        {
            name:     "ignore comment present — safe",
            sql:      "-- locksmith:ignore ADD_COLUMN_DEFAULT\nALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'",
            wantFlag: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := parser.ParseSQL(tt.sql)
            if err != nil {
                t.Fatalf("parse error: %v", err)
            }
            if len(result.Statements) == 0 {
                t.Fatal("no statements parsed")
            }

            stmt := result.Statements[0]
            finding := rule.Check(stmt)

            if tt.wantFlag && finding == nil {
                t.Errorf("expected finding but got nil")
            }
            if !tt.wantFlag && finding != nil {
                t.Errorf("expected no finding but got: %+v", finding)
            }
            if tt.wantFlag && finding != nil && finding.RuleID != tt.wantRule {
                t.Errorf("expected rule %s but got %s", tt.wantRule, finding.RuleID)
            }
        })
    }
}
```

---

## Required Tests Per Rule

| Rule | Positive (should flag) | Negative (should NOT flag) | Ignore |
|------|----------------------|---------------------------|--------|
| ADD_COLUMN_DEFAULT | NOT NULL + DEFAULT | nullable + default, NOT NULL only | ✅ |
| INDEX_WITHOUT_CONCURRENTLY | standard CREATE INDEX | CONCURRENTLY | ✅ |
| FOREIGN_KEY_NO_NOT_VALID | FK without NOT VALID | FK with NOT VALID | ✅ |
| DROP_OBJECT | DROP TABLE, DROP COLUMN | DROP INDEX, ADD COLUMN | ✅ |
| ALTER_COLUMN_TYPE | type change | no type change | ✅ |
| NOT_NULL_NO_DEFAULT | SET NOT NULL | ADD COLUMN NOT NULL | ✅ |
| TRUNCATE_TABLE | TRUNCATE, TRUNCATE CASCADE | DELETE FROM | ✅ |
| MISSING_FK_INDEX | FK without index in migration | FK with index in same migration | ✅ |
| RENAME_OBJECT | RENAME COLUMN, RENAME TABLE | no rename | ✅ |
| MISSING_LOCK_TIMEOUT | dangerous op without timeout | dangerous op with SET lock_timeout | ✅ |

---

## Integration Tests

Integration tests run the full pipeline (parse → engine → reporter) against real SQL files in `testdata/`.

```go
// internal/integration_test.go
package integration_test

import (
    "os/exec"
    "testing"
)

func TestDangerousMigrations(t *testing.T) {
    tests := []struct {
        file         string
        expectedRule string
        expectedExit int
    }{
        {
            file:         "testdata/migrations/dangerous/001_add_column_with_default.sql",
            expectedRule: "ADD_COLUMN_DEFAULT",
            expectedExit: 1,
        },
        {
            file:         "testdata/migrations/dangerous/002_create_index_blocking.sql",
            expectedRule: "INDEX_WITHOUT_CONCURRENTLY",
            expectedExit: 1,
        },
        // ... all dangerous files
    }

    for _, tt := range tests {
        t.Run(tt.file, func(t *testing.T) {
            cmd := exec.Command("./bin/locksmith", "check", "--format", "json", tt.file)
            output, _ := cmd.Output()
            exitCode := cmd.ProcessState.ExitCode()

            if exitCode != tt.expectedExit {
                t.Errorf("expected exit %d got %d\noutput: %s", tt.expectedExit, exitCode, output)
            }
            // Parse JSON and verify expectedRule appears in findings
        })
    }
}

func TestCleanMigrations(t *testing.T) {
    cmd := exec.Command("./bin/locksmith", "check", "testdata/migrations/clean/")
    output, _ := cmd.CombinedOutput()
    exitCode := cmd.ProcessState.ExitCode()

    if exitCode != 0 {
        t.Errorf("expected exit 0 got %d\noutput: %s", exitCode, output)
    }
}
```

---

## Parser Tests

```go
func TestParseFile(t *testing.T) {
    t.Run("single statement", func(t *testing.T) {
        result, err := parser.ParseSQL("CREATE TABLE t (id INT);")
        assertNoError(t, err)
        assertStatementCount(t, result, 1)
    })

    t.Run("multiple statements", func(t *testing.T) {
        sql := "CREATE TABLE t (id INT);\nALTER TABLE t ADD COLUMN name TEXT;"
        result, err := parser.ParseSQL(sql)
        assertNoError(t, err)
        assertStatementCount(t, result, 2)
    })

    t.Run("ignore comment parsing", func(t *testing.T) {
        sql := "-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY\nCREATE INDEX idx ON t(col);"
        result, err := parser.ParseSQL(sql)
        assertNoError(t, err)
        assertStatementCount(t, result, 1)
        stmt := result.Statements[0]
        if !contains(stmt.IgnoredRules, "INDEX_WITHOUT_CONCURRENTLY") {
            t.Error("expected INDEX_WITHOUT_CONCURRENTLY in ignored rules")
        }
    })

    t.Run("multiple ignore rules", func(t *testing.T) {
        sql := "-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY,DROP_OBJECT\nCREATE INDEX idx ON t(col);"
        result, err := parser.ParseSQL(sql)
        assertNoError(t, err)
        stmt := result.Statements[0]
        assertContains(t, stmt.IgnoredRules, "INDEX_WITHOUT_CONCURRENTLY")
        assertContains(t, stmt.IgnoredRules, "DROP_OBJECT")
    })

    t.Run("dollar-quoted string not split", func(t *testing.T) {
        sql := `CREATE FUNCTION f() RETURNS void AS $$
            BEGIN INSERT INTO t VALUES (1); END;
        $$ LANGUAGE plpgsql;`
        result, err := parser.ParseSQL(sql)
        assertNoError(t, err)
        assertStatementCount(t, result, 1) // Must be 1, not 2
    })

    t.Run("line number accuracy", func(t *testing.T) {
        sql := "CREATE TABLE t (id INT);\n\nALTER TABLE t ADD COLUMN x TEXT;"
        result, err := parser.ParseSQL(sql)
        assertNoError(t, err)
        if result.Statements[1].Line != 3 {
            t.Errorf("expected line 3 got %d", result.Statements[1].Line)
        }
    })

    t.Run("empty file", func(t *testing.T) {
        result, err := parser.ParseSQL("")
        assertNoError(t, err)
        assertStatementCount(t, result, 0)
    })
}
```

---

## Reporter Tests

```go
func TestReporterOutput(t *testing.T) {
    t.Run("dangerous finding format", func(t *testing.T) {
        findings := []rules.Finding{
            {
                RuleID:   "ADD_COLUMN_DEFAULT",
                Severity: rules.SeverityDangerous,
                Line:     4,
                Summary:  "ADD COLUMN with NOT NULL and DEFAULT",
                Why:      "Rewrites entire table",
                LockType: "ACCESS EXCLUSIVE",
                Fix:      "Add column nullable first",
            },
        }
        
        var buf bytes.Buffer
        r := reporter.New(&buf, reporter.Options{NoColor: true})
        r.Print(findings, "test.sql")
        
        output := buf.String()
        assertContains(t, output, "DANGEROUS")
        assertContains(t, output, "Line 4")
        assertContains(t, output, "ADD COLUMN with NOT NULL and DEFAULT")
        assertContains(t, output, "Why:")
        assertContains(t, output, "Lock:")
        assertContains(t, output, "Fix:")
        assertContains(t, output, "1 issue found")
    })

    t.Run("no findings", func(t *testing.T) {
        var buf bytes.Buffer
        r := reporter.New(&buf, reporter.Options{NoColor: true})
        r.Print(nil, "test.sql")
        
        output := buf.String()
        assertContains(t, output, "PASSED")
        assertContains(t, output, "no issues found")
    })
}
```

---

## Config Tests

```go
func TestLoadConfig(t *testing.T) {
    t.Run("missing config returns defaults", func(t *testing.T) {
        cfg, err := config.LoadConfig("/nonexistent/path/locksmith.yml")
        assertNoError(t, err)
        assertNotNil(t, cfg)
        assertEmpty(t, cfg.Rules)
    })

    t.Run("rule override applied", func(t *testing.T) {
        yaml := `
rules:
  MISSING_FK_INDEX: error
  MISSING_LOCK_TIMEOUT: ignore
`
        // Write to temp file
        f := writeTempFile(t, yaml)
        cfg, err := config.LoadConfig(f)
        assertNoError(t, err)
        assertEqual(t, cfg.Rules["MISSING_FK_INDEX"], "error")
        assertEqual(t, cfg.Rules["MISSING_LOCK_TIMEOUT"], "ignore")
    })

    t.Run("env var expansion", func(t *testing.T) {
        os.Setenv("TEST_DB_URL", "postgresql://localhost/test")
        defer os.Unsetenv("TEST_DB_URL")
        
        yaml := "database_url: ${TEST_DB_URL}\n"
        f := writeTempFile(t, yaml)
        cfg, err := config.LoadConfig(f)
        assertNoError(t, err)
        assertEqual(t, cfg.DatabaseURL, "postgresql://localhost/test")
    })
}
```

---

## Running Tests

```bash
# All tests
go test ./...

# With race detector (required before release)
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific rule
go test ./internal/rules/ -run TestAddColumnDefault -v

# Integration tests only
go test ./internal/ -run TestIntegration -v

# Benchmarks
go test -bench=. -benchmem ./internal/parser/
```

---

## Coverage Targets

| Package | Target |
|---------|--------|
| `internal/rules/` | 95%+ |
| `internal/parser/` | 90%+ |
| `internal/reporter/` | 85%+ |
| `internal/config/` | 85%+ |
| Overall | 90%+ |

---

## Test Data SQL Files

All SQL in `testdata/` should be realistic migration-style SQL, not minimal one-liners. This catches bugs where the parser or rule engine behaves differently in context.

Good:
```sql
-- Migration: 20240312_add_user_status
-- Description: Add status column to users table

ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
CREATE INDEX idx_users_status ON users(status);
```

Not ideal (too minimal, misses context bugs):
```sql
ALTER TABLE t ADD COLUMN x TEXT NOT NULL DEFAULT 'v';
```
