# Locksmith — 25 Coding Agent Prompts

## How to Use This File

These 25 prompts are designed to be executed sequentially by a coding agent (Claude, Cursor, Aider, etc.). Each prompt builds on the previous. Do not skip prompts or reorder them.

Before starting:
- Have `context.md`, `design.md`, `rules.md`, `architecture.md`, and `security.md` in the project root
- Have Go 1.25+ installed
- Have a GitHub repository created at `github.com/emartai/locksmith`

After each prompt: verify the output compiles and tests pass before moving to the next prompt.

---

## Prompt 1 — Project Scaffold

```
Create the initial Go project structure for a CLI tool called "locksmith".

Initialize a Go module at github.com/emartai/locksmith using Go 1.25.

Create the following empty directory structure:
- cmd/
- internal/parser/
- internal/rules/
- internal/reporter/
- testdata/migrations/dangerous/
- testdata/migrations/clean/
- testdata/migrations/warnings/
- action/
- .github/workflows/

Create go.mod with the module path and Go version.

Add these dependencies to go.mod (do not run go mod tidy yet — just add them):
- github.com/spf13/cobra v1.8.0
- github.com/spf13/viper v1.18.0
- github.com/fatih/color v1.16.0
- github.com/pganalyze/pg_query_go/v6 v6.2.2

Create a minimal main.go that prints "locksmith" and exits.

Create a .gitignore for Go projects.
```

---

## Prompt 2 — Go Module Setup

```
Run go mod tidy to resolve all dependencies in go.mod.

Then verify the project compiles with go build ./...

If pg_query_go requires CGO (it does — it wraps a C library), ensure the go.mod and environment are set up correctly for CGO compilation. Add a comment in main.go noting that CGO_ENABLED=1 is required.

Create a Makefile with these targets:
- make build      → go build -o bin/locksmith ./
- make test       → go test ./...
- make lint       → golangci-lint run (if installed)
- make clean      → rm -rf bin/

Verify: make build produces a binary in bin/locksmith that runs.
```

---

## Prompt 3 — Core Types

```
Create internal/rules/types.go with the following types. Do not implement any logic yet — types only.

Severity type (string enum): Dangerous, Warning, Info
Constants for each severity: SeverityDangerous = "DANGEROUS", SeverityWarning = "WARNING", SeverityInfo = "INFO"

Finding struct with fields:
- RuleID string
- Severity Severity  
- Line int
- FilePath string
- Summary string      (short one-line description)
- Why string          (explanation of the danger)
- LockType string     (e.g., "ACCESS EXCLUSIVE" — empty string if not applicable)
- Fix string          (safe rewrite suggestion)

Rule interface with methods:
- ID() string
- Severity() Severity
- Check(stmt Statement) *Finding

Statement struct with fields:
- Raw string          (original SQL text of this statement)
- Line int            (line number in the file where the statement starts)
- Node interface{}    (will hold *pg_query.Node — use interface{} for now)
- IgnoredRules []string  (rule IDs from locksmith:ignore comments)

ParseResult struct with fields:
- Statements []Statement
- RawSQL string
- FilePath string

Write unit tests in internal/rules/types_test.go that verify:
- Severity constants have correct string values
- Finding struct can be constructed and fields are accessible
```

---

## Prompt 4 — Parser

```
Create internal/parser/parser.go.

Implement a Parser struct and a ParseFile(path string) (*ParseResult, error) function.

The function must:
1. Read the file at the given path
2. Split the SQL into individual statements by semicolons (be careful — semicolons inside string literals should not split)
3. For each statement:
   a. Parse it with pg_query_go: pg_query.Parse(sql)
   b. Detect if the line immediately before the statement contains "-- locksmith:ignore RULE_ID" and extract the rule ID(s)
   c. Calculate the line number of the statement in the original file
   d. Build a Statement struct
4. Return a ParseResult

Use github.com/pganalyze/pg_query_go/v6 for parsing. Import it as pg_query.

Handle these error cases:
- File not found: return a descriptive error
- File is empty: return a ParseResult with zero statements (not an error)
- pg_query.Parse fails: return the statement with a nil Node (do not fail the whole file)

Create internal/parser/parser_test.go with tests for:
- ParseFile on a simple CREATE TABLE statement
- ParseFile on a multi-statement file (two ALTER TABLE statements separated by semicolon)
- Correct detection of locksmith:ignore comments
- Correct line number calculation
- Empty file returns zero statements without error

Create testdata/migrations/clean/001_create_users.sql with:
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
```

---

## Prompt 5 — Rule Engine

