package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandDangerousDirectoryExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := ExecuteArgs(
		[]string{"check", filepath.Join("..", "testdata", "migrations", "dangerous")},
		&stdout,
		&stderr,
		"test", "abc1234", "2024-03-01",
	)
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestCheckCommandCleanDirectoryExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := ExecuteArgs(
		[]string{"check", filepath.Join("..", "testdata", "migrations", "clean")},
		&stdout,
		&stderr,
		"test", "abc1234", "2024-03-01",
	)
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestCheckCommandJSONOutputFile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "locksmith-report.json")

	code, err := ExecuteArgs(
		[]string{
			"check",
			"--format", "json",
			"--severity", "warning",
			"--output", outputPath,
			"--config", filepath.Join("..", "locksmith.yml"),
			filepath.Join("..", "testdata", "migrations", "warnings"),
		},
		&stdout,
		&stderr,
		"test", "abc1234", "2024-03-01",
	)
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var payload []map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("len(payload) = %d, want 1", len(payload))
	}
}

func TestVersionOutputIncludesBuildMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code, err := ExecuteArgs([]string{"--version"}, &stdout, &stderr, "1.0.0", "abc1234", "2024-03-01")
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	want := "locksmith version 1.0.0 (commit: abc1234, built: 2024-03-01)\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestCollectSQLFilesSkipsEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.sql")
	if err := os.WriteFile(empty, []byte(""), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	normal := filepath.Join(dir, "ok.sql")
	if err := os.WriteFile(normal, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	files, warnings, progress, err := collectSQLFiles([]string{dir})
	if err != nil {
		t.Fatalf("collectSQLFiles() error = %v", err)
	}
	if len(files) != 1 || files[0] != normal {
		t.Fatalf("files = %#v, want %#v", files, []string{normal})
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if len(progress) != 0 {
		t.Fatalf("len(progress) = %d, want 0", len(progress))
	}
}
