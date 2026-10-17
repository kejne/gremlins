# Project guidance

Gremlins is a Go mutation-testing tool. The mutation module discovers and transforms AST-level mutations; the engine orchestrates testing them against isolated copies and reporting which mutations survive.

## Tooling and checks

Use `hk check` by default; run `hk check --all` before handoff. Install tools with `mise install`. For other project tasks, run `mise tasks ls`.

## Architecture

- `internal/mutation`: AST discovery and source transformations.
- `internal/engine`: mutation workflow orchestration.
- `internal/mutator`: mutation rules/types and lifecycle/status definitions.
- `internal/coverage`: identifies mutations exercised by tests.
- `internal/execution`: runs test commands and captures outcomes.
- `internal/report`: console and machine-readable result formatting.
- `internal/configuration`: CLI/configuration settings.
- `cmd/gremlins`: CLI entry point and command wiring.

Keep changes focused, preserve mutation isolation, and add regression tests for behavior changes. Go tests should be deterministic and should not depend on network access.

## Agent skills

### Issue tracking

GitHub Issues hold reports, specs, and high-level work; Beads holds implementation tickets. See `docs/agents/issue-tracker.md`. When working with implementation tickets, read `.agents/skills/beads/SKILL.md`.

### Triage labels

Use the five default triage labels. See `docs/agents/triage-labels.md`.

### Domain docs

Use the single-context layout. See `docs/agents/domain.md`.
