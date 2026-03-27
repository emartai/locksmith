package rules

import pg_query "github.com/pganalyze/pg_query_go/v6"

// RenameObjectRule flags table and column renames.
type RenameObjectRule struct{}

// ID returns the stable rule identifier.
func (r *RenameObjectRule) ID() string {
	return ruleIDRenameObject
}

// Severity returns the default severity for the rule.
func (r *RenameObjectRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects statements for table and column renames.
func (r *RenameObjectRule) Check(stmt Statement) *Finding {
	node, ok := stmt.Node.(*pg_query.Node)
	if !ok || node == nil {
		return nil
	}

	renameStmt := node.GetRenameStmt()
	if renameStmt == nil {
		return nil
	}

	switch renameStmt.GetRenameType() {
	case pg_query.ObjectType_OBJECT_COLUMN:
		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "RENAME COLUMN",
			Why:      "Immediately breaks any application code referencing the old name. No lock phase - the error is instant on next query.",
			Fix:      "Add a new column/table with the new name, dual-write, migrate reads, then drop the old name in a future migration.",
		}
	case pg_query.ObjectType_OBJECT_TABLE:
		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "RENAME TABLE",
			Why:      "Immediately breaks any application code referencing the old name. No lock phase - the error is instant on next query.",
			Fix:      "Add a new column/table with the new name, dual-write, migrate reads, then drop the old name in a future migration.",
		}
	default:
		return nil
	}
}
