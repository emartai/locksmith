# Locksmith — Design Guidelines

## Design Philosophy

Locksmith is a developer-first CLI tool. Every design decision must reinforce:

- **Minimal** — nothing decorative, everything functional
- **Technical** — looks like infrastructure tooling, not a startup product
- **Trustworthy** — engineers adopt tools they trust; trust comes from precision
- **Terminal-native** — CLI output is the primary UI; it must be clean and scannable
- **Database-focused** — the visual language references tables, grids, structure

### Anti-patterns (never do these)
- Gradients of any kind
- 3D effects or shadows
- Playful illustrations or mascots
- Bright or saturated color palettes
- Marketing copy tone in the README or CLI output
- Serif fonts anywhere

---

## Logo

### Concept
**Database table grid with a highlighted protected cell**

- Grid = database table structure
- Highlighted cell = the migration being protected
- No text decoration, no lock icon (avoids cliché)

### Primary Logo (240×64)

```svg
<svg width="240" height="64" viewBox="0 0 240 64" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect x="8" y="18" width="32" height="28" rx="2" stroke="#0F172A" stroke-width="2"/>
  <line x1="8" y1="32" x2="40" y2="32" stroke="#0F172A" stroke-width="2"/>
  <line x1="24" y1="18" x2="24" y2="46" stroke="#0F172A" stroke-width="2"/>
  <rect x="26" y="20" width="12" height="10" rx="1.5" fill="#0F172A"/>
  <text x="56" y="38" font-family="JetBrains Mono, monospace" font-size="20" fill="#0F172A">locksmith</text>
</svg>
```

### Icon Only (32×32) — for favicon, GitHub avatar

```svg
<svg width="32" height="32" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect x="2" y="6" width="28" height="20" rx="2" stroke="#0F172A" stroke-width="2"/>
  <line x1="2" y1="16" x2="30" y2="16" stroke="#0F172A" stroke-width="2"/>
  <line x1="16" y1="6" x2="16" y2="26" stroke="#0F172A" stroke-width="2"/>
  <rect x="18" y="8" width="8" height="6" rx="1" fill="#0F172A"/>
</svg>
```

### Dark Mode Logo Variants

Replace `#0F172A` strokes with `#E5E7EB`.
Replace filled rect with `#E5E7EB`.

### Logo Rules
- Never change proportions
- Never add color to the grid lines
- Never add drop shadows
- Minimum clear space: 8px on all sides
- Do not stretch or squish

---

## Color Palette

| Token | Hex | Usage |
|-------|-----|-------|
| `--color-text` | `#0F172A` | Primary text, strokes, logo |
| `--color-danger` | `#EF4444` | ❌ DANGEROUS rule output |
| `--color-warning` | `#F59E0B` | ⚠️ WARNING rule output |
| `--color-success` | `#22C55E` | ✅ PASSED rule output |
| `--color-accent` | `#3B82F6` | Links, highlights (use sparingly) |
| `--color-bg-light` | `#FFFFFF` | Light mode background |
| `--color-bg-dark` | `#020617` | Dark mode background |
| `--color-muted` | `#64748B` | Secondary text, line numbers |
| `--color-border` | `#E2E8F0` | Dividers (docs/web only) |

**Rule**: Never introduce colors outside this palette. No purples, no teals, no pinks.

---

## Typography

### Fonts

| Role | Font | Fallback |
|------|------|---------|
| UI / Documentation | `IBM Plex Sans` | `system-ui, sans-serif` |
| Code / CLI / Commands | `JetBrains Mono` | `Fira Code, monospace` |

> **Change from original**: Replaced `Inter` with `IBM Plex Sans`. IBM Plex Sans has a more technical, infrastructure-tool character that matches Locksmith's positioning. Inter reads as "startup SaaS." IBM Plex Sans reads as "serious engineering tool."

### Font Size Scale

```
xs:   11px  — CLI secondary labels
sm:   13px  — CLI body, code comments  
base: 15px  — CLI primary output
lg:   18px  — README section headers
xl:   24px  — README title
```

### Rules
- All CLI output uses `JetBrains Mono`
- All command examples use `JetBrains Mono`
- Never use bold in CLI output except for rule names
- Never use italic in CLI output