```
Create internal/rules/engine.go.

Implement an Engine struct that holds a slice of Rules.

Implement:
- NewEngine(rules []Rule) *Engine
- DefaultEngine() *Engine  (returns engine with all 10 rules registered — use placeholder rules for now)
- (e *Engine) Run(result ParseResult) []Finding

The Run method:
1. Iterates over every Statement in the ParseResult
2. For each statement, runs every registered Rule
3. Skips a rule if the statement's IgnoredRules contains the rule's ID
4. Collects all non-nil Findings
5. Returns sorted findings (by line number ascending)

Create a placeholder rule in internal/rules/placeholder.go called NoOpRule that always returns nil. Register 10 instances of it (with different IDs) in DefaultEngine() so the engine compiles and runs. We will replace these one by one in subsequent prompts.

Write tests in internal/rules/engine_test.go:
- Engine with no rules returns empty findings
- Engine with a rule that always returns a finding returns that finding
- Engine skips rules listed in IgnoredRules
- Engine returns findings sorted by line number
```

---

## Prompt 6 — Rule 1: ADD_COLUMN_DEFAULT

```
Implement the first real rule. Replace the NoOpRule placeholder for ADD_COLUMN_DEFAULT.

Create internal/rules/add_column_default.go.

Rule ID: ADD_COLUMN_DEFAULT
Severity: DANGEROUS

Detection logic using pg_query_go AST:
- Look for AlterTableStmt nodes
- Check if any AlterTableCmd has cmdtype = AT_AddColumn
- Check if the column definition (ColumnDef) has both:
  a. A DEFAULT value specified
  b. A NOT NULL constraint OR is_not_null = true

Finding output:
- Summary: "ADD COLUMN with NOT NULL and DEFAULT"
- Why: "On Postgres 10 and earlier, this rewrites the entire table. An ACCESS EXCLUSIVE lock is held for the full duration."
- LockType: "ACCESS EXCLUSIVE"
- Fix: "Add column as nullable first, backfill in batches, then add NOT NULL constraint separately."

Write tests with these cases:
- ALTER TABLE t ADD COLUMN x TEXT → should NOT flag (nullable, no default)
- ALTER TABLE t ADD COLUMN x TEXT DEFAULT 'val' → should NOT flag (has default but nullable)
- ALTER TABLE t ADD COLUMN x TEXT NOT NULL DEFAULT 'val' → SHOULD flag
- ALTER TABLE t ADD COLUMN x TEXT NOT NULL → should NOT flag (no default)
- With locksmith:ignore ADD_COLUMN_DEFAULT comment → should NOT flag
```

---

## Prompt 7 — Rule 2: INDEX_WITHOUT_CONCURRENTLY

```
Implement internal/rules/index_without_concurrently.go.

Rule ID: INDEX_WITHOUT_CONCURRENTLY  
Severity: DANGEROUS

Detection logic:
- Look for IndexStmt nodes
- Check if node.Concurrent is false

Finding output:
- Summary: "CREATE INDEX without CONCURRENTLY"
- Why: "Holds a SHARE lock on the table for the full duration of the index build, blocking all writes."
- LockType: "SHARE"
- Fix: "Use CREATE INDEX CONCURRENTLY. Note: CONCURRENTLY cannot run inside a transaction block."

Write tests:
- CREATE INDEX idx ON t(col) → SHOULD flag
- CREATE UNIQUE INDEX idx ON t(col) → SHOULD flag
- CREATE INDEX CONCURRENTLY idx ON t(col) → should NOT flag
- CREATE UNIQUE INDEX CONCURRENTLY idx ON t(col) → should NOT flag
- With ignore comment → should NOT flag

Add a clean migration to testdata/migrations/clean/ using CONCURRENTLY.
Add a dangerous migration to testdata/migrations/dangerous/ with a standard CREATE INDEX.
```

---

## Prompt 8 — Rule 3: FOREIGN_KEY_NO_NOT_VALID

```
Implement internal/rules/foreign_key_no_not_valid.go.

Rule ID: FOREIGN_KEY_NO_NOT_VALID
Severity: DANGEROUS

Detection logic:
- Look for AlterTableStmt nodes
- Find AlterTableCmd with cmdtype = AT_AddConstraint
- Check if the constraint is a CONSTR_FOREIGN (foreign key)
- Check if skip_validation is false (meaning NOT VALID was not specified)

Finding output:
- Summary: "ADD FOREIGN KEY without NOT VALID"
- Why: "Validates every existing row in the table immediately. Holds SHARE ROW EXCLUSIVE lock blocking all writes during validation."
- LockType: "SHARE ROW EXCLUSIVE"
- Fix: "Add the constraint with NOT VALID first, then run VALIDATE CONSTRAINT in a separate migration."

Write tests:
- ALTER TABLE orders ADD CONSTRAINT fk FOREIGN KEY (user_id) REFERENCES users(id) → SHOULD flag
- ALTER TABLE orders ADD CONSTRAINT fk FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID → should NOT flag
- ALTER TABLE orders ADD CONSTRAINT chk CHECK (amount > 0) → should NOT flag (not a FK)
- With ignore comment → should NOT flag
```

