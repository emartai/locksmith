# Contributing

## Development Setup

### Prerequisites

- Go 1.22+
- `CGO_ENABLED=1`
- `make`

### Clone and build

```bash
git clone https://github.com/emartai/locksmith.git
cd locksmith
make build
```

### Run tests

```bash
make test
```

No Docker is required.

## Adding a New Rule

1. Create `internal/rules/your_rule_name.go`.
2. Implement the `Rule` interface: `ID`, `Severity`, and `Check`.
3. Register the rule in `internal/rules/engine.go` inside `DefaultEngine()`.
4. Add `internal/rules/your_rule_name_test.go` with table-driven tests.
5. Add fixture SQL to `testdata/migrations/`.
6. Add the rule to the rules table in `README.md`.
7. Add the rule to `doc/doc-dev/rules.md`.

Code template:

```go
package rules

import pg_query "github.com/pganalyze/pg_query_go/v6"

type YourRuleNameRule struct{}

func (r *YourRuleNameRule) ID() string {
	return "YOUR_RULE_NAME"
}

func (r *YourRuleNameRule) Severity() Severity {
	return SeverityWarning
}

func (r *YourRuleNameRule) Check(stmt Statement) *Finding {
	node, ok := stmt.Node.(*pg_query.Node)
	if !ok || node == nil {
		return nil
	}

	// Inspect the AST and return nil when the statement is safe.

	return &Finding{
		RuleID:   r.ID(),
		Severity: r.Severity(),
		Line:     stmt.Line,
		Summary:  "Short description",
		Why:      "Why this migration pattern is risky.",
		LockType: "",
		Fix:      "Safe rewrite guidance.",
	}
}
```

## Rule Quality Standards

- Every rule must have tests for a positive case, a negative case, and an ignore comment case.
- False positives are worse than false negatives. When in doubt, do not flag.
- Every rule must provide a `Fix` that works on all Postgres versions `>= 10`.

## Testing

```bash
# Unit tests
go test ./...

# Race detector
go test -race ./...

# Integration tests
make integration-test
```

## Submitting a PR

- One rule per PR.
- Use the title format: `rule: add RULE_NAME - short description`
- Include the rule implementation, tests, `README.md` update, and `rules.md` update.

## Code Style

- `gofmt` and `goimports` are required. CI enforces both.
- No global variables.
- Error strings should be lowercase and should not end with punctuation.
- Document exported types and functions.
