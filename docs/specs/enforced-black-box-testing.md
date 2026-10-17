## Problem Statement

Gremlins needs tests that protect observable behavior without coupling contributors and coding agents to private helpers, internal state, scheduling protocols, or implementation layout. Same-package Go tests currently allow direct access to private implementation, and external-package syntax alone does not prevent tests from reaching into another module's implementation subpackages.

Maintainers also lack an automatic check that prevents the repository from accumulating more same-package tests. A wholesale rewrite would entangle this improvement with the broader modularity migration. The next step must establish enforceable boundaries and migrate a contained set of existing tests without weakening regression protection or exporting implementation details merely to satisfy lint.

## Solution

Adopt module-level black-box testing as the policy: exercise a cohesive module through its supported interface and assert externally observable behavior. Black-box testing does not mean every test must invoke the CLI, nor does an external test package automatically make an assertion behavioral.

Enable the existing `testpackage` linter through the repository's current lint, check-runner, and CI integration. Remove implicit exemptions and use explicit, explained, temporary per-file allowances for unmigrated same-package tests. Convert the two engine helper-test files and two diff-test files in this increment, leaving five known same-package files explicitly tracked. Review prevents new white-box tests inside grandfathered files until their allowances disappear.

Preserve meaningful behavioral scenarios rather than the exact number or structure of existing tests. Do not introduce test-only exports or private-access bridges. Inventory external tests that reach into implementation subpackages separately, then remove those violations as their owning modules migrate.

## User Stories

1. As a maintainer, I want black-box testing defined at the module boundary, so that contributors do not mistake the policy for CLI-only testing.
2. As a test author, I want to use supported interfaces, so that tests survive changes to private implementation.
3. As a contributor, I want tests to assert observable results, so that a refactor does not require rewriting tests that protect unchanged behavior.
4. As a coding agent, I want a clear distinction between external-package syntax and behavioral black-box testing, so that passing lint does not create false confidence.
5. As a maintainer, I want new same-package test files rejected automatically, so that implementation coupling does not grow unnoticed.
6. As a contributor, I want to reuse the installed linter, so that enforcement does not require another utility or maintenance surface.
7. As a maintainer, I want implicit linter exemptions disabled, so that filenames or package names cannot silently bypass the policy.
8. As a reviewer, I want existing exceptions listed individually, so that legacy debt is visible and bounded.
9. As a maintainer, I want each exception explained as temporary, so that contributors do not treat it as an approved testing style.
10. As a contributor, I want the allowance removed when its test migrates, so that completed migrations tighten enforcement immediately.
11. As a maintainer, I want no blanket package or directory exemptions, so that unrelated new tests remain subject to enforcement.
12. As a reviewer, I want additions inside grandfathered files checked for new white-box tests, so that the file-level ratchet's known limitation is covered.
13. As a contributor, I want the same checks locally and in CI, so that failures are reproducible before publishing changes.
14. As a maintainer, I want lint-configuration-only changes to trigger lint, so that configuration changes cannot silently disable enforcement.
15. As a coding agent, I want a contained first migration slice, so that this improvement can land independently of a broad architecture refactor.
16. As a maintainer, I want engine helper behavior tested through existing execution and mutation interfaces, so that private helper access is unnecessary.
17. As a CLI user, I want build tags, CPU settings, timeout construction, and integration/package selection preserved, so that test execution remains compatible.
18. As a CLI user, I want original and mutated snippets preserved, so that mutation reports remain useful after test restructuring.
19. As a maintainer, I want diff behavior tested through supported construction and changed-location queries, so that private parser decomposition can change safely.
20. As a CLI user, I want changed-line selection preserved, so that the migration does not change which mutations are selected.
21. As a test author, I want controlled command output and errors through existing seams, so that diff tests remain deterministic without real repository or network dependencies.
22. As a maintainer, I want malformed diff and command-failure behavior protected, so that rewriting tests does not erase error coverage.
23. As a contributor, I want redundant private-helper cases replaced or removed when their behavior is covered elsewhere, so that the suite does not preserve accidental structure.
24. As a reviewer, I want meaningful regression scenarios retained, so that fewer tests do not disguise weaker behavioral protection.
25. As a maintainer, I want the existing coverage gate retained, so that enforcement is not achieved by relaxing repository quality checks.
26. As a module owner, I want no implementation details exported solely for tests, so that test migration does not expand the supported interface accidentally.
27. As a reviewer, I want test-only export bridges prohibited, so that external-package tests cannot recover private access through a workaround.
28. As a contributor, I want awkward tests to prompt reconsideration of the behavior or module boundary, so that poor seams are not patched with API pollution.
29. As a maintainer, I want implementation-subpackage imports identified as black-box violations, so that package naming cannot disguise internal coupling.
30. As a coding agent, I want those semantic violations inventoried separately from lint allowances, so that mechanically enforced rules and review obligations remain distinguishable.
31. As a reporting contributor, I want implementation-subpackage coupling removed with its owning module's migration, so that this initial slice does not become an unrelated reporting rewrite.
32. As a maintainer, I want focused AST, real process, and workspace tests retained, so that black-box policy does not eliminate useful testing levels.
33. As a CLI user, I want mutation outcomes, source isolation, restoration, and error semantics unchanged, so that this remains a testing-policy change rather than a product change.
34. As a contributor, I want existing helper-process and filesystem fixture patterns reused, so that tests remain deterministic and independent of network access.
35. As a maintainer, I want positive and negative enforcement demonstrations, so that the lint configuration is shown to catch representative violations.
36. As a coding agent, I want the remaining migration debt documented, so that follow-up work can continue without reconstructing this conversation.
37. As a reviewer, I want race-enabled tests and the full repository checks passing before handoff, so that lint and test restructuring do not introduce regressions.

