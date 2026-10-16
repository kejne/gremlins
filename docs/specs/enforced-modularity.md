## Problem Statement

Gremlins needs a structure that makes its behavior, dependencies, and change ownership easy for maintainers and coding agents to understand. Its task-oriented package names already provide useful navigation, but important dependencies and lifecycle obligations are hidden: global configuration reaches most of the workflow, the engine mixes discovery with process execution and reporting, and reporting also calculates statistics and determines quality-gate failures.

These responsibilities make apparently local changes require knowledge of unrelated implementation details. Adding architectural layers or interfaces everywhere would introduce more navigation and coordination rather than solve the problem. The desired improvement is clearer ownership and mechanically enforced dependency rules without changing the Go CLI's existing behavior.

## Solution

Introduce `go-arch-lint` and incrementally reshape Gremlins around cohesive, task-oriented modules with explicit inputs. Apply hexagonal principles only where a seam hides meaningful complexity or supports justified substitution; do not impose a conventional hexagonal directory layout.

Keep mutation mechanics together, place the apply/test/restore lifecycle under execution ownership, return mutation outcome snapshots independent of executable mutation machinery, and separate quality-gate assessment from presentation. Resolve configuration once per invocation and pass typed settings to the modules that need them.

Use deterministic workflow-level tests to protect behavior throughout the migration. Pin the architecture linter through the existing tool manager and run it through the same checks locally and in CI. Initially permit existing unwanted dependencies through specific, explained temporary allowances, then remove those allowances as the migration proceeds.

## User Stories

1. As a maintainer, I want package names to describe the work Gremlins performs, so that I can navigate by behavior rather than architectural terminology.
2. As a coding agent, I want each behavior to have an obvious owner, so that I can identify where a change belongs without exploring unrelated code.
3. As a maintainer, I want dependency rules enforced automatically, so that architectural intent does not depend on reviewer memory.
4. As a contributor, I want existing architectural debt identified explicitly, so that I can distinguish temporary exceptions from intended dependencies.
5. As a maintainer, I want dependency permissions tightened incrementally, so that the migration does not require a big-bang rewrite.
6. As a contributor, I want matching local and CI architecture checks, so that I can reproduce failures before publishing changes.
7. As a maintainer, I want a pinned architecture-linter version, so that tool upgrades are deliberate and checks are reproducible.
8. As a CLI user, I want existing flags, configuration sources, precedence, and defaults preserved, so that existing invocations continue to behave as before.
9. As a coding agent, I want constructor inputs to reveal the settings a module needs, so that I do not have to discover hidden global lookups.
10. As a maintainer, I want settings owned by their receiving modules, so that unrelated configuration does not become a shared dependency.
11. As a test author, I want independent workflow settings, so that tests do not need global resets and locking merely to configure their subject.
12. As a contributor adding a mutator, I want discovery rules and source transformations kept together, so that I can change Go mutation mechanics locally.
13. As a maintainer, I want Go AST and token concepts to remain usable directly, so that a Go-specific tool does not acquire speculative language abstractions.
14. As a coding agent, I want execution to own the full mutation lifecycle, so that applying and restoring mutations does not require coordinating obligations across callers.
15. As a CLI user, I want mutations tested against isolated program copies, so that mutation testing does not alter my original source files.
16. As a maintainer, I want scheduling and worker bookkeeping hidden inside execution, so that orchestration does not need to manage channels, wait groups, or worker identities.
17. As a test author, I want controlled test execution at an existing substitution seam, so that workflow tests are deterministic and do not depend on network access.
18. As a maintainer, I want focused real process and workspace tests retained, so that controlled workflow execution does not conceal defects in external execution or isolation.
19. As a reporting contributor, I want ordinary mutation outcome values, so that rendering does not require executable mutation objects or AST setup.
20. As a maintainer, I want reported outcomes detached from later mutation activity, so that completed results remain stable after execution continues or cleanup runs.
21. As a CLI user, I want existing mutation statuses preserved, so that skipped, not-covered, runnable, killed, lived, timed-out, and not-viable outcomes retain their meaning.
22. As a CLI user, I want dry-run behavior preserved, so that I can inspect runnable mutations without changing established semantics.
23. As a CLI user, I want coverage, diff selection, exclusions, and enabled mutators preserved, so that the refactor does not change which mutations are selected or executed.
24. As a maintainer, I want statistics and quality-gate assessment independent of rendering, so that changing presentation cannot change whether a run passes.
25. As a CLI user, I want existing threshold calculations and comparison behavior preserved, so that CI quality gates do not change unexpectedly.
26. As a CLI user, I want existing console and machine-readable reporting preserved, so that scripts and normal usage continue working.
27. As a CLI user, I want progress filters and mutation snippets preserved, so that I retain the information used to investigate outcomes.
28. As a maintainer, I want CLI exit-code mapping owned by the CLI, so that assessment logic does not encode process-exit policy.
29. As a test author, I want workflow tests to assert observable outcomes, so that tests survive package moves and internal scheduling changes.
30. As a maintainer, I want discovery failures tested before execution begins, so that malformed or unreadable input cannot silently trigger a partial mutation-testing run.
31. As a contributor, I want existing useful seams retained, so that the migration does not add unnecessary interfaces or replace working test infrastructure.
32. As a maintainer, I want consistent distinctions between mutation, mutant, mutator, and mutation outcome, so that code and documentation communicate the same concepts.
33. As a reviewer, I want independently reviewable migration steps with passing checks, so that behavior and dependency changes remain understandable.
34. As a CLI user, I want discovered behavioral defects handled separately from this refactor, so that architectural work does not silently change execution, cancellation, timeout, or error semantics.

