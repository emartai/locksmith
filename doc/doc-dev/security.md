# Locksmith — Security Guidelines

## Overview

Locksmith is a static analysis tool. It reads SQL migration files and analyzes their AST. For MVP, it does not connect to any database, does not transmit data, and does not require authentication.

This document covers:
1. What Locksmith does and does not access
2. Security posture for the `--database-url` flag (v1)
3. GitHub Action security model
4. Dependency security
5. Vulnerability disclosure policy

---

## Data Access Model

### MVP (Static Analysis Only)

| Data | Accessed | Transmitted | Stored |
|------|----------|-------------|--------|
| SQL migration files | ✅ Read locally | ❌ Never | ❌ Never |
| Database schema | ❌ Not accessed | ❌ Never | ❌ Never |
| Live database data | ❌ Not accessed | ❌ Never | ❌ Never |
| User credentials | ❌ Not accessed | ❌ Never | ❌ Never |
| Environment variables | ❌ Not read | ❌ Never | ❌ Never |

Locksmith MVP reads only the SQL file(s) you explicitly pass to it. Nothing else is read, nothing is sent anywhere.

### v1 (`--database-url` flag)

When `--database-url` is provided:

| Data | Accessed | Transmitted | Stored |
|------|----------|-------------|--------|
| `pg_stat_user_tables` | ✅ Row counts only | ❌ Local only | ❌ Never |
| `pg_indexes` | ✅ Index metadata | ❌ Local only | ❌ Never |
| Actual table data | ❌ Never | ❌ Never | ❌ Never |
| Database credentials | ✅ Used to connect | ❌ Never logged | ❌ Never |

The database URL is used only to query table metadata (row counts, index presence). Locksmith never reads application data, never queries user tables for content, and never logs or stores the connection string.

---

## GitHub Action Security Model

### What the Action Does
1. Downloads the Locksmith binary from GitHub Releases
2. Verifies the binary checksum (SHA256)
3. Runs `locksmith check <path>` on the migration files in the repository
4. Reports output to the PR check

### What the Action Does NOT Do
- Does not receive a database URL (MVP)
- Does not exfiltrate repository contents
- Does not call any external API
- Does not store any state between runs

### Pinning Versions
Users should always pin the action to a specific version tag, never `@main`:

```yaml
# Good — pinned to release
- uses: your-org/locksmith-action@v1.0.0

# Acceptable — pinned to major version
- uses: your-org/locksmith-action@v1

# Bad — unpinned
- uses: your-org/locksmith-action@main
```

### Supply Chain Integrity
Every GitHub Release includes:
- SHA256 checksums for all binaries (`checksums.txt`)
- GPG signature on the checksums file
- The GitHub Action downloads the binary and verifies the checksum before executing

```yaml
# In the action internals:
sha256sum --check checksums.txt
```

---

## Dependency Security

### Go Module Security

All dependencies are pinned in `go.sum`. Run these before any release:

```bash
# Check for known vulnerabilities
govulncheck ./...

# Audit dependencies
go mod verify

# Update dependencies (review diffs carefully)
go get -u ./...
go mod tidy
```

### Key Dependencies and Their Risk Profile

| Dependency | Purpose | Risk Level | Notes |
|------------|---------|------------|-------|
| `pg_query_go` | SQL parsing | Low | Wraps Postgres C library, no network access |
| `cobra` | CLI framework | Low | Widely used, actively maintained |
| `viper` | Config parsing | Low | Reads local YAML only |
| `fatih/color` | Terminal color | Very Low | No network, no file I/O beyond stdout |
| `pgx` (v1) | DB driver | Medium | Handles credentials — review carefully |

### Automated Vulnerability Scanning

Add to CI (`ci.yml`):
```yaml
- name: Security scan
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
```

---

## Binary Distribution Security

### GoReleaser Signing

Configure GoReleaser to sign release artifacts:

```yaml
# .goreleaser.yml
signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"
```

### Homebrew Tap

The Homebrew formula must specify the SHA256 of the downloaded binary. GoReleaser generates this automatically.

```ruby
sha256 "abc123..."  # Must be verified on each release
```

---

## Credential Handling (v1 `--database-url`)

### Rules
1. **Never log the database URL** — mask it in all output, even debug output
2. **Never write credentials to disk** — don't cache the connection string
3. **Prefer environment variables** over inline flags:

```bash
# Preferred
export DATABASE_URL="postgresql://user:pass@host/db"
locksmith check migrations/

# Also acceptable
locksmith check migrations/ --database-url "$DATABASE_URL"

# Never do this (credentials in shell history)
locksmith check migrations/ --database-url "postgresql://user:pass@host/db"
```

4. **Minimum required permissions** — the database user needs only:

```sql
GRANT SELECT ON pg_stat_user_tables TO locksmith_user;
GRANT SELECT ON pg_indexes TO locksmith_user;
GRANT SELECT ON information_schema.columns TO locksmith_user;
```

5. **Read-only connection** — enforce in the connection string where possible:

```
postgresql://locksmith_user:pass@host/db?options=-c%20default_transaction_read_only%3Don
```

---

## SQL Injection Considerations

Locksmith parses SQL but never executes the migration files it analyzes. The `pg_query_go` parser is used in parsing mode only — it produces an AST but does not execute anything.

There is no injection risk in the static analysis path.

For the `--database-url` path (v1), Locksmith queries only system catalog tables using parameterized queries. It never interpolates content from the migration file into database queries.

---

## Sensitive Data in Migration Files

Locksmith may read migration files that contain sensitive information (e.g., hardcoded seed data with real credentials). Locksmith:
- Does not log migration file contents
- Does not transmit migration file contents
- Only logs the specific flagged SQL statements in output (the dangerous lines), not full file contents

Users should not store credentials in migration files. Locksmith does not enforce this but may add a rule for it in a future version.

---

## Responsible Disclosure

If you discover a security vulnerability in Locksmith:

1. **Do not** open a public GitHub issue
2. Email `security@[domain]` with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Your preferred credit/acknowledgment
3. You will receive a response within 48 hours
4. We target a patch release within 7 days for confirmed vulnerabilities

We follow a 90-day disclosure timeline. After 90 days, vulnerabilities may be disclosed publicly regardless of patch status.

---

## Security Checklist for Contributors

Before opening a PR:

- [ ] No hardcoded credentials or tokens
- [ ] No new network calls without explicit documentation
- [ ] No writing to disk beyond the current directory
- [ ] `govulncheck ./...` passes
- [ ] New dependencies are justified and reviewed
- [ ] Database-related code uses parameterized queries only
- [ ] Credentials are never logged at any log level

---

## Known Limitations

1. **No RBAC in MVP** — there is no concept of users or permissions. The tool runs with the permissions of the local user.
2. **No audit log in MVP** — ignore comments are not logged anywhere. Team audit logging is a v2 paid feature.
3. **Trust of migration files** — Locksmith assumes migration files are controlled by the repository owner. It does not sandbox the parsing process.
