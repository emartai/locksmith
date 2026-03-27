package internal_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	locksmithcmd "github.com/emartai/locksmith/cmd"
	"github.com/emartai/locksmith/internal/reporter"
	"github.com/emartai/locksmith/internal/rules"
)

func TestDangerousMigrations(t *testing.T) {
	tests := []struct {
		file   string
		ruleID string
	}{
		{"001_add_column_with_default.sql", "ADD_COLUMN_DEFAULT"},
		{"002_create_index_blocking.sql", "INDEX_WITHOUT_CONCURRENTLY"},
		{"003_add_foreign_key_no_not_valid.sql", "FOREIGN_KEY_NO_NOT_VALID"},
		{"004_drop_column.sql", "DROP_OBJECT"},
		{"005_rename_column.sql", "RENAME_OBJECT"},
		{"006_truncate.sql", "TRUNCATE_TABLE"},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			output, code := runCheckJSON(t, filepath.Join("..", "testdata", "migrations", "dangerous", test.file), "dangerous")
			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if len(output) != 1 {
				t.Fatalf("len(output) = %d, want 1", len(output))
			}
			if len(output[0].Findings) == 0 {
				t.Fatal("len(output[0].Findings) = 0, want at least 1")
			}
			if output[0].Findings[0].Severity != rules.SeverityDangerous {
				t.Fatalf("severity = %q, want %q", output[0].Findings[0].Severity, rules.SeverityDangerous)
			}
			if output[0].Findings[0].RuleID != test.ruleID {
				t.Fatalf("rule id = %q, want %q", output[0].Findings[0].RuleID, test.ruleID)
			}
		})
	}
}

func TestCleanMigrations(t *testing.T) {
	tests := []string{
		"001_safe_add_column.sql",
		"002_concurrent_index.sql",
		"003_fk_with_not_valid.sql",
	}

	for _, file := range tests {
		t.Run(file, func(t *testing.T) {
			output, code := runCheckJSON(t, filepath.Join("..", "testdata", "migrations", "clean", file), "warning")
			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if len(output) != 1 {
				t.Fatalf("len(output) = %d, want 1", len(output))
			}
			if len(output[0].Findings) != 0 {
				t.Fatalf("len(output[0].Findings) = %d, want 0", len(output[0].Findings))
			}
		})
	}
}

func TestWarningMigrations(t *testing.T) {
	output, code := runCheckJSON(t, filepath.Join("..", "testdata", "migrations", "warnings", "001_fk_no_index.sql"), "warning")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if len(output) != 1 {
		t.Fatalf("len(output) = %d, want 1", len(output))
	}
	if len(output[0].Findings) != 1 {
		t.Fatalf("len(output[0].Findings) = %d, want 1", len(output[0].Findings))
	}
	if output[0].Findings[0].Severity != rules.SeverityWarning {
		t.Fatalf("severity = %q, want %q", output[0].Findings[0].Severity, rules.SeverityWarning)
	}
	if output[0].Findings[0].RuleID != "MISSING_FK_INDEX" {
		t.Fatalf("rule id = %q, want %q", output[0].Findings[0].RuleID, "MISSING_FK_INDEX")
	}
}

func runCheckJSON(t *testing.T, target string, severity string) ([]reporter.JSONOutput, int) {
	t.Helper()

	configPath := writeEmptyConfig(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := locksmithcmd.ExecuteArgs(
		[]string{"check", "--format", "json", "--severity", severity, "--config", configPath, target},
		&stdout,
		&stderr,
		"test", "abc1234", "2024-03-01",
	)
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v, stderr = %s", err, stderr.String())
	}

	var output []reporter.JSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, stdout = %s", err, stdout.String())
	}

	return output, code
}

func writeEmptyConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "locksmith.yml")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
