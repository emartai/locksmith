# Locksmith — Launch Guide

## Pre-Launch Checklist (Complete Before Tagging v1.0.0)

### Code
- [ ] All 25 prompts executed and verified
- [ ] `go test ./... -race` passes with zero failures
- [ ] `go vet ./...` zero warnings
- [ ] `gofmt -l .` zero output
- [ ] Binary size under 20MB (`go build -ldflags="-s -w"`)
- [ ] All 10 rules implemented and tested
- [ ] Exit codes correct (0/1/2)
- [ ] `--format json` produces valid JSON
- [ ] `--no-color` strips ANSI codes
- [ ] Ignore comments work for all 10 rules

### Repository
- [ ] Repository is public on GitHub
- [ ] LICENSE file is MIT
- [ ] .gitignore is complete
- [ ] No secrets, credentials, or personal data in git history
- [ ] go.sum is committed
- [ ] README renders correctly on GitHub
- [ ] All README links work

### Release Infrastructure
- [ ] .goreleaser.yml is valid (`goreleaser check`)
- [ ] GitHub Actions CI is green
- [ ] GoReleaser release workflow is configured
- [ ] GPG key configured for release signing (optional but recommended)
- [ ] Homebrew tap repository created at emartai/homebrew-locksmith

### Documentation
- [ ] README complete (all 12 sections)
- [ ] CONTRIBUTING.md complete
- [ ] rules.md accurate for all 10 rules
- [ ] context.md, design.md, architecture.md, security.md in /docs

---

## Launch Day Sequence

### Step 1: Tag and Release (Day 0)

```bash
git tag v1.0.0
git push origin v1.0.0
```

Watch the release workflow:
- Go to GitHub Actions → release workflow
- Confirm GoReleaser completes without error
- Confirm GitHub Release appears with 5 binary archives + checksums.txt

Verify each binary works:
```bash
# Download the Linux amd64 binary and test it
curl -sSL https://github.com/emartai/locksmith/releases/download/v1.0.0/locksmith_linux_amd64.tar.gz | tar -xz
./locksmith --version
./locksmith check testdata/migrations/dangerous/001_add_column_with_default.sql
```

### Step 2: Homebrew Tap

```bash
# Verify Homebrew formula was auto-created by GoReleaser
brew tap emartai/locksmith
brew install locksmith
locksmith --version
```

### Step 3: Test the One-Line Installer

```bash
curl -sSL https://locksmith.dev/install.sh | sh
locksmith --version
```

If locksmith.dev is not yet live, update install.sh to point directly to the GitHub Release URL.

### Step 4: GitHub Repository Setup

- Add repository topics: `postgres`, `postgresql`, `migrations`, `database`, `devops`, `cli`, `github-actions`, `golang`
- Add a repository description: "Prevent dangerous Postgres migrations before they hit production"
- Pin the repository if it's under an organization
- Enable GitHub Discussions (for community questions)
- Enable GitHub Sponsors if you want to accept donations

---

## Launch Posts

### Hacker News (Show HN)

**Title:**
```
Show HN: Locksmith – catch dangerous Postgres migrations before they reach prod
```

**Body:**
```
I built Locksmith after watching a team's ALTER TABLE on a 40M-row table lock production writes for 18 minutes.

It's a CLI tool + GitHub Action that reads your migration files, detects dangerous operations, and blocks the PR if it finds them.

Example: locksmith check migrations/ catches CREATE INDEX without CONCURRENTLY, ADD COLUMN with NOT NULL DEFAULT, foreign keys added without NOT VALID, and 7 other patterns that have caused real outages.

For each issue it explains why it's dangerous, what lock it acquires, and shows the correct zero-downtime rewrite.

It uses the actual Postgres C parser (pg_query_go) — not regex.

GitHub: https://github.com/emartai/locksmith

The 10 rules cover about 80% of migration incidents I've seen in production. Would love feedback on what's missing.
```

**When to post:** Tuesday–Thursday, 9–11am US Eastern (when HN traffic peaks)

---

### Reddit r/PostgreSQL

**Title:**
```
I built a CLI that catches dangerous Postgres migrations before you run them (open source)
```

