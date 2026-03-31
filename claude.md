# Claude AI Assistant Notes

This file keeps durable project preferences for Claude-oriented workflows. Avoid storing volatile facts here when they can drift from the codebase.

## Working Style

- Explore the code, tests, docs, and recent repo state before changing behavior.
- For non-trivial work, break execution into numbered tasks with explicit verification.
- Prefer tests first for risky behavior changes, CLI changes, compatibility fixes, and bug work.
- After the main fix, do a separate pass for edge cases and failure cleanup.
- Treat docs, changelog, and release automation as part of completion.

## Code Preferences

- Check cleanup errors for `Close()`, `Flush()`, and similar operations.
- Use `defer func()` when cleanup needs error handling or logging.
- Prefer `DBDUMP_MYSQL_PWD` in docs, but document compatibility behavior when other credential paths exist.

## Documentation Preferences

- Verify docs against source code, scripts, workflows, and tests rather than updating from memory.
- Prefer `docker compose` syntax in user-facing docs.
- Only reference files that actually exist.
- Remove stale or aspirational statements when behavior is not implemented.

## Review Preferences

When auditing changes, cover:

1. implementation correctness
2. test gaps and fragile tests
3. documentation accuracy
4. CI/CD and release drift
5. edge cases and recovery behavior

Prefer severity-ordered findings with file references.

## Commit and Release Preferences

- Prefer conventional commit prefixes: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`
- Keep commits atomic by intent
- Use plain version tags and release titles in the form `vX.Y.Z`
- Keep `CHANGELOG.md` aligned with the actual shipped changes before tagging

## Testing Preferences

- Test scripts should remain portable across macOS and Linux where practical.
- Prefer targeted verification first, then broader suite coverage before finishing.
- Do not treat a change as done if only the happy path was exercised.

## Reference

For the fuller tool-agnostic workflow, see `AGENTS.md`.
