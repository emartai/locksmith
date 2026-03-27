package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emartai/locksmith/internal/parser"
	"github.com/emartai/locksmith/internal/rules"
)

func runSingleRuleOnSQL(t *testing.T, sql string, rule rules.Rule) []rules.Finding {
	t.Helper()

	path := writeTempSQLFile(t, sql)
	result, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	engine := rules.NewEngine([]rules.Rule{rule})
	return engine.Run(*result)
}

func runEngineOnSQL(t *testing.T, sql string, engine *rules.Engine) []rules.Finding {
	t.Helper()

	path := writeTempSQLFile(t, sql)
	result, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	return engine.Run(*result)
}

func writeTempSQLFile(t *testing.T, sql string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "migration.sql")
	if err := os.WriteFile(path, []byte(sql), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	return path
}
