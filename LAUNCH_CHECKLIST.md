# Launch Checklist

1. Tag `v1.0.0` and push the tag to GitHub. This triggers GoReleaser.
2. Verify the GitHub Release appears with all 5 binary archives and `checksums.txt`.
3. Test `brew install emartai/locksmith/locksmith` from the Homebrew tap.
4. Test the curl installer script on macOS and Linux.
5. Open the GitHub repository and verify `README.md` renders correctly.
6. Post to Hacker News: `Show HN: Locksmith - catch dangerous Postgres migrations before they reach production`
7. Post to `r/PostgreSQL` and `r/devops`.
8. Publish a terminal recording showing Locksmith catching a dangerous migration.
