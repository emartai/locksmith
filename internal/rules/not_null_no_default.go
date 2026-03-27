package rules

import pg_query "github.com/pganalyze/pg_query_go/v5"

// NotNullNoDefaultRule flags SET NOT NULL on existing columns.
type NotNullNoDefaultRule struct{}

// ID returns the stable rule identifier.
func (r *NotNullNoDefaultRule) ID() string {
	return ruleIDNotNullNoDefault
}

// Severity returns the default severity for the rule.
func (r *NotNullNoDefaultRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects ALTER TABLE commands for SET NOT NULL operations.
func (r *NotNullNoDefaultRule) Check(stmt Statement) *Finding {
	node, ok := stmt.Node.(*pg_query.Node)
	if !ok || node == nil {
		return nil
	}

	alterStmt := node.GetAlterTableStmt()
	if alterStmt == nil {
		return nil
	}

	for _, cmdNode := range alterStmt.GetCmds() {
		cmd := cmdNode.GetAlterTableCmd()
		if cmd == nil || cmd.GetSubtype() != pg_query.AlterTableType_AT_SetNotNull {
			continue
		}

		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "SET NOT NULL on existing column",
			Why:      "Scans the entire table to verify no nulls. Holds ACCESS EXCLUSIVE lock during the scan.",
			LockType: "ACCESS EXCLUSIVE",
			Fix:      "Add a CHECK (col IS NOT NULL) constraint with NOT VALID, then VALIDATE CONSTRAINT separately.",
		}
	}

	return nil
}
