# Locksmith — Project Context

## What Is Locksmith?

Locksmith is an open-source CLI tool and GitHub Action that analyzes Postgres migration files **before they run**, flags dangerous operations, estimates downtime, and suggests zero-downtime rewrites.

It is a **static analysis tool** — it reads SQL files and walks the AST. No database needs to be running for core functionality.

---

## Problem It Solves

Every backend team has a war story:
- An `ALTER TABLE` on a 50M-row table locked writes for 20 minutes
- A dropped column an old deploy was still reading
- A missing index that turned a background job into a full table scan

These mistakes are **expensive and preventable**. The best teams already do manual migration reviews — Locksmith automates what they do by hand and enforces it at the PR level.

---

## Target Users

**Primary**: Backend engineers and DevOps engineers at companies running Postgres in production

**Secondary**: Engineering managers who want visibility and policy enforcement across teams

**Persona 1 — The Careful Backend Dev**
- Uses Rails, Django, or writes raw SQL migrations
- Has personally experienced or witnessed a migration incident
- Wants a safety net before merging
- Will adopt a tool if it installs in under 2 minutes and has zero false positives on day one

**Persona 2 — The Eng Manager**
- Responsible for production reliability
- Wants org-wide enforcement, not per-developer configuration
- Will pay for team features once the free tool proves itself

---

## Product Positioning

**Category**: Developer tooling / database safety / CI tooling

**Tagline**: "Prevent dangerous Postgres migrations before they hit production"

**Differentiation**:
- Not just a linter — it explains why something is dangerous and shows the correct safe rewrite
- GitHub Action means it runs at PR time, not when someone remembers
- Uses the real Postgres C parser (`pg_query_go`) — no guessing at SQL syntax
- Postgres-first, deep coverage rather than shallow multi-database support

**Competitors**:
- Squawk (open source, unmaintained, limited rules)
- ORM-level warnings (shallow, framework-specific)
- Manual review (what most teams currently do)

---

## Tech Stack

| Layer | Choice | Reason |
|-------|--------|--------|
| Language | Go 1.22+ | Single binary, fast, great tooling |
| SQL Parser | pg_query_go | Wraps actual Postgres C parser |
| CLI Framework | cobra | Industry standard, powers kubectl/gh |
| Terminal Output | fatih/color | Colored ❌ ⚠️ ✅ output |
| Config | viper | YAML config, pairs with cobra |
| DB Driver (v1) | pgx | Best Postgres driver for Go |
| Release | GoReleaser | Cross-platform binary builds |
| Action | GitHub Actions | YAML-based, downloads binary |

---

## Project Structure

```
locksmith/
├── main.go
├── cmd/
│   └── check.go              # cobra CLI commands
├── internal/
│   ├── parser/
│   │   └── parser.go         # pg_query_go wrapper
│   ├── rules/
│   │   ├── engine.go         # Rule interface + runner
│   │   ├── add_column.go
│   │   ├── create_index.go
│   │   ├── foreign_key.go
│   │   ├── drop_object.go
│   │   ├── rename_object.go
│   │   ├── alter_column.go
│   │   ├── not_null.go
│   │   ├── truncate.go
│   │   ├── missing_fk_index.go
│   │   └── lock_timeout.go
│   └── reporter/
│       └── reporter.go       # Output formatting
├── testdata/
│   └── migrations/           # SQL fixtures for tests
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── action/
│   └── action.yml            # GitHub Action definition
├── locksmith.yml             # Example config
├── go.mod
├── go.sum
└── README.md
```

---

## MVP Scope (What Gets Built)

### In Scope
- 10 core rules (see rules.md)
- CLI: `locksmith check <file|dir>`
- `--database-url` flag scaffold (wired but not fully implemented in MVP)
- `-- locksmith:ignore RULE_NAME` comment support
- `locksmith.yml` config file support
- GitHub Action (YAML + binary download)
- GoReleaser config for binary distribution
- Homebrew tap scaffold
- Full test coverage for all 10 rules
- README with logo, install, usage, example output

### Out of Scope for MVP
- Web dashboard
- Paid tier / billing
- Live row count estimates (--database-url full implementation)
- ORM detection (Rails/Django migration format)
- Slack/PagerDuty integration
- Custom rules API
- MySQL / SQLite support
- SSO / SAML

---

## Business Model

**MVP**: 100% free and open source. Goal is GitHub stars and CI adoption.

**v1 Paid Tier** ($29/mo per org): Org-wide policy enforcement, centralized locksmith.yml

**v2 Pro** ($99/mo per org): History dashboard, Slack alerts, custom rules, multi-repo

**v3 Enterprise** ($500–$2,000/mo): SSO, audit logs, on-premise deployment, AI rewrites

---

## Distribution Strategy

1. **Open source repo** — primary top-of-funnel
2. **Technical blog posts** — "7 Postgres migration mistakes" style content
3. **Community presence** — Stack Overflow, Hacker News, Rails/Django communities
4. **Platform integrations** — Neon, Supabase, Railway native integrations (v1+)

---

## Success Metrics for MVP Launch

- 500 GitHub stars within 60 days
- 50 teams running Locksmith in CI
- 5 public mentions (tweets, blog posts, HN comments)
- Zero reported false positives on the 10 core rules

---

## Key Constraints

- **No Docker required** — for the builder or the user
- **No database required for MVP** — pure static analysis
- **Single binary** — install must work in under 60 seconds
- **Zero config to get started** — `locksmith check migrations/` must work with no config file
- **Escape hatch always available** — `locksmith:ignore` prevents tool rejection

---

## Repository

- GitHub: `github.com/emartai/locksmith`
- License: MIT
- Language badge, CI badge, GoReleaser badge in README
