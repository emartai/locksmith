# Locksmith GitHub Action

Analyze Postgres migration files in CI before they are merged.

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `path` | No | `migrations/` | Path to a migration file or directory. |
| `severity` | No | `dangerous` | Minimum severity to fail on. Supported values: `dangerous`, `warning`. |
| `format` | No | `text` | Output format. Supported values: `text`, `json`. |
| `output` | No | `""` | Optional file path for JSON output. |
| `upload_json_artifact` | No | `false` | Upload the JSON report as a workflow artifact when `format: json`. |

## Example

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

## What It Does

1. Downloads the latest Locksmith release, or the version from `LOCKSMITH_VERSION` if set.
2. Verifies the release checksum.
3. Runs `locksmith check` against the configured path.

## Output

The action prints the same CLI output as local `locksmith check` runs and automatically writes a Markdown summary to the GitHub Actions job summary when running in GitHub Actions.

- `dangerous` findings fail the job.
- `warning` findings fail the job when `severity: warning` is selected.
- clean migrations pass with zero findings.
