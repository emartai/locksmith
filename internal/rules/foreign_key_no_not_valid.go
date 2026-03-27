package rules

import pg_query "github.com/pganalyze/pg_query_go/v5"

// ForeignKeyNoNotValidRule flags foreign keys added without NOT VALID.
type ForeignKeyNoNotValidRule struct{}

// ID returns the stable rule identifier.
func (r *ForeignKeyNoNotValidRule) ID() string {
	return ruleIDForeignKeyNoNotValid
}

// Severity returns the default severity for the rule.
func (r *ForeignKeyNoNotValidRule) Severity() Severity {
	return SeverityDangerous
}

// Check inspects ALTER TABLE ADD CONSTRAINT statements for foreign keys without NOT VALID.
func (r *ForeignKeyNoNotValidRule) Check(stmt Statement) *Finding {
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
		if cmd == nil || cmd.GetSubtype() != pg_query.AlterTableType_AT_AddConstraint {
			continue
		}

		constraint := cmd.GetDef().GetConstraint()
		if constraint == nil {
			continue
		}

		if constraint.GetContype() != pg_query.ConstrType_CONSTR_FOREIGN {
			continue
		}

		if constraint.GetSkipValidation() {
			continue
		}

		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "ADD FOREIGN KEY without NOT VALID",
			Why:      "Validates every existing row in the table immediately. Holds SHARE ROW EXCLUSIVE lock blocking all writes during validation.",
			LockType: "SHARE ROW EXCLUSIVE",
			Fix:      "Add the constraint with NOT VALID first, then run VALIDATE CONSTRAINT in a separate migration.",
		}
	}

	return nil
}