---

## Prompt 9 — Rule 4: DROP_OBJECT

```
Implement internal/rules/drop_object.go.

Rule ID: DROP_OBJECT
Severity: DANGEROUS

Detection logic — flag any of these:
- DropStmt nodes with removeType = OBJECT_TABLE → DROP TABLE
- AlterTableStmt with AlterTableCmd cmdtype = AT_DropColumn → DROP COLUMN

Finding output:
- For DROP TABLE:
  - Summary: "DROP TABLE"
  - Why: "Acquires ACCESS EXCLUSIVE lock and permanently deletes table data. Application code referencing this table will immediately error."
  - LockType: "ACCESS EXCLUSIVE"
  - Fix: "Ensure all application code has been deployed without references to this table before dropping it."

- For DROP COLUMN:
  - Summary: "DROP COLUMN"
  - Why: "Acquires ACCESS EXCLUSIVE lock. If application code still reads this column, it will immediately error."
  - LockType: "ACCESS EXCLUSIVE"
  - Fix: "Deploy application code that no longer references this column, verify no reads in prod, then drop."

Write tests:
- DROP TABLE users → SHOULD flag
- DROP TABLE IF EXISTS users → SHOULD flag
- ALTER TABLE users DROP COLUMN email → SHOULD flag
- ALTER TABLE users DROP COLUMN IF EXISTS email → SHOULD flag
- ALTER TABLE users ADD COLUMN email TEXT → should NOT flag
- DROP INDEX idx → should NOT flag (index drops are lower risk)
```

---

## Prompt 10 — Rules 5–7: Remaining DANGEROUS Rules

```
Implement three more rules in separate files.

--- Rule 5: ALTER_COLUMN_TYPE (internal/rules/alter_column_type.go) ---
Rule ID: ALTER_COLUMN_TYPE, Severity: DANGEROUS
Detect: AlterTableStmt with AT_AlterColumnType cmdtype
Summary: "ALTER COLUMN TYPE"
Why: "Rewrites every row in the table to convert existing data. Holds ACCESS EXCLUSIVE lock for full rewrite duration."
LockType: "ACCESS EXCLUSIVE"
Fix: "Add a new column with the new type, backfill, deploy app to use new column, then drop old column."

--- Rule 6: NOT_NULL_NO_DEFAULT (internal/rules/not_null_no_default.go) ---
Rule ID: NOT_NULL_NO_DEFAULT, Severity: DANGEROUS
Detect: AlterTableStmt with AT_SetNotNull cmdtype (SET NOT NULL on existing column)
Summary: "SET NOT NULL on existing column"
Why: "Scans the entire table to verify no nulls. Holds ACCESS EXCLUSIVE lock during the scan."
LockType: "ACCESS EXCLUSIVE"  
Fix: "Add a CHECK (col IS NOT NULL) constraint with NOT VALID, then VALIDATE CONSTRAINT separately."

--- Rule 7: TRUNCATE_TABLE (internal/rules/truncate_table.go) ---
Rule ID: TRUNCATE_TABLE, Severity: DANGEROUS
Detect: TruncateStmt nodes
Summary: "TRUNCATE TABLE"
Why: "Acquires ACCESS EXCLUSIVE lock. TRUNCATE CASCADE will silently delete data from all referencing tables."
LockType: "ACCESS EXCLUSIVE"
Fix: "Use batched DELETE for production data removal. Schedule TRUNCATE during a maintenance window with explicit sign-off."

Write at least 3 tests per rule covering: flagged case, non-flagged case, ignore comment case.
```

---

## Prompt 11 — Rules 8–10: WARNING Rules

