# Enforce modularity without imposing a hexagonal layout

Gremlins will use task-oriented modules with explicit dependencies, adopting hexagonal principles only where they improve clarity, locality, and testability. We reject a wholesale conversion to `domain/application/ports/adapters`: preserving recognizable responsibilities and minimizing the knowledge needed for a change matter more than conformity to a pattern. This decision describes the target design, not the current implementation.

## Ownership

| Module | Owns |
| --- | --- |
| CLI/configuration | Flags, configuration precedence, typed settings, CLI exit-code mapping |
| Orchestration | Coordination of the mutation-testing workflow |
| Mutation | AST discovery, mutation rules, source transformations |
| Execution | Scheduling, isolated workspaces, applying mutations, running tests, restoring files |
| Results/assessment | Outcome values, statistics, quality-gate assessment |
| Reporting | Progress presentation, console rendering, JSON rendering |

These are responsibility assignments, not a requirement to create six packages. Execution owns when and where mutations are applied and restored; the mutation module owns the Go-specific transformation mechanics.

## Constraints

- Preserve existing CLI/configuration behavior, mutation outcomes, threshold calculations, and isolation guarantees. Handle discovered behavior changes separately.
- Design for the current Go CLI, not hypothetical library consumers, daemons, remote workers, or additional languages. Go AST and token types are legitimate implementation dependencies.
- Resolve configuration once per invocation, preserving precedence and defaults. Pass typed settings owned by receiving modules; do not replace global lookups with a giant shared settings object or configuration interface.
- Keep channels, wait groups, and worker identities internal to execution rather than imposing their lifecycle obligations on orchestration.
- Return outcome snapshots containing values such as location, mutation type, status, and snippets, not executable mutants with mutation or workspace setters.
- Keep results and quality-gate assessment independent of presentation. Reporting consumes outcomes; CLI code maps assessment failures to exit codes.
- Introduce small, consumer-owned interfaces only where substitution is justified. Keep process and workspace seams internal to execution where possible; retain useful existing seams such as `fs.FS`.
- Use concrete coverage, changed-line, exclusion, and settings values. Do not introduce a central ports package or an interface for every package.
- Use the vocabulary in [GLOSSARY.md](../../GLOSSARY.md); migrate code names alongside responsibilities rather than through a separate mass rename.

## Incremental migration

1. Establish deterministic workflow tests and baseline `go-arch-lint` rules. Workflow tests cover discovery through outcomes and quality-gate assessment using fixture source and controlled test execution without network access; retain focused AST, process, and workspace tests.
2. Replace ambient configuration with typed inputs.
3. Separate outcome values, statistics, and assessment from rendering.
4. Consolidate mutation mechanics and extract execution lifecycle ownership.
5. Remove temporary dependency allowances as each migration step lands.

Pin `go-arch-lint` through `mise` and enforce it through `hk` and existing CI. Permit existing unwanted dependencies only through specific, explained temporary allowances; avoid blanket permissions for shared packages. Keep each step independently reviewable and passing repository checks; do not add speculative package scaffolding.
