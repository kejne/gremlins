# Project guidance

Gremlins is a Go mutation-testing tool. The engine finds AST-level mutations, executes tests against isolated copies, and reports which mutations survive.

## Tooling and checks

Use mise for project tools and tasks, and hk for local quality checks:

```sh
mise install
hk check --all
mise run check
```

`hk check --all` runs formatting, vet, lint, race-enabled tests, module-tidy validation, and workflow linting. Run focused Go tests while iterating, then the full check before handoff. Build and release tasks are available through `mise run build` and `mise run snapshot`.

## Architecture

- `internal/engine`: AST traversal and mutation execution orchestration.
- `internal/mutator`: mutation types and lifecycle/status definitions.
- `internal/coverage`: identifies mutations exercised by tests.
- `internal/execution`: runs test commands and captures outcomes.
- `internal/report`: console and machine-readable result formatting.
- `internal/configuration`: CLI/configuration settings.
- `cmd/gremlins`: CLI entry point and command wiring.

Keep changes focused, preserve mutation isolation, and add regression tests for behavior changes. Go tests should be deterministic and should not depend on network access.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues via `gh`. See `docs/agents/issue-tracker.md`.

### Triage labels

Use the five default triage labels. See `docs/agents/triage-labels.md`.

### Domain docs

Use the single-context layout. See `docs/agents/domain.md`.