```
Implement the three WARNING-level rules.

--- Rule 8: MISSING_FK_INDEX (internal/rules/missing_fk_index.go) ---
Rule ID: MISSING_FK_INDEX, Severity: WARNING
Detection: When a FOREIGN KEY constraint is added (AlterTableStmt with AT_AddConstraint of type CONSTR_FOREIGN), check if the same migration also creates an index on the referencing column. If not, flag it.
Summary: "Foreign key column has no index"
Why: "Without an index on the FK column, ON DELETE CASCADE operations on the parent table cause full scans of this table. Join queries will also be slow."
Fix: "Add CREATE INDEX CONCURRENTLY idx_table_col ON table(fk_col) in the same migration."
Note: This rule requires analyzing multiple statements together (looking across the ParseResult). The Check method will need access to the full ParseResult, not just a single statement. Adjust the Rule interface or add an alternate method CheckWithContext(stmt Statement, result ParseResult) *Finding.

--- Rule 9: RENAME_OBJECT (internal/rules/rename_object.go) ---  
Rule ID: RENAME_OBJECT, Severity: DANGEROUS
Detect: 
- RenameStmt with renameType = OBJECT_COLUMN → RENAME COLUMN
- RenameStmt with renameType = OBJECT_TABLE → RENAME TABLE
Summary for column: "RENAME COLUMN"
Summary for table: "RENAME TABLE"
Why: "Immediately breaks any application code referencing the old name. No lock phase — the error is instant on next query."
Fix: "Add a new column/table with the new name, dual-write, migrate reads, then drop the old name in a future migration."

--- Rule 10: MISSING_LOCK_TIMEOUT (internal/rules/missing_lock_timeout.go) ---
Rule ID: MISSING_LOCK_TIMEOUT, Severity: WARNING
Detection: This rule operates at the ParseResult level (not per-statement). If any statement in the migration would trigger a DANGEROUS rule AND there is no SET lock_timeout or SET statement_timeout in the migration, flag it.
Implement as a special post-processing step in the Engine.Run method that checks the full result after individual rules have run.
Summary: "No lock timeout set before dangerous operation"
Why: "Without lock_timeout, this migration will wait indefinitely to acquire a lock. In a busy database this creates a lock queue pileup that can take down the service."
Fix: "Add SET lock_timeout = '2s'; at the top of the migration."

Write tests for all three rules.
```

---

## Prompt 12 — Reporter

```
Create internal/reporter/reporter.go.

Implement a Reporter struct with a Print(findings []Finding, filePath string) method.

Output format (match design.md exactly):

❌ DANGEROUS   Line 4
   ADD COLUMN with NOT NULL and DEFAULT
   
   Why:   Rewrites entire table on Postgres < 11
   Lock:  ACCESS EXCLUSIVE — blocks all reads and writes
   Fix:   Add column nullable → backfill → add NOT NULL

⚠️  WARNING    Line 9
   Foreign key column has no index
   
   Why:   ON DELETE CASCADE causes full table scans
   Fix:   Add CREATE INDEX CONCURRENTLY on the FK column

✅ PASSED — no issues found

2 issues found. Migration blocked.

Rules:
- Use fatih/color: red bold for DANGEROUS, yellow bold for WARNING, green bold for PASSED
- Severity label is always padded to 9 characters for alignment
- Line number on the same line as severity
- Why/Lock/Fix are indented with 3 spaces, left-aligned labels
- Lock line only shown if LockType is non-empty
- Blank line between findings
- Final summary: "X issues found. Migration blocked." for dangerous, "X warnings found." for warning-only
- "✅ PASSED — no issues found" if findings is empty

Also implement a PrintSummary(allFindings map[string][]Finding) method for directory-level output:
"Checked 5 files. 2 files with issues (3 dangerous, 1 warning)."

Implement a JSON output mode:
type JSONOutput struct {
    File     string    `json:"file"`
    Findings []Finding `json:"findings"`
    Passed   bool      `json:"passed"`
}

Write tests verifying the text output format matches expected strings exactly.
```

---

## Prompt 13 — CLI: check command

```
Create cmd/check.go using cobra.

Implement the check command:
- Use: "check [file|dir...]"
- Short: "Analyze migration files for dangerous operations"
- Aliases: ["c"]

Flags:
- --database-url string    (placeholder for v1 — accept the flag but print "database connection coming in v1" if used)
- --config string          path to locksmith.yml (default: auto-detect in current dir)
- --format string          "text" or "json" (default: "text")
- --severity string        minimum severity to report: "dangerous", "warning", "info" (default: "dangerous")
- --no-color               disable colored output

Logic:
1. Collect file paths from args (if arg is a directory, walk it for *.sql files, sorted alphabetically)
2. Skip files in paths matching ignore_paths from config
3. For each file: parse → run engine → collect findings
4. Output via reporter (text or json based on --format)
5. Exit codes:
   - 0: no issues
   - 1: one or more DANGEROUS findings
   - 2: one or more WARNING findings (and no DANGEROUS)

Update main.go to register the check command with cobra.

Wire up the root command with:
- Version flag showing current version (use ldflags -X main.version)
- --help default behavior

Write an integration test that:
- Calls check on testdata/migrations/dangerous/ and asserts exit code 1
- Calls check on testdata/migrations/clean/ and asserts exit code 0
```