## Implementation Decisions

- Follow the accepted modularity decision and the related architecture spec in issue #1. Preserve task-oriented module ownership and existing CLI behavior; do not impose a new architectural layout.
- Define black-box tests as tests of a cohesive module's supported interface and observable behavior. Private state, internal call sequences, worker identities, channel traffic, and implementation layout are not test contracts.
- Require external Go test packages as a mechanically enforceable access boundary, while explicitly documenting that this is necessary policy enforcement rather than proof of behavioral test quality.
- Use `testpackage` from the already pinned and installed `golangci-lint`; do not implement a custom repository linter or per-test baseline checker for this increment.
- Explicitly configure the installed linter so that implicit filename and package exemptions do not undermine the policy. Do not rely on defaults inferred from another linter version.
- Represent remaining same-package debt with exact, explained per-file exclusions limited to this rule. Do not exclude entire packages, directories, all tests, or unrelated linters. Remove allowances in the same change that migrates their tests; do not suppress individual violations with unexplained directives.
- The reviewed endpoint for this increment is five known same-package files remaining after the four selected files migrate. The eventual target is no migration allowances. Any necessary lasting exception requires a specific rationale and an explicit policy decision, not a silently retained migration allowance.
- Recognize the ratchet's granularity: it rejects new violating files but cannot reject a new white-box test added inside an already allowed file. Review enforces the no-new-white-box-tests policy within those files until they migrate.
- Integrate enforcement into the existing Go lint step and CI path rather than a separate CI-only job. Trigger lint for relevant Go, linter-configuration, and check/tooling-configuration changes, including configuration-only changes.
- Migrate the two engine private-helper test files and the two diff test files. Re-express meaningful scenarios through existing supported execution, mutation, and diff interfaces; a package declaration change alone is insufficient for private-helper assertions.
- Preserve supported production interfaces. Do not export helpers or state solely to move tests, add test-only export bridges, or use private-access workarounds. If a scenario cannot be expressed without such changes, reconsider whether it describes supported behavior and surface a genuine boundary problem rather than adding an escape hatch.
- Preserve meaningful behavioral protection, not a one-to-one correspondence with helper cases or test counts. Replace or remove redundant tests when the same observable contract is protected elsewhere; do not weaken the coverage threshold to make migration pass.
- Treat imports into a module's implementation subpackages as black-box violations even in external-package tests. Separately inventory known cases and their owning modules; remove them as those modules migrate. Do not claim that `testpackage` detects these imports or arbitrary implementation-coupled assertions.
- Keep existing focused AST transformation and process/workspace testing where it protects behavior. This policy does not require all tests to run at CLI level or all module tests to use a single workflow test harness.
- Keep the first increment independent of the broader configuration, reporting/assessment, execution ownership, and orchestration migrations. Existing global configuration in other tests is not permission to expand this scope into a repository-wide settings refactor.
- Document the policy, temporary allowance rationale, semantic-coupling inventory, and the review obligation for additions inside allowed files. Do not use the domain glossary as a repository for engineering implementation policy.

## Testing Decisions

