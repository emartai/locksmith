package rules

import pg_query "github.com/pganalyze/pg_query_go/v5"

// AddColumnDefaultRule flags ADD COLUMN statements that combine DEFAULT and NOT NULL.
type AddColumnDefaultRule struct{}

// ID returns the stable rule identifier.
func (r *AddColumnDefaultRule) ID() string {
	return ruleIDAddColumnDefault
}

// Severity returns the default severity for the rule.
func (r *AddColumnDefaultRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects ALTER TABLE ... ADD COLUMN commands for NOT NULL + DEFAULT definitions.
func (r *AddColumnDefaultRule) Check(stmt Statement) *Finding {
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
		if cmd == nil || cmd.GetSubtype() != pg_query.AlterTableType_AT_AddColumn {
			continue
		}

		columnDef := cmd.GetDef().GetColumnDef()
		if columnDef == nil {
			continue
		}

		if hasColumnDefault(columnDef) && hasNotNullConstraint(columnDef) {
			return &Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Line:     stmt.Line,
				Summary:  "ADD COLUMN with NOT NULL and DEFAULT",
				Why:      "On Postgres 10 and earlier, this rewrites the entire table. An ACCESS EXCLUSIVE lock is held for the full duration.",
				LockType: "ACCESS EXCLUSIVE",
				Fix:      "Add column as nullable first, backfill in batches, then add NOT NULL constraint separately.",
			}
		}
	}

	return nil
}

func hasColumnDefault(columnDef *pg_query.ColumnDef) bool {
	if columnDef.GetRawDefault() != nil || columnDef.GetCookedDefault() != nil {
		return true
	}

	for _, constraintNode := range columnDef.GetConstraints() {
		constraint := constraintNode.GetConstraint()
		if constraint != nil && constraint.GetContype() == pg_query.ConstrType_CONSTR_DEFAULT {
			return true
		}
	}

	return false
}

func hasNotNullConstraint(columnDef *pg_query.ColumnDef) bool {
	if columnDef.GetIsNotNull() {
		return true
	}

	for _, constraintNode := range columnDef.GetConstraints() {
		constraint := constraintNode.GetConstraint()
		if constraint != nil && constraint.GetContype() == pg_query.ConstrType_CONSTR_NOTNULL {
			return true
		}
	}

	return false
}
