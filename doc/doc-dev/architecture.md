# Locksmith — Architecture

## System Overview

Locksmith is a pure static analysis tool. It takes SQL migration files as input, parses them into an AST using the real Postgres parser, runs a set of rules against the AST, and produces structured output.

```
SQL File(s)
    │
    ▼
┌─────────────┐
│   Parser    │  pg_query_go → Postgres C parser → AST
└─────────────┘
    │
    ▼
┌─────────────┐
│ Rule Engine │  Walks AST nodes, applies each rule
└─────────────┘
    │
    ▼
┌─────────────┐
│  Reporter   │  Formats findings → terminal output
└─────────────┘
    │
    ▼
Exit code (0/1/2)
```

---

## Component Breakdown

### 1. Parser (`internal/parser/parser.go`)

Wraps `pg_query_go` to produce a normalized `ParseResult`.

```go
type ParseResult struct {
    Statements []Statement
    RawSQL     string
    FilePath   string
}

type Statement struct {
    Raw      string          // Original SQL text
    Line     int             // Line number in file
    Node     *pg_query.Node  // AST node
    Ignore   []string        // Rule IDs to ignore (from comment)
}
```

**Responsibilities**:
- Read file from disk
- Split into individual statements
- Parse each statement with pg_query_go
- Detect `-- locksmith:ignore RULE_ID` comments immediately preceding statements
- Attach line numbers

**Does NOT**:
- Execute SQL
- Connect to any database
- Modify any files

---

### 2. Rule Engine (`internal/rules/`)

#### Interface

Every rule implements the `Rule` interface:

```go
type Rule interface {
    ID() string
    Severity() Severity
    Check(stmt Statement) *Finding
}
```

#### Finding

```go
type Finding struct {
    RuleID   string
    Severity Severity
    Line     int
    Summary  string    // Short one-line description
    Why      string    // Explanation of danger
    LockType string    // e.g., "ACCESS EXCLUSIVE"
    Fix      string    // Safe rewrite suggestion
}
```

#### Engine

```go
type Engine struct {
    rules []Rule
}

func (e *Engine) Run(result ParseResult) []Finding {
    var findings []Finding
    for _, stmt := range result.Statements {
        for _, rule := range e.rules {
            if ignored(stmt, rule.ID()) {
                continue
            }
            if f := rule.Check(stmt); f != nil {
                findings = append(findings, *f)
            }
        }
    }
    return findings
}
```

#### Rule Registration

Rules are registered in `engine.go`:

```go
func DefaultEngine() *Engine {
    return &Engine{
        rules: []Rule{
            &AddColumnDefaultRule{},
            &AlterColumnTypeRule{},
            &NotNullNoDefaultRule{},
            &IndexWithoutConcurrentlyRule{},
            &DropObjectRule{},
            &ForeignKeyNoNotValidRule{},
            &TruncateRule{},
            &MissingFKIndexRule{},
            &RenameObjectRule{},
            &MissingLockTimeoutRule{},
        },
    }
}
```

---

### 3. Individual Rules (`internal/rules/*.go`)

Each rule is a separate file. Rules use `pg_query_go`'s AST node types to detect patterns.

#### Example: IndexWithoutConcurrentlyRule

```go
type IndexWithoutConcurrentlyRule struct{}

func (r *IndexWithoutConcurrentlyRule) ID() string       { return "INDEX_WITHOUT_CONCURRENTLY" }
func (r *IndexWithoutConcurrentlyRule) Severity() Severity { return Dangerous }

func (r *IndexWithoutConcurrentlyRule) Check(stmt Statement) *Finding {
    node := stmt.Node.GetIndexStmt()
    if node == nil {
        return nil
    }
    if node.Concurrent {
        return nil // Safe
    }
    return &Finding{
        RuleID:   r.ID(),
        Severity: r.Severity(),
        Line:     stmt.Line,
        Summary:  "CREATE INDEX without CONCURRENTLY",
        Why:      "Holds a SHARE lock for the full index build duration, blocking all writes",
        LockType: "SHARE",
        Fix:      "Use CREATE INDEX CONCURRENTLY instead",
    }
}
```

---

### 4. Reporter (`internal/reporter/reporter.go`)

Formats `[]Finding` into terminal output. Uses `fatih/color` for colors.

```go
type Reporter struct {
    writer io.Writer
}

func (r *Reporter) Print(findings []Finding, filePath string) {
    // Print file header
    // For each finding: print severity block with Why/Lock/Fix
    // Print summary line
}
```

