package rules

import pg_query "github.com/pganalyze/pg_query_go/v6"

// DropObjectRule flags DROP TABLE and ALTER TABLE ... DROP COLUMN statements.
type DropObjectRule struct{}

// ID returns the stable rule identifier.
func (r *DropObjectRule) ID() string {
	return ruleIDDropObject
}

// Severity returns the default severity for the rule.
func (r *DropObjectRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects statements for destructive table and column drops.
func (r *DropObjectRule) Check(stmt Statement) *Finding {
	node, ok := stmt.Node.(*pg_query.Node)
	if !ok || node == nil {
		return nil
	}

	if dropStmt := node.GetDropStmt(); dropStmt != nil {
		if dropStmt.GetRemoveType() == pg_query.ObjectType_OBJECT_TABLE {
			return &Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Line:     stmt.Line,
				Summary:  "DROP TABLE",
				Why:      "Acquires ACCESS EXCLUSIVE lock and permanently deletes table data. Application code referencing this table will immediately error.",
				LockType: "ACCESS EXCLUSIVE",
				Fix:      "Ensure all application code has been deployed without references to this table before dropping it.",
			}
		}

		return nil
	}

	alterStmt := node.GetAlterTableStmt()
	if alterStmt == nil {
		return nil
	}

	for _, cmdNode := range alterStmt.GetCmds() {
		cmd := cmdNode.GetAlterTableCmd()
		if cmd == nil || cmd.GetSubtype() != pg_query.AlterTableType_AT_DropColumn {
			continue
		}

		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "DROP COLUMN",
			Why:      "Acquires ACCESS EXCLUSIVE lock. If application code still reads this column, it will immediately error.",
			LockType: "ACCESS EXCLUSIVE",
			Fix:      "Deploy application code that no longer references this column, verify no reads in prod, then drop.",
		}
	}

	return nil
}