---

## Prompt 14 — Config File Support

```
Create internal/config/config.go.

Implement LoadConfig(path string) (*Config, error) using viper.

Config struct:
type Config struct {
    Rules       map[string]string `mapstructure:"rules"`         // rule ID → "error"|"warning"|"info"|"ignore"
    IgnorePaths []string          `mapstructure:"ignore_paths"`
    DatabaseURL string            `mapstructure:"database_url"`
}

LoadConfig behavior:
- If path is explicitly provided, load from that path (error if not found)
- If path is empty, look for locksmith.yml in current directory (no error if not found — use defaults)
- Viper should expand environment variables in values (${DATABASE_URL} → actual value)

Apply config in the engine:
- If a rule's ID is in Config.Rules with value "ignore", skip that rule entirely
- If a rule's ID is in Config.Rules with value "error", override its severity to DANGEROUS
- If a rule's ID is in Config.Rules with value "warning", override its severity to WARNING

Create a sample locksmith.yml in the project root:
rules:
  MISSING_FK_INDEX: error
  MISSING_LOCK_TIMEOUT: ignore

ignore_paths:
  - migrations/legacy/

Write tests:
- LoadConfig returns defaults when no file exists
- LoadConfig correctly reads rules overrides
- LoadConfig expands environment variables
- Engine applies rule overrides from config
```

---

## Prompt 15 — Test Data and Full Test Suite

```
Create a comprehensive test data suite in testdata/migrations/.

In testdata/migrations/dangerous/ create these SQL files:

001_add_column_with_default.sql:
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

002_create_index_blocking.sql:
CREATE INDEX idx_orders_user_id ON orders(user_id);

003_add_foreign_key_no_not_valid.sql:
ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id);

004_drop_column.sql:
ALTER TABLE users DROP COLUMN legacy_token;

005_rename_column.sql:
ALTER TABLE users RENAME COLUMN email TO email_address;

006_truncate.sql:
TRUNCATE sessions;

In testdata/migrations/clean/ create:

001_safe_add_column.sql:
ALTER TABLE users ADD COLUMN metadata JSONB;

002_concurrent_index.sql:
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);

003_fk_with_not_valid.sql:
ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
ALTER TABLE orders VALIDATE CONSTRAINT fk_user;

In testdata/migrations/warnings/ create:

001_fk_no_index.sql:
ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
-- No corresponding index creation

Then write an integration test file in internal/integration_test.go that:
- Runs locksmith check on each dangerous file and asserts DANGEROUS finding with correct rule ID
- Runs locksmith check on each clean file and asserts zero findings
- Runs locksmith check on each warning file and asserts WARNING finding

Run go test ./... and verify all tests pass.
```

---

## Prompt 16 — GitHub Action

```
Create the GitHub Action definition.

Create action/action.yml:
name: "Locksmith — Postgres Migration Safety"
description: "Analyze Postgres migration files for dangerous operations before merging"
author: "emartai"

inputs:
  path:
    description: "Path to migration files or directory"
    required: false
    default: "migrations/"
  severity:
    description: "Minimum severity to fail on: dangerous, warning"
    required: false
    default: "dangerous"
  format:
    description: "Output format: text, json"
    required: false
    default: "text"

runs:
  using: "composite"
  steps:
    - name: Download Locksmith
      shell: bash
      run: |
        VERSION="${{ env.LOCKSMITH_VERSION || 'latest' }}"
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
        
        if [ "$VERSION" = "latest" ]; then
          VERSION=$(curl -s https://api.github.com/repos/emartai/locksmith/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
        fi
        
        URL="https://github.com/emartai/locksmith/releases/download/${VERSION}/locksmith_${OS}_${ARCH}.tar.gz"
        curl -sSL "$URL" | tar -xz -C /usr/local/bin locksmith
        chmod +x /usr/local/bin/locksmith
        
        # Verify checksum
        CHECKSUMS_URL="https://github.com/emartai/locksmith/releases/download/${VERSION}/checksums.txt"
        curl -sSL "$CHECKSUMS_URL" -o /tmp/checksums.txt
        sha256sum --ignore-missing --check /tmp/checksums.txt
        
    - name: Run Locksmith
      shell: bash
      run: |
        locksmith check "${{ inputs.path }}" \
          --severity "${{ inputs.severity }}" \
          --format "${{ inputs.format }}"

Create .github/workflows/locksmith-example.yml showing how users would use the action:
name: Migration Safety
on:
  pull_request:
    paths:
      - 'migrations/**'
jobs:
  locksmith:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: emartai/locksmith-action@v1
        with:
          path: migrations/

Create a README in action/ explaining the action inputs and outputs.
```

---