Output format matches the spec in `design.md`.

---

### 5. CLI (`cmd/check.go`)

Built with `cobra`. The `check` command:

1. Accepts one or more file paths or a directory path
2. Calls the parser on each `.sql` file
3. Passes results to the rule engine
4. Passes findings to the reporter
5. Exits with the appropriate code

```go
var checkCmd = &cobra.Command{
    Use:   "check [file|dir]",
    Short: "Analyze migration files for dangerous operations",
    RunE:  runCheck,
}
```

#### Flags
```
--database-url string    Postgres connection string for live row count estimates
--config string          Path to locksmith.yml (default: ./locksmith.yml)
--format string          Output format: text|json (default: text)
--severity string        Minimum severity to report: dangerous|warning|info (default: dangerous)
```

---

### 6. Config (`locksmith.yml`)

Parsed by `viper`. Loaded automatically from the current directory or `--config` flag.

```yaml
# locksmith.yml
rules:
  MISSING_FK_INDEX: error
  MISSING_LOCK_TIMEOUT: ignore

ignore_paths:
  - migrations/legacy/
  - migrations/seeds/

database_url: ${DATABASE_URL}  # Viper expands env vars
```

---

## Data Flow

### Check a single file

```
locksmith check migrations/001_add_users.sql
    │
    ├── Load locksmith.yml (if present)
    │
    ├── parser.ParseFile("migrations/001_add_users.sql")
    │       │
    │       ├── Read file bytes
    │       ├── Split by semicolon (statement boundary)
    │       ├── For each statement:
    │       │       ├── pg_query.Parse(sql) → AST
    │       │       ├── Detect ignore comments
    │       │       └── Build Statement struct
    │       └── Return ParseResult
    │
    ├── engine.Run(ParseResult)
    │       │
    │       └── For each Statement × Rule:
    │               ├── Check if ignored
    │               └── Rule.Check(stmt) → *Finding
    │
    ├── reporter.Print(findings)
    │
    └── os.Exit(exitCode)
```

### Check a directory

```
locksmith check migrations/
    │
    ├── Walk directory for *.sql files
    ├── Sort files by name (ensures consistent ordering)
    ├── For each file: run the single-file flow above
    └── Aggregate findings across all files
        └── Print summary: "X issues found across Y files"
```

---

## Error Handling

| Scenario | Behavior |
|----------|---------|
| File not found | Print error, exit 1 |
| File is not valid SQL | Print parse error with line number, continue to next file |
| pg_query_go parse error | Print error, treat as WARNING (don't fail silently) |
| Config file not found | Continue with defaults (config is optional) |
| Empty migration file | Print "no statements found", exit 0 |
| All statements ignored | Print "all statements ignored", exit 0 |

---

## Testing Strategy

### Unit Tests (per rule)

Each rule has a `_test.go` file with table-driven tests:

```go
func TestIndexWithoutConcurrently(t *testing.T) {
    tests := []struct {
        name     string
        sql      string
        wantFlag bool
    }{
        {
            name:     "standard CREATE INDEX — should flag",
            sql:      "CREATE INDEX idx_users_email ON users(email)",
            wantFlag: true,
        },
        {
            name:     "CREATE INDEX CONCURRENTLY — should pass",
            sql:      "CREATE INDEX CONCURRENTLY idx_users_email ON users(email)",
            wantFlag: false,
        },
        {
            name:     "ignored — should pass",
            sql:      "-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY\nCREATE INDEX idx ON t(col)",
            wantFlag: false,
        },
    }
    // ...
}
```

### Integration Tests

`testdata/migrations/` contains real-world migration SQL files organized as:
```
testdata/
├── dangerous/        # Files that should trigger DANGEROUS rules
├── warnings/         # Files that should trigger WARNING rules
├── clean/            # Files that should pass all rules
└── ignored/          # Files with ignore comments
```

The integration test runs `locksmith check` on each directory and asserts the expected exit code.

---

## Release Process

1. Tag the commit: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. GoReleaser runs automatically via GitHub Actions
4. Produces binaries for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
5. Uploads to GitHub Releases with checksums
6. Homebrew tap is updated automatically

---

## Performance Targets

| Operation | Target |
|-----------|--------|
| Parse + analyze single file | < 100ms |
| Parse + analyze 100-file directory | < 2s |
| Binary cold start | < 50ms |
| GitHub Action total runtime | < 30s |

`pg_query_go` is fast — parsing a 1,000-line migration file takes under 10ms. Performance is not a concern for MVP.
