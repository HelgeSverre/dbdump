# Claude AI Assistant Notes

This file contains preferences and conventions for working with Claude AI on this project.

## Release Management

### GitHub Release Naming
- **Release titles**: Use simple version format `vX.X.X` (e.g., `v1.0.1`)
- **DO NOT** add prefixes like "Release v1.0.1" or suffixes
- **DO NOT** add descriptive text in the title (save for the release notes body)

### Release Notes Format
- Extract relevant sections from CHANGELOG.md
- Format as clear Markdown with proper headings
- Include "Full Changelog" link comparing previous version
- Organize by categories: Security, Fixed, Added, Changed, Performance, Documentation

Example structure:
```markdown
## Fixed
- Clear, concise bullet points

## Added
- New features

---

**Full Changelog**: https://github.com/user/repo/compare/v1.0.0...v1.0.1
```

## Code Style

### Error Handling
- All `Close()`, `Flush()`, and similar cleanup operations must check errors
- Use `defer func()` pattern with error checking for cleanup operations
- Log warnings to stderr for non-critical cleanup failures

Example:
```go
defer func() {
    if err := db.Close(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to close database connection: %v\n", err)
    }
}()
```

### Environment Variables
- Prefer `DBDUMP_MYSQL_PWD` over `MYSQL_PWD` in documentation
- Document both for backward compatibility
- Always emphasize environment variables over command-line password flags

## Documentation

### Commands
- Use `docker compose` (not `docker-compose`) for newer Docker CLI
- Use Go `1.23+` as minimum version requirement
- Keep dates in format `YYYY-MM-DD` (e.g., `2024-10-28`)

### File References
- Only reference files that actually exist
- Remove links to planned/future documentation files
- Keep documentation in sync with actual codebase

## Git Workflow

### Commit Messages
- Follow conventional commits format: `type: description`
- Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`
- Include detailed body for significant changes
- Always add Claude Code attribution footer

### CI/CD Considerations
- Integration tests detect CI environment via `CI` env var
- Tests skip Docker Compose startup when running in GitHub Actions
- Use portable commands (e.g., `ls -l` for permissions instead of `stat`)

## Testing

### Test Scripts
- Must work on both macOS and Linux
- Use bash 3.2 compatible syntax (macOS default)
- Avoid platform-specific commands or provide fallbacks

### Integration Tests
- Run against MySQL 5.7, 8.0, 8.4, and MariaDB 10.11
- Security tests verify password hiding and file permissions
- Data integrity tests ensure triggers, procedures, and restoration work

## Project Preferences

### Changelog Maintenance
- Keep [Unreleased] section for work in progress
- Move to versioned section when releasing
- Include both functional and documentation changes
- Use semantic versioning (MAJOR.MINOR.PATCH)

---

**Last Updated**: 2024-10-28
**Maintained by**: Helge Sverre with Claude Code assistance
