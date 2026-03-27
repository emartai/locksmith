package rules

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// MissingLockTimeoutRule flags dangerous migrations that do not set a lock timeout.
type MissingLockTimeoutRule struct{}

// ID returns the stable rule identifier.
func (r *MissingLockTimeoutRule) ID() string {
	return ruleIDMissingLockTimeout
}

// Severity returns the default severity for the rule.
func (r *MissingLockTimeoutRule) Severity() Severity {
	return SeverityWarning
}

// Check is unused because this rule is evaluated after all other findings are collected.
func (r *MissingLockTimeoutRule) Check(stmt Statement) *Finding {
	return nil
}

// CheckResult inspects the full migration for timeouts after dangerous operations have been identified.
func (r *MissingLockTimeoutRule) CheckResult(result ParseResult, findings []Finding) *Finding {
	firstDangerousLine := 0
	for _, finding := range findings {
		if finding.Severity != SeverityDangerous {
			continue
		}
		if hasTimeoutBeforeLine(result, finding.Line) {
			continue
		}
		firstDangerousLine = finding.Line
		break
	}

	if firstDangerousLine == 0 {
		return nil
	}

	return &Finding{
		RuleID:   r.ID(),
		Severity: r.Severity(),
		Line:     firstDangerousLine,
		Summary:  "No lock timeout set before dangerous operation",
		Why:      "Without lock_timeout, this migration will wait indefinitely to acquire a lock. In a busy database this creates a lock queue pileup that can take down the service.",
		Fix:      "Add SET lock_timeout = '2s'; at the top of the migration.",
	}
}

func hasTimeoutBeforeLine(result ParseResult, dangerousLine int) bool {
	for _, stmt := range result.Statements {
		if stmt.Line >= dangerousLine {
			break
		}

		node, ok := stmt.Node.(*pg_query.Node)
		if !ok || node == nil {
			continue
		}

		setStmt := node.GetVariableSetStmt()
		if setStmt == nil {
			continue
		}

		name := strings.ToLower(setStmt.GetName())
		if name == "lock_timeout" || name == "statement_timeout" {
			return true
		}
	}

	return false
}
