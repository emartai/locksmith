package rules

import (
	"slices"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// MissingFKIndexRule flags foreign keys that do not get a matching index in the same migration.
type MissingFKIndexRule struct{}

// ID returns the stable rule identifier.
func (r *MissingFKIndexRule) ID() string {
	return ruleIDMissingFKIndex
}

// Severity returns the default severity for the rule.
func (r *MissingFKIndexRule) Severity() Severity {
	return SeverityWarning
}

// Check is unused because this rule needs full migration context.
func (r *MissingFKIndexRule) Check(stmt Statement) *Finding {
	return nil
}

// CheckWithContext inspects the full migration for a matching index on the FK columns.
func (r *MissingFKIndexRule) CheckWithContext(stmt Statement, result ParseResult) *Finding {
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
		if constraint == nil || constraint.GetContype() != pg_query.ConstrType_CONSTR_FOREIGN {
			continue
		}

		tableName := alterStmt.GetRelation().GetRelname()
		fkColumns := getConstraintColumnNames(constraint.GetFkAttrs())
		if len(fkColumns) == 0 {
			continue
		}

		if migrationHasMatchingIndex(result, tableName, fkColumns) {
			continue
		}

		return &Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Line:     stmt.Line,
			Summary:  "Foreign key column has no index",
			Why:      "Without an index on the FK column, ON DELETE CASCADE operations on the parent table cause full scans of this table. Join queries will also be slow.",
			Fix:      "Add CREATE INDEX CONCURRENTLY idx_table_col ON table(fk_col) in the same migration.",
		}
	}

	return nil
}

func migrationHasMatchingIndex(result ParseResult, tableName string, fkColumns []string) bool {
	for _, stmt := range result.Statements {
		node, ok := stmt.Node.(*pg_query.Node)
		if !ok || node == nil {
			continue
		}

		indexStmt := node.GetIndexStmt()
		if indexStmt == nil || indexStmt.GetRelation().GetRelname() != tableName {
			continue
		}

		indexColumns := getIndexColumnNames(indexStmt.GetIndexParams())
		if slices.Equal(indexColumns, fkColumns) {
			return true
		}
	}

	return false
}

func getConstraintColumnNames(nodes []*pg_query.Node) []string {
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		name := node.GetString_().GetSval()
		if name != "" {
			names = append(names, name)
		}
	}

	return names
}

func getIndexColumnNames(nodes []*pg_query.Node) []string {
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		indexElem := node.GetIndexElem()
		if indexElem == nil || indexElem.GetName() == "" || indexElem.GetExpr() != nil {
			return nil
		}
		names = append(names, indexElem.GetName())
	}

	return names
}
