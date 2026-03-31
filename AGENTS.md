# Agent Workflow Guide

This repository benefits from a consistent agent workflow. Use these rules for coding agents, review agents, and release agents.

## Core Workflow

For non-trivial work, follow this sequence:

1. Explore the current code, tests, docs, and recent repo state before proposing changes.
2. Clarify missing constraints or success criteria when they materially affect the solution.
3. If the change is architectural or ambiguous, propose 2-3 approaches with trade-offs and recommend one.
4. Break implementation into numbered tasks with explicit verification steps.
5. Prefer tests first for risky behavior changes, parser changes, CLI behavior changes, and compatibility fixes.
6. After the main fix, do a dedicated edge-case pass rather than assuming the happy path is enough.
7. Run the narrowest relevant checks first, then the broader suite before finishing.
8. Treat docs, release notes, and automation as part of done, not post-work cleanup.

## Prompting Patterns

Use prompts shaped like these:

- "Explore current behavior first. Identify code, tests, docs, and release impact before changing anything."
- "Propose 2-3 approaches with trade-offs, then implement the recommended path in numbered tasks."
- "Write failing tests for the bug or regression first, then implement the fix and run the relevant suite."
- "Audit docs against source code and classify each area as CORRECT, INCORRECT, or MISSING."
- "Analyze the git changes and create logical, atomic commits using conventional commits."

## Review Expectations

Reviews should cover more than implementation correctness:

1. Behavior and regression risk
2. Test gaps and weak or overly coupled tests
3. Documentation accuracy
4. Release and CI/CD drift
5. Edge cases, cleanup behavior, and failure modes

For review-heavy tasks, prefer findings ordered by severity with file references.

## Testing Guidance

When changing behavior, consider all of these layers:

- Unit tests for isolated logic
- CLI or command-path tests for user-facing behavior
- Integration tests for real MySQL and MariaDB behavior
- Edge-case tests for identifiers, exclusions, interrupts, and compatibility fallbacks
- Restore-fidelity checks, not just dump-file existence

Avoid declaring a change complete if only the happy path was exercised.

## Documentation Guidance

Do not update docs from memory. Verify commands, defaults, versions, supported behaviors, and release/install steps from the code, scripts, workflows, and tests.

Prefer source-audit wording over vague doc tasks:

- Good: "Verify README install examples against release assets and CLI flags."
- Weak: "Clean up the docs."

## Release Guidance

Before tagging a release, verify:

1. `CHANGELOG.md` reflects the actual shipped changes
2. README and guide examples still work as written
3. GitHub workflow action versions and release docs are current
4. Integration coverage still matches documented support
5. The working tree is clean and the branch is in sync with remote

## Commit Guidance

- Use conventional commits where practical: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`
- Prefer atomic commits grouped by intent
- Do not mix release metadata, feature work, and unrelated cleanup unless they are tightly coupled