## Implementation Decisions

- Follow the accepted decision to enforce modularity without imposing a hexagonal layout. Responsibility assignments do not mandate a particular number of packages.
- CLI/configuration owns flags, configuration loading and precedence, conversion into typed settings, and CLI exit-code mapping. Resolve settings once per invocation while preserving current defaults and precedence.
- Receiving modules own their settings shapes. Remove ambient configuration access from workflow logic and implementation modules; do not substitute an injected configuration lookup interface or a giant shared settings object. Keep Viper confined to configuration loading and CLI flag binding.
- Orchestration coordinates the workflow without owning process control, workspace management, rendering, or scheduling protocols.
- The mutation module owns AST discovery, mutation rules, and Go-specific source transformation mechanics. Execution owns when and where transformations are applied and restored.
- Execution owns scheduling, isolated workspaces, applying mutations, running tests, restoring source, and associated lifecycle bookkeeping. Keep channels, wait groups, and worker identities out of the interface consumed by orchestration.
- Completed run results contain ordinary mutation outcome values, including location, mutation type, status, and snippets where available. Reporting must not receive executable mutation objects or depend on mutation/workspace setters. Snapshot data must not alias mutable state in a way that changes completed outcomes later.
- Results/assessment owns statistics and quality-gate evaluation independently of rendering. Preserve current formulas, denominator choices, threshold comparisons, empty-result behavior, skipped-only behavior, and dry-run behavior rather than redesigning them.
- Reporting owns progress presentation and console/JSON rendering, consuming outcome values. Supply progress through a small consumer-owned interface or callback instead of constructing concrete reporting implementations inside orchestration.
- Assessment failures must be interpretable by the CLI without embedding CLI exit-code mappings in the assessment module.
- Introduce small, consumer-owned interfaces only where substitution is justified. Prefer existing seams, retain `fs.FS` for discovery, and keep process/workspace substitution internal to execution where practical.
- Coverage profiles, changed-line sets, exclusions, and settings remain concrete values. Go AST and token types are legitimate dependencies; do not introduce language-neutral abstractions.
- Architecture rules describe intended dependencies and specific existing exceptions. CLI frameworks and configuration-loading dependencies must not leak into mutation logic, assessment, scheduling, or reporting implementations through ambient settings access. Mutation discovery and assessment must not depend on reporting or process-execution implementations; reporting must not depend on execution orchestration.
- Composition code may wire concrete implementations together. Allowances must not grant blanket access through configuration, logging, shared, or catch-all modules. Remove an exception in the same step that removes the corresponding dependency.
- Pin `go-arch-lint` through the existing tool manager and add it to the existing check runner, which CI already invokes. Check triggering must cover changes to Go code, architecture rules, and relevant tooling configuration; configuration-only changes must not silently bypass validation. Add no separate CI-only enforcement mechanism.
- Migrate code names with their responsibilities using the agreed glossary, not through a separate mass rename. Existing names are not a requirement for the target ownership model.
- Deliver in reviewable steps: establish workflow tests and baseline architecture rules; replace ambient configuration; separate results and assessment from rendering; consolidate mutation mechanics and extract execution ownership; tighten rules throughout. Keep each step passing repository checks.