**Body:**
```
After dealing with a production incident from a missing CONCURRENTLY on CREATE INDEX, I built Locksmith.

It's a CLI + GitHub Action that statically analyzes your migration files and flags operations that will cause downtime:
- CREATE INDEX without CONCURRENTLY (takes SHARE lock)
- ADD COLUMN with NOT NULL DEFAULT (full table rewrite on PG10)
- ADD FOREIGN KEY without NOT VALID (validates all rows, blocks writes)
- DROP COLUMN while app code still references it
- RENAME COLUMN (silently breaks existing queries)
- ...and 5 more

For each flag: explains the danger, the lock type acquired, and shows the safe rewrite.

GitHub Action setup is one YAML block — runs on every PR that touches a migration file.

Open source, MIT license. Uses pg_query_go (actual Postgres parser, not regex).

https://github.com/emartai/locksmith

Would appreciate feedback on rules I've missed.
```

---

### Reddit r/devops

**Title:**
```
Open source: CLI tool to prevent Postgres migration downtime in CI
```

Similar to r/PostgreSQL post but emphasize the CI/GitHub Action angle.

---

### Twitter/X Thread

Tweet 1:
```
I built Locksmith — an open source CLI that analyzes Postgres migrations and blocks dangerous ones before merge.

It catches: missing CONCURRENTLY on CREATE INDEX, ADD COLUMN with NOT NULL DEFAULT, FK without NOT VALID, DROP COLUMN while app still uses it.

github.com/emartai/locksmith
```

Tweet 2:
```
The GitHub Action is one YAML block:

[paste the GitHub Action YAML]

Every PR that touches migrations/ gets auto-checked. Fails CI if dangerous ops detected.
```

Tweet 3:
```
It uses pg_query_go — the actual Postgres C parser. Not regex.

So it understands the real AST, not a guess at SQL syntax.

For each flag: explains the lock type, why it's dangerous, and shows the correct zero-downtime rewrite.
```

---

### Dev.to / Hashnode Blog Post

Write a technical blog post (800–1200 words):

**Title:** "The 5 Postgres migration mistakes that have taken down production (and how to catch them automatically)"

**Structure:**
1. The incident that inspired this (brief story)
2. Rule 1: CREATE INDEX without CONCURRENTLY (explain the lock, show the fix)
3. Rule 2: ADD COLUMN with NOT NULL DEFAULT (explain the rewrite, PG11 nuance)
4. Rule 3: ADD FOREIGN KEY without NOT VALID (explain validation scan)
5. Rule 4: DROP COLUMN while app still reads it (blue-green column removal)
6. Rule 5: Rename column (silent breakage)
7. "We automated all of this with Locksmith" — brief intro, GitHub link, GitHub Action snippet
8. Call to action: star the repo, add the Action

This post does double duty: it ranks on Google for "postgres migration safety" and drives stars.

**Cross-post:** dev.to, Hashnode, your personal blog

---

## Post-Launch Week 1 Tasks

### Day 1–2
- Monitor GitHub issues — respond to every issue within 24 hours
- Monitor HN/Reddit comments — respond substantively, not just "thanks"
- Fix any installation bugs immediately (these block adoption)

### Day 3–7
- Add any missing rules people request (if clearly correct, add quickly)
- Write a short follow-up tweet showing a real example caught in the wild
- Reach out personally to 5–10 engineers you know and ask them to try it

### First Week Metrics to Track
- GitHub stars
- GitHub forks
- Clone count (available in GitHub Insights → Traffic)
- Issues opened
- Any PRs from external contributors (great signal)

---

## First 30 Days — Growth Targets

| Metric | Target |
|--------|--------|
| GitHub Stars | 500 |
| Forks | 50 |
| Teams using in CI | 50 |
| Blog post views | 2,000 |
| HN points (if posted) | 100+ |
| External contributors | 2+ |

---

## When to Introduce Paid Features

Don't introduce paid features until:
1. The open source tool has 500+ GitHub stars
2. You have 5+ teams actively using it in CI
3. You've had at least 3 conversations where someone asked "can we use this org-wide?"

The third signal is the most important — that's your product-market fit signal for the paid tier.

The first paid feature should be **org-wide policy enforcement** (a single `locksmith.yml` that applies across all repos in a GitHub organization). Price: $29/month per organization.

---

## Community Building

### GitHub Discussions
Enable Discussions with these categories:
- Q&A (for usage questions)
- Ideas (for rule suggestions)
- Show & Tell (for teams sharing how they use it)

### Response Time Targets
- Issues: 24 hours
- PRs: 48 hours
- Discussions: 48 hours

### First External Contributor Guide
When you get the first external PR:
- Review it thoroughly and leave detailed, kind feedback
- Merge it if correct — even if you'd have written it differently
- Thank them publicly
- Add them to CONTRIBUTORS.md

The first external contributor often becomes your first team paying customer.