- The test seams were confirmed during the design discussion: supported module interfaces, with engine execution/mutation and diff behavior selected for this increment. Prefer these existing seams; no new product interface or test-only export is required.
- For execution argument behavior, exercise the supported executor construction/settings and existing command-execution substitution. Assert the external subprocess invocation contract where relevant, not the private argument-building method or internal scheduling sequence. Preserve meaningful coverage of empty/nonempty build tags, zero/nonzero CPU settings, timeout behavior, and package versus integration selection.
- For snippets, apply mutations to deterministic fixture source and inspect the exposed original/mutated snippets. Cover reachable boundary behavior such as source near the beginning or end and short files. Do not preserve arbitrary private-helper parameter combinations merely because they were previously callable from same-package tests.
- For diff behavior, use supported construction with controlled command output, then assert returned changes and changed-location queries. Cover nil/empty behavior, included/excluded locations, multiple files and change ranges, insertion/deletion behavior, command failures, and malformed diff input where these represent existing contracts.
- Existing generic command substitution supports a test-local concrete command result implementing the required output method; do not export its private constraint merely to write external tests. Prefer direct explicit reference inputs over unrelated global configuration setup when using this seam.
- Existing prior art includes external engine executor tests with injected command execution and helper processes, mutation tests that apply changes to temporary source files and assert snippets, workflow tests using filesystem fixtures, and diff tests with controlled command output. Reuse those patterns without copying their implementation-detail assertions wholesale.
- Demonstrate that a representative new same-package test outside the allowance list produces a lint failure and that an external-package equivalent passes. Verify that would-be implicit filename/package exemptions do not bypass the intended configuration. Use disposable fixtures or temporary changes, with no seeded violations left in the repository.
- Verify that the remaining five exact allowances work, that the four migrated files need no allowance, and that exclusions do not suppress unrelated lint diagnostics. Do not present this file-level check as protection against additions inside allowed files.
- Demonstrate that changing only linter configuration triggers the local lint step. Verify relevant check/tooling configuration changes also trigger lint and that CI continues using the same full-check path.
- Review the migrated assertions for observable behavior and the absence of private-access bridges or implementation-subpackage imports. Record any pre-existing out-of-scope semantic violations rather than silently blessing them because package lint passes.
- Keep tests deterministic and independent of network access. Retain source isolation/restoration and existing process cleanup semantics. Handle discovered behavioral defects separately rather than changing them opportunistically.
- Run `hk check` during implementation and `hk check --all` before handoff, including the existing race-enabled test suite and 80% coverage gate. Do not lower the gate, remove race checking, or substitute historical check results for current validation.

## Out of Scope

- Converting every remaining same-package test in this increment.
- A custom linter, semantic assertion analyzer, or function-level grandfathering utility.
- Claiming that external-package syntax proves behavioral black-box quality or forbids imports into every implementation subpackage.
- Mechanically detecting new white-box tests added inside grandfathered files; this is an explicit temporary review obligation.
- Adding test-only production exports, private-access bridges, or new APIs solely to satisfy lint.
- Moving all tests to the CLI boundary or eliminating focused AST, process, and workspace tests.
- Completing the wider configuration, orchestration, reporting/assessment, or execution architecture refactor from issue #1.
- Rewriting all existing external-package tests or removing all implementation-subpackage imports immediately; inventory those cases for their owning-module migrations.
- New mutation behavior, different mutation outcomes, changed CLI/configuration contracts, changed snippet semantics, or changes to execution, cancellation, timeout, error, or cleanup behavior.
- Lowering coverage requirements or preserving helper-level coverage through artificial public API expansion.

## Further Notes

Related architectural work: https://github.com/kejne/gremlins/issues/1. The accepted ADR describes a target design, not a claim that the whole modularity migration is already complete. This spec complements that work; it does not require its full completion before the contained testing-policy increment can land.

The read-only inventory at design time found 26 Go test files: 9 same-package files and 17 external-package files. File counts describe package declarations, not a semantic audit of every assertion. Revalidate the inventory against the implementation checkout and explain any drift rather than broadening exclusions.

The agreed first migration comprises:

- `internal/engine/executor_internal_test.go`
- `internal/engine/tokenmutator_internal_test.go`
- `internal/diff/diff_test.go`
- `internal/diff/parse_test.go`

The five remaining same-package files to identify individually in the migration allowances are:

- `cmd/gremlins_test.go`
- `cmd/unleash_test.go`
- `cmd/internal/flags/flags_test.go`
- `internal/configuration/configuration_test.go`
- `internal/exclusion/rules_test.go`

A known semantic violation outside the first slice is `internal/report/report_test.go` importing `report/internal`. Tests also retain some ambient configuration setup and implementation-coupled assertions despite external package declarations. The initial inventory was representative, not exhaustive; document additional findings without turning this increment into a whole-repository rewrite.

The installed golangci-lint version was verified as 2.14.0 and lists `testpackage` as available but disabled. Older cached source indicated implicit filename/package exemptions, but matching installed-version defaults were not established during inventory; implementation must configure and verify the intended behavior explicitly.

The current architecture linter excludes test files, so its presence does not enforce this policy. No new semantic import checker was agreed for this increment.

Use the repository glossary consistently: a mutation is a source-code change at a location, a mutant is the resulting program variant, a mutator is a rule identifying/generating mutations, and a mutation outcome is the reported disposition. This testing-policy spec does not change those concepts.

The inventory did not run tests or change code. Existing uncommitted glossary and architecture documents were left intact. Implementation must run its own checks. GitHub holds this high-level spec; any later implementation tickets belong in the repo-local Beads workspace and should link to this issue.
