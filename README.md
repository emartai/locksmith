<p align="center">
  <svg width="240" height="64" viewBox="0 0 240 64" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="locksmith logo">
    <rect x="8" y="18" width="32" height="28" rx="2" stroke="#0F172A" stroke-width="2"/>
    <line x1="8" y1="32" x2="40" y2="32" stroke="#0F172A" stroke-width="2"/>
    <line x1="24" y1="18" x2="24" y2="46" stroke="#0F172A" stroke-width="2"/>
    <rect x="26" y="20" width="12" height="10" rx="1.5" fill="#0F172A"/>
    <text x="56" y="38" font-family="JetBrains Mono, monospace" font-size="20" fill="#0F172A">locksmith</text>
  </svg>
</p>

# Locksmith

Prevent dangerous Postgres migrations before they hit production.

![CI](https://github.com/emartai/locksmith/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/github/go-mod/go-version/emartai/locksmith)
![License](https://img.shields.io/badge/license-MIT-blue)
![Latest Release](https://img.shields.io/github/v/release/emartai/locksmith)

## Install

```bash
# Homebrew (macOS/Linux)
brew install emartai/locksmith/locksmith

# Direct download (Linux/macOS)
curl -sSL https://locksmith.dev/install.sh | sh

# Go install
go install github.com/emartai/locksmith@latest
```

## Quick start

```bash
locksmith check migrations/
```

Example output for a dangerous migration:

```text
❌ DANGEROUS Line 1
   ADD COLUMN with NOT NULL and DEFAULT
   
   Why:   On Postgres 10 and earlier, this rewrites the entire table. An ACCESS EXCLUSIVE lock is held for the full duration.
   Lock:  ACCESS EXCLUSIVE
   Fix:   Add column as nullable first, backfill in batches, then add NOT NULL constraint separately.


1 issues found. Migration blocked.
```

## Example output

```text
❌ DANGEROUS Line 1
   CREATE INDEX without CONCURRENTLY
   
   Why:   Holds a SHARE lock on the table for the full duration of the index build, blocking all writes.
   Lock:  SHARE
   Fix:   Use CREATE INDEX CONCURRENTLY. Note: CONCURRENTLY cannot run inside a transaction block.

1 issues found. Migration blocked.

⚠️ WARNING   Line 1
   Foreign key column has no index
   
   Why:   Without an index on the FK column, ON DELETE CASCADE operations on the parent table cause full scans of this table. Join queries will also be slow.
   Fix:   Add CREATE INDEX CONCURRENTLY idx_table_col ON table(fk_col) in the same migration.

1 warnings found.

✅ PASSED - no issues found
```

## Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `ADD_COLUMN_DEFAULT` | `DANGEROUS` | `ADD COLUMN` with `NOT NULL` and `DEFAULT` |
| `ALTER_COLUMN_TYPE` | `DANGEROUS` | `ALTER COLUMN ... TYPE` |
| `NOT_NULL_NO_DEFAULT` | `DANGEROUS` | `SET NOT NULL` on an existing column |
| `INDEX_WITHOUT_CONCURRENTLY` | `DANGEROUS` | `CREATE INDEX` without `CONCURRENTLY` |
| `DROP_OBJECT` | `DANGEROUS` | `DROP TABLE` or `DROP COLUMN` |
| `FOREIGN_KEY_NO_NOT_VALID` | `DANGEROUS` | foreign key added without `NOT VALID` |
| `TRUNCATE_TABLE` | `DANGEROUS` | `TRUNCATE` |
| `MISSING_FK_INDEX` | `WARNING` | foreign key column without a matching index |
| `RENAME_OBJECT` | `DANGEROUS` | `RENAME COLUMN` or `RENAME TABLE` |
| `MISSING_LOCK_TIMEOUT` | `WARNING` | dangerous migration without `lock_timeout` or `statement_timeout` |

## GitHub Action

```yaml
name: Migration Safety

on:
  pull_request:
    paths:
      - "migrations/**"

jobs:
  locksmith:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: emartai/locksmith/action@v1
        with:
          path: migrations/
          severity: dangerous
          format: text
```

## Config file

```yaml
rules:
  MISSING_FK_INDEX: error
  MISSING_LOCK_TIMEOUT: ignore

ignore_paths:
  - migrations/legacy/

database_url: ${DATABASE_URL}
```

## Ignore comments

```sql
-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY
CREATE INDEX idx_small_table ON config(key);
```

## How it works

Locksmith parses migration SQL with the Postgres parser through `pg_query_go`.
It turns each statement into an AST node with file and line metadata.
It runs a rule engine over those nodes and reports dangerous or warning findings before the migration runs.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
