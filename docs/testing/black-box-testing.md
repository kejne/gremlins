# Black-box testing policy

Gremlins tests should exercise a cohesive module through its supported interface and assert observable behavior. Black-box testing is not CLI-only testing: focused AST, process, and workspace tests remain appropriate when those are the behavior under test. An external Go test package is a useful access boundary, but its package declaration alone does not make private or implementation-coupled assertions behavioral.

Do not export production details solely for tests or add private-access bridges. If a useful scenario cannot be expressed through the supported module interface, reconsider whether it represents a supported behavior and raise a genuine module-boundary issue rather than adding a test-only escape hatch.

## Same-package enforcement and ratchet

The pinned golangci-lint `testpackage` linter is enabled with its implicit filename and package exemptions disabled (`skip-regexp: '^$'` and an explicit non-matching allowed-package name). Exact per-file exclusions temporarily grandfather the same-package files below; they apply only to `testpackage`. New same-package test files must fail lint. An equivalent external-package test should pass this rule. The rule does not detect imports into implementation subpackages or assertions coupled to internal semantics.

These are file-level allowances. Lint cannot distinguish an old test from a new white-box test added inside an allowed file. Review must reject new white-box tests in every grandfathered file. Remove each allowance with its migration. No allowance is an approved lasting exception; any lasting exception requires an explicit policy decision and specific rationale.

The migration history, checkout drift, and mapping from planned to implemented paths are recorded in the [enforcement spec](../specs/enforced-black-box-testing.md). The table below is the current nine-file allowance baseline.

| Current same-package file | Temporary rationale |
| --- | --- |
| `cmd/gremlins_test.go` | CLI unit tests exercise flag wiring and command behavior before CLI boundaries are separated. |
| `cmd/unleash_test.go` | CLI unleash tests share private command setup and fixture helpers. |
| `cmd/internal/flags/flags_test.go` | Flag parser tests use same-package declarations during command modularization. |
| `internal/configuration/configuration_test.go` | Configuration tests cover unexported parsing and defaults. |
| `internal/exclusion/rules_test.go` | Rule tests exercise private exclusion-rule construction details. |
| `internal/execution/cancellation_test.go` | Cancellation tests use package-private execution lifecycle fixtures. |
| `internal/execution/executor_test.go` | Executor tests currently share internal execution setup. |
| `internal/execution/executor_test_helpers_test.go` | Shared executor test helpers are package-private and stay with the existing tests. |
| `internal/results/results_test.go` | Result value tests currently construct package-private result values. |

These nine exact file allowances apply only to `testpackage`; there are no blanket package or directory exemptions.

## Semantic coupling inventory

`testpackage` enforces package declaration only. External-package syntax does not prevent importing a module's implementation subpackages or asserting incidental internals. The current known implementation-subpackage import is `internal/report/report_test.go` importing `internal/report/internal`; it remains reporting-module follow-up work, not a lint allowance or accepted black-box test. Other representative semantic coupling includes ambient global configuration mutation in `internal/engine/engine_test.go` and `internal/execution/executor_test.go`, and direct Viper setup in `internal/coverage/coverage_test.go`; address these with the configuration/orchestration, execution, and coverage owners respectively as those modules migrate. Reporting tests also manipulate global Viper state. Review should scrutinize external-package tests for these and other implementation-coupled expectations. The inventory is representative, not an assertion that these are the only semantic issues.

Keep focused AST transformation, real subprocess, and filesystem/workspace tests where they protect observable contracts; black-box policy does not require replacing useful test levels or expanding all scenarios to CLI workflows.

## Validation

`sh scripts/check-testpackage-regression.sh` uses a disposable checkout to show that same-package tests fail even with `internal_test.go` filenames and the `main` package name, external-package equivalents pass, unrelated govet diagnostics still report in a grandfathered file, and removing one exact allowance restores enforcement. The script leaves no fixture or violation in the working tree. It runs in the existing `go-lint` step after repository lint, which also triggers for `.golangci.yml`, `hk.pkl`, `mise.toml`, and the demonstration script; CI continues to call `hk check --all`.

At initial enforcement validation, `hk check --all` passed including race-enabled tests and the existing 80% coverage threshold (observed coverage: 84.41%). The check runner's config-only plan for `.golangci.yml` selects both `go-lint` and the regression demonstration.