---

## CLI Output Style

### Format

```
❌ DANGEROUS   Line 4
   ADD COLUMN with non-null default
   
   Why:   Rewrites entire table on Postgres < 11
   Lock:  ACCESS EXCLUSIVE — blocks all reads and writes
   Fix:   Add column nullable → backfill → add NOT NULL

⚠️  WARNING    Line 9
   Foreign key without NOT VALID
   
   Why:   Validates all existing rows on creation
   Lock:  SHARE ROW EXCLUSIVE — blocks writes
   Fix:   ADD CONSTRAINT ... NOT VALID, then VALIDATE CONSTRAINT separately

✅ PASSED      Line 14
   CREATE INDEX CONCURRENTLY
```

### Output Rules

1. Use exactly these emojis: `❌` `⚠️` `✅`
2. Severity label is always uppercase: `DANGEROUS`, `WARNING`, `PASSED`
3. Line number is always shown
4. `Why:` and `Fix:` labels are always present for non-passing rules
5. Add `Lock:` label when the lock type is known
6. Blank line between each rule block
7. No ASCII art borders or boxes
8. Consistent 3-space indent for body content
9. Summary line at the end: `2 issues found. Migration blocked.`
10. Short lines — wrap at 80 characters

### Exit Codes
```
0  — all rules passed
1  — one or more DANGEROUS rules triggered
2  — one or more WARNING rules triggered (configurable)
```

---

## README Structure

```
1. Logo (SVG inline)
2. Tagline: "Prevent dangerous Postgres migrations before they hit production"
3. Badges: CI, Go version, License, Latest release
4. Install (3 methods: brew, curl, go install)
5. Quick start (30-second example)
6. Example output (real terminal screenshot or code block)
7. Rules table (all 10 rules with severity)
8. GitHub Action setup
9. Config file reference
10. Ignore comments
11. Contributing
12. License
```

### README Tone
- Short sentences
- No exclamation points
- No "amazing", "powerful", "incredible"
- Technical and direct
- Write like the `gh` CLI docs or Terraform docs

---

## GitHub README Badges

```markdown
![CI](https://github.com/emartai/locksmith/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/github/go-mod/go-version/emartai/locksmith)
![License](https://img.shields.io/badge/license-MIT-blue)
![Latest Release](https://img.shields.io/github/v/release/emartai/locksmith)
```

---

## Terminal Color Mapping (Go)

```go
// Using fatih/color
var (
    dangerous = color.New(color.FgRed, color.Bold)
    warning   = color.New(color.FgYellow, color.Bold)
    success   = color.New(color.FgGreen, color.Bold)
    muted     = color.New(color.FgHiBlack)
    label     = color.New(color.Bold)
)
```

---

## Visual Tone Reference

### Locksmith should look like:
- `kubectl` — precise, structured output
- `terraform plan` — clear danger/change indicators
- `gh` CLI — clean, no noise
- `pgcli` — database-native feel

### Locksmith should NOT look like:
- Vercel CLI (too polished, too startup)
- Prisma Studio (web app energy)
- Any tool with a mascot
- Any tool with a gradient in its logo

---

## Dark Mode

Dark mode applies to documentation site and README rendering only. CLI output respects the terminal's own color scheme.

| Token | Dark Value |
|-------|-----------|
| Background | `#020617` |
| Text | `#E5E7EB` |
| Logo stroke | `#E5E7EB` |
| Logo cell fill | `#E5E7EB` |
| Muted | `#94A3B8` |
| Border | `#1E293B` |

---

## File Naming Conventions

```
locksmith.yml        — config file (not .locksmith, not locksmith.config.yml)
.github/workflows/   — standard GitHub Actions location
migrations/          — default migration directory Locksmith looks for
```

---

## Changelog

| Version | Change |
|---------|--------|
| 1.1 | Replaced Inter with IBM Plex Sans for more technical character |
| 1.1 | Added `Lock:` field to CLI output format for lock type clarity |
| 1.1 | Added exit code spec |
| 1.1 | Added terminal color Go mapping |
| 1.1 | Expanded README structure to 12 sections |
| 1.0 | Initial design guidelines |
