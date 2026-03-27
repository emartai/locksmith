package rules

import pg_query "github.com/pganalyze/pg_query_go/v6"

// TruncateTableRule flags TRUNCATE statements.
type TruncateTableRule struct{}

// ID returns the stable rule identifier.
func (r *TruncateTableRule) ID() string {
	return ruleIDTruncateTable
}

// Severity returns the default severity for the rule.
func (r *TruncateTableRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects statements for TRUNCATE operations.
func (r *TruncateTableRule) Check(stmt Statement) *Finding {
	node, ok := stmt.Node.(*pg_query.Node)
	if !ok || node == nil {
		return nil
	}

	if node.GetTruncateStmt() == nil {
		return nil
	}

	return &Finding{
		RuleID:   r.ID(),
		Severity: r.Severity(),
		Line:     stmt.Line,
		Summary:  "TRUNCATE TABLE",
		Why:      "Acquires ACCESS EXCLUSIVE lock. TRUNCATE CASCADE will silently delete data from all referencing tables.",
		LockType: "ACCESS EXCLUSIVE",
		Fix:      "Use batched DELETE for production data removal. Schedule TRUNCATE during a maintenance window with explicit sign-off.",
	}
}
