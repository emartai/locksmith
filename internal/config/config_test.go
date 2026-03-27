package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigReturnsDefaultsWhenNoFileExists(t *testing.T) {
	dir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Rules) != 0 {
		t.Fatalf("len(cfg.Rules) = %d, want 0", len(cfg.Rules))
	}
	if len(cfg.IgnorePaths) != 0 {
		t.Fatalf("len(cfg.IgnorePaths) = %d, want 0", len(cfg.IgnorePaths))
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("cfg.DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
}

func TestLoadConfigReadsRulesOverrides(t *testing.T) {
	path := writeConfigFile(t, `
rules:
  missing_fk_index: error
  MISSING_LOCK_TIMEOUT: ignore
ignore_paths:
  - migrations/legacy/
`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Rules["MISSING_FK_INDEX"] != "error" {
		t.Fatalf("cfg.Rules[MISSING_FK_INDEX] = %q, want %q", cfg.Rules["MISSING_FK_INDEX"], "error")
	}
	if cfg.Rules["MISSING_LOCK_TIMEOUT"] != "ignore" {
		t.Fatalf("cfg.Rules[MISSING_LOCK_TIMEOUT] = %q, want %q", cfg.Rules["MISSING_LOCK_TIMEOUT"], "ignore")
	}
	if len(cfg.IgnorePaths) != 1 || cfg.IgnorePaths[0] != "migrations/legacy/" {
		t.Fatalf("cfg.IgnorePaths = %#v, want %#v", cfg.IgnorePaths, []string{"migrations/legacy/"})
	}
}

func TestLoadConfigExpandsEnvironmentVariables(t *testing.T) {
	const wantURL = "postgres://example"
	if err := os.Setenv("DATABASE_URL", wantURL); err != nil {
		t.Fatalf("os.Setenv() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("DATABASE_URL")
	})

	path := writeConfigFile(t, `
database_url: ${DATABASE_URL}
`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DatabaseURL != wantURL {
		t.Fatalf("cfg.DatabaseURL = %q, want %q", cfg.DatabaseURL, wantURL)
	}
}

func writeConfigFile(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "locksmith.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
