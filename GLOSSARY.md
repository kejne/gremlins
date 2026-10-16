# Gremlins

Gremlins evaluates Go test suites by introducing source-code changes and observing whether tests detect them.

## Language

**Mutation**:
A specific source-code change at a location.

**Mutant**:
The program variant produced by applying a mutation.
_Avoid_: Mutator

**Mutator**:
A rule that identifies and generates possible mutations.
_Avoid_: Mutant, executor

**Mutation outcome**:
A mutation's reported disposition, including unexecuted states such as skipped or not covered.
