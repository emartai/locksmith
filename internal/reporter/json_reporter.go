package reporter

import (
	"encoding/json"
	"io"

	"github.com/emartai/locksmith/internal/rules"
)

// JSONFinding is the machine-readable finding shape for CI integrations.
type JSONFinding struct {
	RuleID   string         `json:"rule_id"`
	Severity rules.Severity `json:"severity"`
	Line     int            `json:"line"`
	Summary  string         `json:"summary"`
	Why      string         `json:"why"`
	LockType string         `json:"lock_type,omitempty"`
	Fix      string         `json:"fix"`
}

// JSONOutput is the per-file JSON payload for machine-readable output.
type JSONOutput struct {
	File     string        `json:"file"`
	Findings []JSONFinding `json:"findings"`
	Passed   bool          `json:"passed"`
}

// JSONReporter writes machine-readable output.
type JSONReporter struct {
	writer io.Writer
}

// NewJSON creates a JSON reporter.
func NewJSON(writer io.Writer) *JSONReporter {
	if writer == nil {
		writer = io.Discard
	}
	return &JSONReporter{writer: writer}
}

// Print writes JSON output for one or more files.
func (r *JSONReporter) Print(outputs []JSONOutput) error {
	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(outputs)
}

// NewJSONOutput converts findings for a single file into the JSON schema.
func NewJSONOutput(file string, findings []rules.Finding) JSONOutput {
	converted := make([]JSONFinding, 0, len(findings))
	for _, finding := range findings {
		converted = append(converted, JSONFinding{
			RuleID:   finding.RuleID,
			Severity: finding.Severity,
			Line:     finding.Line,
			Summary:  finding.Summary,
			Why:      finding.Why,
			LockType: finding.LockType,
			Fix:      finding.Fix,
		})
	}

	return JSONOutput{
		File:     file,
		Findings: converted,
		Passed:   len(converted) == 0,
	}
}
