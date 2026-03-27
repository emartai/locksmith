package rules

import pg_query "github.com/pganalyze/pg_query_go/v5"

// IndexWithoutConcurrentlyRule flags CREATE INDEX statements that omit CONCURRENTLY.
type IndexWithoutConcurrentlyRule struct{}

// ID returns the stable rule identifier.
func (r *IndexWithoutConcurrentlyRule) ID() string {
	return ruleIDIndexWithoutConcurrently
}

// Severity returns the default severity for the rule.
func (r *IndexWithoutConcurrentlyRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects index statements and flags blocking index builds.
func (r *IndexWithoutConcurrentlyRule) Check(stmt Statement) *Finding {
	node, ok := stmt.Node.(*pg_query.Node)
	if !ok || node == nil {
		return nil
	}

	indexStmt := node.GetIndexStmt()
	if indexStmt == nil || indexStmt.GetConcurrent() {
		return nil
	}

	return &Finding{
		RuleID:   r.ID(),
		Severity: r.Severity(),
		Line:     stmt.Line,
		Summary:  "CREATE INDEX without CONCURRENTLY",
		Why:      "Holds a SHARE lock on the table for the full duration of the index build, blocking all writes.",
		LockType: "SHARE",
		Fix:      "Use CREATE INDEX CONCURRENTLY. Note: CONCURRENTLY cannot run inside a transaction block.",
	}
}
