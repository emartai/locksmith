package rules

import pg_query "github.com/pganalyze/pg_query_go/v6"

// AlterColumnTypeRule flags ALTER COLUMN TYPE statements.
type AlterColumnTypeRule struct{}

// ID returns the stable rule identifier.
func (r *AlterColumnTypeRule) ID() string {
	return ruleIDAlterColumnType
}

// Severity returns the default severity for the rule.
func (r *AlterColumnTypeRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects ALTER TABLE commands for ALTER COLUMN TYPE operations.
func (r *AlterColumnTypeRule) Check(stmt Statement) *Finding {
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
		if cmd == nil || cmd.GetSubtype() != pg_query.AlterTableType_AT_AlterColumnType {
			continue
		}

		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "ALTER COLUMN TYPE",
			Why:      "Rewrites every row in the table to convert existing data. Holds ACCESS EXCLUSIVE lock for full rewrite duration.",
			LockType: "ACCESS EXCLUSIVE",
			Fix:      "Add a new column with the new type, backfill, deploy app to use new column, then drop old column.",
		}
	}

	return nil
}
