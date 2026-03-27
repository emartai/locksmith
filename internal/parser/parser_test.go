package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileSimpleCreateTable(t *testing.T) {
	path := writeTempSQLFile(t, "CREATE TABLE users (id BIGINT PRIMARY KEY);")

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want %d", len(result.Statements), 1)
	}

	if result.Statements[0].Raw != "CREATE TABLE users (id BIGINT PRIMARY KEY);" {
		t.Fatalf("Raw = %q, want %q", result.Statements[0].Raw, "CREATE TABLE users (id BIGINT PRIMARY KEY);")
	}

	if result.Statements[0].Node == nil {
		t.Fatal("Node = nil, want parsed node")
	}
}

func TestParseFileMultipleStatements(t *testing.T) {
	sql := "ALTER TABLE users ADD COLUMN email TEXT;\nALTER TABLE users ADD COLUMN status TEXT;"
	path := writeTempSQLFile(t, sql)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want %d", len(result.Statements), 2)
	}
}

func TestParseFileDetectsIgnoreComments(t *testing.T) {
	sql := "-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY, DROP_OBJECT\nCREATE INDEX idx_users_email ON users(email);"
	path := writeTempSQLFile(t, sql)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want %d", len(result.Statements), 1)
	}

	ignored := result.Statements[0].IgnoredRules
	if len(ignored) != 2 {
		t.Fatalf("len(IgnoredRules) = %d, want %d", len(ignored), 2)
	}

	if ignored[0] != "INDEX_WITHOUT_CONCURRENTLY" {
		t.Fatalf("IgnoredRules[0] = %q, want %q", ignored[0], "INDEX_WITHOUT_CONCURRENTLY")
	}

	if ignored[1] != "DROP_OBJECT" {
		t.Fatalf("IgnoredRules[1] = %q, want %q", ignored[1], "DROP_OBJECT")
	}
}

func TestParseFileCalculatesLineNumbers(t *testing.T) {
	sql := "\n\nALTER TABLE users ADD COLUMN email TEXT;\n\nALTER TABLE users ADD COLUMN status TEXT;"
	path := writeTempSQLFile(t, sql)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 2 {
		t.Fatalf("len(Statements) = %d, want %d", len(result.Statements), 2)
	}

	if result.Statements[0].Line != 3 {
		t.Fatalf("Statements[0].Line = %d, want %d", result.Statements[0].Line, 3)
	}

	if result.Statements[1].Line != 5 {
		t.Fatalf("Statements[1].Line = %d, want %d", result.Statements[1].Line, 5)
	}
}

func TestParseFileEmptyFile(t *testing.T) {
	path := writeTempSQLFile(t, "")

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 0 {
		t.Fatalf("len(Statements) = %d, want %d", len(result.Statements), 0)
	}
}

func TestParseFileInvalidStatementKeepsStatementWithNilNode(t *testing.T) {
	path := writeTempSQLFile(t, "ALTER TABLE users ADD COLUMN ;")

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want %d", len(result.Statements), 1)
	}

	if result.Statements[0].Node != nil {
		t.Fatal("Node != nil, want nil for unparsable statement")
	}

	if result.Statements[0].ParseError == "" {
		t.Fatal("ParseError = empty, want descriptive parse error")
	}
}

func TestParseFileDollarQuotedFunctionStaysSingleStatement(t *testing.T) {
	sql := `CREATE FUNCTION example() RETURNS void AS $$
BEGIN
  INSERT INTO t VALUES (1);
END;
$$ LANGUAGE plpgsql;`
	path := writeTempSQLFile(t, sql)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(result.Statements) != 1 {
		t.Fatalf("len(Statements) = %d, want 1", len(result.Statements))
	}
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
