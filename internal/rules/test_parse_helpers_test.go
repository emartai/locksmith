package rules

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func mustParseNode(t *testing.T, sql string) interface{} {
	t.Helper()

	tree, err := pg_query.Parse(sql)
	if err != nil {
		t.Fatalf("pg_query.Parse() error = %v", err)
	}
	if len(tree.Stmts) == 0 {
		t.Fatal("pg_query.Parse() returned zero statements")
	}
	return tree.Stmts[0].Stmt
}