## Prompt 17 — CI Pipeline

```
Create .github/workflows/ci.yml for the project's own CI.

The CI pipeline must:

On every push and pull_request to main:

1. Test job:
   - matrix: [ubuntu-latest, macos-latest]
   - Go version: 1.25.8
   - Steps:
     a. Checkout
     b. Set up Go
     c. Cache Go modules
     d. go mod verify
     e. go build ./...
     f. go test ./... -v -race -coverprofile=coverage.out
     g. Upload coverage to Codecov (optional)

2. Lint job:
   - ubuntu-latest only
   - Uses golangci-lint-action@v4
   - golangci-lint version: v1.57

3. Security scan job:
   - ubuntu-latest only
   - Install and run govulncheck ./...

4. Integration test job:
   - ubuntu-latest only
   - Build the binary
   - Run locksmith check testdata/migrations/dangerous/ and assert exit code 1
   - Run locksmith check testdata/migrations/clean/ and assert exit code 0

Create .golangci.yml with linting rules:
- Enable: errcheck, govet, staticcheck, unused, gofmt, goimports
- Disable: gochecknoglobals (too strict for CLI tools)
- Max line length: 120

The CI badge URL will be:
https://github.com/emartai/locksmith/actions/workflows/ci.yml/badge.svg
```

---

## Prompt 18 — GoReleaser Config

```
Create .goreleaser.yml for automated binary releases.

Configuration must produce:
- Binaries for: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Archive format: .tar.gz for unix, .zip for windows
- Binary name: locksmith
- Include README.md and LICENSE in each archive
- Generate checksums.txt (SHA256)
- Create GitHub Release with auto-generated changelog
- Build with ldflags to embed version, commit, and build date:
  -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}

Homebrew tap configuration:
- tap repo: emartai/homebrew-locksmith
- formula name: locksmith
- description: "Prevent dangerous Postgres migrations before they hit production"
- license: MIT
- install: bin.install "locksmith"
- test: system "#{bin}/locksmith --version"

Create .github/workflows/release.yml:
- Trigger: on push of tags matching v*
- Steps:
  a. Checkout with fetch-depth: 0 (GoReleaser needs git history)
  b. Set up Go 1.25.8
  c. Run GoReleaser: goreleaser release --clean
  d. Environment: GITHUB_TOKEN, GPG_FINGERPRINT (for signing)

Update main.go to print version info from ldflags:
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

---

## Prompt 19 — README

```
Create the project README.md following the structure in design.md exactly.

The README must include:

1. Logo (inline SVG from design.md, centered)
2. Tagline: "Prevent dangerous Postgres migrations before they hit production"
3. Badges: CI, Go Version, License (MIT), Latest Release
4. Blank line separator

5. Install section with 3 methods:
   # Homebrew (macOS/Linux)
   brew install emartai/locksmith/locksmith
   
   # Direct download (Linux/macOS)
   curl -sSL https://locksmith.dev/install.sh | sh
   
   # Go install
   go install github.com/emartai/locksmith@latest

6. Quick start:
   locksmith check migrations/
   
   With a real example output block (use the dangerous test migration output)

7. Example output section — full terminal output block showing ❌ ⚠️ ✅

8. Rules table:
   | Rule | Severity | Description |
   for all 10 rules from rules.md

9. GitHub Action section with complete YAML example

10. Config file section showing full locksmith.yml example

11. Ignore comment section with example

12. How it works (3-line technical explanation — parser → AST → rules)

13. Contributing section (link to CONTRIBUTING.md)

14. License section

Tone requirements (from design.md):
- No exclamation points
- No "powerful", "amazing", "incredible"  
- Technical and direct
- Write like Terraform or gh CLI docs
- Short sentences
```

---

## Prompt 20 — CONTRIBUTING.md

```
Create CONTRIBUTING.md.

Include these sections:

1. Development Setup
   - Prerequisites: Go 1.25+, CGO_ENABLED=1, make
   - Clone and build: git clone, make build
   - Run tests: make test
   - Note: no Docker required

2. Adding a New Rule
   Step-by-step guide:
   a. Create internal/rules/your_rule_name.go
   b. Implement the Rule interface (ID, Severity, Check)
   c. Register in internal/rules/engine.go DefaultEngine()
   d. Add test file internal/rules/your_rule_name_test.go with table-driven tests
   e. Add test data to testdata/migrations/
   f. Add rule to rules table in README.md
   g. Add rule to rules.md documentation
   
   Include a complete code template for a new rule.

3. Rule Quality Standards
   - Every rule must have tests for: positive case (should flag), negative case (should not flag), ignore comment case
   - False positives are worse than false negatives — when in doubt, don't flag
   - Every rule must provide a Fix that works on all Postgres versions >= 10

