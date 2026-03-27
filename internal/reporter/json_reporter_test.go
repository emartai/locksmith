package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/emartai/locksmith/internal/rules"
)

func TestJSONReporterProducesValidJSONSchema(t *testing.T) {
	var out bytes.Buffer
	reporter := NewJSON(&out)

	err := reporter.Print([]JSONOutput{
		NewJSONOutput("migrations/001_add_users.sql", []rules.Finding{
			{
				RuleID:   "ADD_COLUMN_DEFAULT",
				Severity: rules.SeverityDangerous,
				Line:     4,
				Summary:  "ADD COLUMN with NOT NULL and DEFAULT",
				Why:      "On Postgres 10 and earlier, this rewrites the entire table.",
				LockType: "ACCESS EXCLUSIVE",
				Fix:      "Add column nullable first, backfill, then add NOT NULL.",
			},
		}),
	})
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	if !json.Valid(out.Bytes()) {
		t.Fatal("Print() did not produce valid JSON")
	}

	var payload []map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("len(payload) = %d, want 1", len(payload))
	}

	if payload[0]["file"] != "migrations/001_add_users.sql" {
		t.Fatalf("file = %#v, want %#v", payload[0]["file"], "migrations/001_add_users.sql")
	}

	findings, ok := payload[0]["findings"].([]any)
	if !ok || len(findings) != 1 {
		t.Fatalf("findings = %#v, want single finding", payload[0]["findings"])
	}

	finding, ok := findings[0].(map[string]any)
	if !ok {
		t.Fatalf("finding = %#v, want object", findings[0])
	}

	for _, field := range []string{"rule_id", "severity", "line", "summary", "why", "lock_type", "fix"} {
		if _, exists := finding[field]; !exists {
			t.Fatalf("finding missing field %q: %#v", field, finding)
		}
	}
}

func TestJSONReporterPassedOutput(t *testing.T) {
	var out bytes.Buffer
	reporter := NewJSON(&out)

	err := reporter.Print([]JSONOutput{
		NewJSONOutput("migrations/001_safe_migration.sql", nil),
	})
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	var payload []JSONOutput
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("len(payload) = %d, want 1", len(payload))
	}
	if !payload[0].Passed {
		t.Fatal("Passed = false, want true")
	}
	if len(payload[0].Findings) != 0 {
		t.Fatalf("len(Findings) = %d, want 0", len(payload[0].Findings))
	}
}