## Testing Decisions

- The primary test seam is the mutation-testing workflow, from fixture source discovery through completed mutation outcomes and quality-gate assessment. This seam and the use of controlled test execution were explicitly approved during design; no additional product interface is needed merely for testing.
- Prefer the existing discovery filesystem and test-execution substitutions. If a workflow seam must be extracted from CLI code, expose the smallest coherent workflow interface rather than adding interfaces to every dependency.
- Assert externally observable behavior: discovered mutations, mutation outcomes, selected/excluded mutations, stable snapshots, assessment results, rendered output, and CLI exit codes. Do not assert internal call sequences, worker identities, channel traffic, or package layout.
- Cover covered/not-covered mutations, changed/unchanged locations, exclusions, enabled/disabled mutators, dry runs, successful tests, failed tests, build failures, timeouts, no mutations, skipped-only results, and existing threshold edge comparisons.
- Preserve and exercise discovery-failure behavior: walking, reading, or parsing failures must fail before mutation execution begins. Cover existing cancellation behavior without introducing new cancellation semantics.
- Test configuration resolution and propagation, including existing precedence/defaults and settings that affect coverage, selection, scheduling, and test execution. Verify workflow cases can use independent settings without resetting application-wide configuration.
- Test assessment independently of rendering, and verify that console/JSON presentation choices do not change quality-gate decisions or CLI exit-code mapping.
- Retain focused AST transformation tests and process/workspace tests, including source restoration, original-source isolation, and existing timeout/process-cleanup behavior. Reuse existing helper-process patterns rather than introducing network-dependent fixtures.
- Existing prior art includes filesystem-backed discovery tests using `fstest.MapFS`, fixture-based mutation tests, discovery-error/cancellation tests, injected command execution with helper test processes, workspace-copy/cache tests, and reporting/assessment cases. Adapt this coverage as ownership changes instead of duplicating implementation-detail tests.
- Validate architecture rules against the initial allowed graph and the tightened target graph. Demonstrate that a representative forbidden dependency causes a failing architecture check and that architecture-configuration changes themselves run that check.
- Run `hk check` during development and `hk check --all` before handoff, including race-enabled tests and the repository coverage gate. Keep Go tests deterministic and independent of network access.

## Out of Scope

- A wholesale conversion to a `domain/application/ports/adapters` directory template.
- Interfaces or new packages for every existing function, dependency, or workflow step.
- A central ports package, dependency-injection framework, generic configuration interface, or giant shared settings object.
- Language-independent mutation abstractions or support for languages other than Go.
- Designing a public library interface, daemon, remote-worker system, or other speculative product direction.
- New mutators, new output formats, changed CLI/configuration contracts, revised mutation outcome meanings, or revised statistics/threshold semantics.
- Opportunistic fixes that change existing execution, cancellation, timeout, error, or cleanup behavior; report such findings separately.
- A separate mass rename or speculative package scaffolding.

## Further Notes

The agreed vocabulary and architectural rationale are recorded in the repository's glossary and ADR-0001. The ADR describes the target design, not an assertion that the migration has already happened. Include these currently uncommitted documents in the implementation handoff so the issue does not rely on inaccessible local context.

The vocabulary is: a mutation is a specific source-code change at a location; a mutant is the program variant produced by applying it; a mutator is a rule that identifies and generates possible mutations; a mutation outcome is its reported disposition, including unexecuted states.

Architecture lint enforces dependency relationships, not mutation isolation, source restoration, or cancellation correctness. Cohesive ownership and behavioral tests remain essential.

The baseline repository checks passed during design, including race-enabled tests, with 87.54% coverage. This is context, not a substitute for rerunning checks during implementation.