4. Testing
   - Run unit tests: go test ./...
   - Run with race detector: go test -race ./...
   - Run integration tests: make integration-test

5. Submitting a PR
   - One rule per PR
   - PR title format: "rule: add RULE_NAME — short description"
   - PR must include: rule implementation, tests, README update, rules.md update

6. Code Style
   - gofmt and goimports required (CI enforces)
   - No global variables
   - Error strings lowercase, no punctuation
   - Document exported types and functions
```

---

## Prompt 21 — install.sh Script

```
Create install.sh — the one-line installer script.

The script must:
1. Detect OS (linux, darwin, windows)
2. Detect architecture (amd64, arm64)
3. Fetch the latest release version from GitHub API
4. Download the appropriate binary archive
5. Verify SHA256 checksum against checksums.txt
6. Extract the binary to /usr/local/bin/locksmith (or $HOME/.local/bin if no sudo)
7. Verify installation by running locksmith --version
8. Print a success message

Handle these edge cases:
- curl not available: fallback to wget
- /usr/local/bin not writable: install to $HOME/.local/bin and print PATH instructions
- Checksum verification fails: exit with error, do not install
- Unsupported OS/arch: print clear error

The script should be usable as:
curl -sSL https://locksmith.dev/install.sh | sh

Or with explicit version:
curl -sSL https://locksmith.dev/install.sh | sh -s -- --version v1.0.0

Create an uninstall.sh that removes the binary.

Both scripts must work on macOS and Linux. Windows users should use the Go install method.
```

---

## Prompt 22 — JSON Output Mode

```
Implement the JSON output mode for CI integration.

When --format json is passed, output:
[
  {
    "file": "migrations/001_add_users.sql",
    "passed": false,
    "findings": [
      {
        "rule_id": "ADD_COLUMN_DEFAULT",
        "severity": "DANGEROUS",
        "line": 4,
        "summary": "ADD COLUMN with NOT NULL and DEFAULT",
        "why": "On Postgres 10 and earlier, this rewrites the entire table.",
        "lock_type": "ACCESS EXCLUSIVE",
        "fix": "Add column nullable first, backfill, then add NOT NULL."
      }
    ]
  }
]

If all files pass, output:
[
  {
    "file": "migrations/001_safe_migration.sql",
    "passed": true,
    "findings": []
  }
]

Implement in internal/reporter/json_reporter.go.

Add --output flag: write JSON to a file instead of stdout.
locksmith check migrations/ --format json --output locksmith-report.json

This is useful for:
- Consuming Locksmith results in other tools
- Storing reports as CI artifacts
- GitHub Actions summary integration

Update the GitHub Action to optionally upload the JSON report as an artifact.

Write tests verifying the JSON output is valid JSON and matches the expected schema.
```

---

## Prompt 23 — GitHub Actions Summary Integration

```
Update the GitHub Action to write a formatted summary to the GitHub Actions job summary.

When running in GitHub Actions (GITHUB_STEP_SUMMARY env var is set):
- Write a Markdown table of findings to $GITHUB_STEP_SUMMARY
- The summary appears in the GitHub Actions UI under the job

Format:
## Locksmith Migration Safety Report

| File | Rule | Severity | Line | Summary |
|------|------|----------|------|---------|
| migrations/001.sql | ADD_COLUMN_DEFAULT | ❌ DANGEROUS | 4 | ADD COLUMN with NOT NULL and DEFAULT |
| migrations/002.sql | INDEX_WITHOUT_CONCURRENTLY | ❌ DANGEROUS | 1 | CREATE INDEX without CONCURRENTLY |

**2 dangerous issues found across 2 files. Merge blocked.**

Or if clean:
## Locksmith Migration Safety Report
✅ All migration files passed. No dangerous operations detected.

Implement this in internal/reporter/summary_reporter.go.

Detection: check if os.Getenv("GITHUB_STEP_SUMMARY") is non-empty.
Write to the file path in GITHUB_STEP_SUMMARY using os.OpenFile with append flag.

This feature activates automatically — no flag needed — when running in GitHub Actions.

Write a test that mocks GITHUB_STEP_SUMMARY to a temp file and verifies the output.
```

---

## Prompt 24 — Final Polish and Edge Cases

```
Fix edge cases and add final polish before launch.

1. Statement boundary detection improvement:
   The naive semicolon split breaks on SQL with dollar-quoted strings:
   CREATE FUNCTION example() RETURNS void AS $$
     BEGIN
       INSERT INTO t VALUES (1);
     END;
   $$ LANGUAGE plpgsql;
   
   Update the parser to correctly handle dollar-quoted strings and not split inside them.

2. Multi-statement transaction handling:
   Handle migrations that wrap everything in BEGIN/COMMIT.
   The MISSING_LOCK_TIMEOUT rule should look for SET lock_timeout inside the transaction, not just at file level.

3. Empty statement filtering:
   Some migration tools generate files with trailing semicolons that create empty statements.
   Filter out statements where Raw is empty or whitespace-only after parsing.

4. Better error messages:
   When pg_query_go fails to parse a statement, include:
   - The line number
   - The first 80 characters of the statement
   - The parse error from pg_query_go
   
5. --version output:
   locksmith version 1.0.0 (commit: abc1234, built: 2024-03-01)
   
6. Graceful handling of non-SQL files:
   If a .sql file in a directory is actually empty or a symlink, skip it with a warning instead of crashing.

7. Large directory performance:
   If checking a directory with > 50 files, show a progress indicator:
   Checking migrations/ (47 files)...

Run the full test suite: go test ./... -race
Verify all 10 rules work correctly against testdata/.
Run go vet ./... and confirm zero warnings.
Run gofmt -l . and confirm zero unformatted files.
```

---

## Prompt 25 — Launch Readiness Verification

```
Final launch readiness check. Verify every item in this checklist.

Build verification:
- [ ] go build ./... succeeds with zero errors
- [ ] go test ./... -race passes with zero failures
- [ ] go vet ./... produces zero warnings  
- [ ] gofmt -l . produces zero output (all files formatted)
- [ ] Binary size is under 20MB (go build -ldflags="-s -w")

Feature verification — run each of these and confirm correct output:
- [ ] locksmith check testdata/migrations/dangerous/001_add_column_with_default.sql → exit 1, shows ❌ DANGEROUS with correct rule ID and fix
- [ ] locksmith check testdata/migrations/clean/ → exit 0, shows ✅ PASSED
- [ ] locksmith check testdata/migrations/dangerous/ → exit 1, shows multiple findings
- [ ] locksmith check --format json testdata/migrations/dangerous/001_add_column_with_default.sql → exit 1, valid JSON output
- [ ] locksmith check --no-color testdata/migrations/dangerous/001_add_column_with_default.sql → exit 1, no ANSI color codes in output
- [ ] echo "-- locksmith:ignore ADD_COLUMN_DEFAULT" > /tmp/test.sql && echo "ALTER TABLE t ADD COLUMN x TEXT NOT NULL DEFAULT 'val';" >> /tmp/test.sql && locksmith check /tmp/test.sql → exit 0

Documentation verification:
- [ ] README.md renders correctly on GitHub
- [ ] All 10 rules are in the README rules table
- [ ] Install instructions work (test at least the go install method)
- [ ] GitHub Action YAML example in README is syntactically valid

File checklist — all these files must exist and be non-empty:
- [ ] main.go
- [ ] go.mod, go.sum
- [ ] cmd/check.go
- [ ] internal/parser/parser.go
- [ ] internal/rules/engine.go
- [ ] internal/rules/types.go
- [ ] internal/rules/add_column_default.go
- [ ] internal/rules/index_without_concurrently.go
- [ ] internal/rules/foreign_key_no_not_valid.go
- [ ] internal/rules/drop_object.go
- [ ] internal/rules/alter_column_type.go
- [ ] internal/rules/not_null_no_default.go
- [ ] internal/rules/truncate_table.go
- [ ] internal/rules/missing_fk_index.go
- [ ] internal/rules/rename_object.go
- [ ] internal/rules/missing_lock_timeout.go
- [ ] internal/reporter/reporter.go
- [ ] internal/reporter/json_reporter.go
- [ ] internal/config/config.go
- [ ] action/action.yml
- [ ] .github/workflows/ci.yml
- [ ] .github/workflows/release.yml
- [ ] .goreleaser.yml
- [ ] locksmith.yml (example config)
- [ ] README.md
- [ ] CONTRIBUTING.md
- [ ] install.sh
- [ ] LICENSE (MIT)

If any check fails, fix it before considering the MVP ready for launch.

Final step: create a LAUNCH_CHECKLIST.md summarizing what to do on launch day:
1. Tag v1.0.0 and push → triggers GoReleaser
2. Verify GitHub Release appears with all 5 binary archives and checksums.txt
3. Test brew install from Homebrew tap
4. Test curl installer script
5. Open the GitHub repository and verify README renders correctly
6. Post to Hacker News (Show HN: Locksmith — catch dangerous Postgres migrations before they reach production)
7. Post to r/PostgreSQL and r/devops
8. Tweet / post with a terminal recording showing locksmith catching a dangerous migration
```
