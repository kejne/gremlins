# Issue tracking

This repo uses two trackers for different kinds of work:

- **GitHub Issues** in `kejne/gremlins` hold incoming reports, specs, design decisions, and other high-level work. This is Kaj's fork; use it, not the upstream `go-gremlins/gremlins` repo. Pass `--repo kejne/gremlins` to `gh` commands when repo resolution is ambiguous.
- **Beads** hold implementation tickets agents will claim and complete. Use the repo-local Beads workspace initialized in this checkout, not an upstream or personal/global issue store. Run `bd context` to verify the active workspace; read `.agents/skills/beads/SKILL.md` when working with implementation tickets.

When a GitHub issue or spec is the source for implementation tickets, link each bead to it with `--external-ref gh-<number>`. Keep detailed implementation acceptance criteria and blocking relationships in Beads; keep the high-level intent and decisions in GitHub.

## GitHub Issues

Use `gh` against `kejne/gremlins`, explicitly passing `--repo kejne/gremlins` when needed; this fork's upstream remote points to `go-gremlins/gremlins`.

- **Create**: `gh issue create --title "..." --body "..."`
- **Read**: `gh issue view <number> --json number,title,body,labels,comments`
- **List**: `gh issue list --state open --json number,title,body,labels,comments`
- **Comment**: `gh issue comment <number> --body "..."`
- **Apply/remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

GitHub sub-issues and dependencies are for organizing high-level GitHub work, such as Wayfinder maps and decision tickets. Use Beads dependencies for implementation-ticket blockers.

PRs are not a request surface for triage. GitHub shares one number space across issues and PRs; resolve an ambiguous `#n` with `gh pr view n` and fall back to `gh issue view n`.

## Beads implementation tickets

Initialize Beads with `bd init` if this repo has no Beads workspace. Use the Beads skill for the full workflow; these commands are a quick reference:

- **Find unblocked tickets**: `bd ready`
- **Inspect**: `bd show <id>`
- **Create**: `bd create "<title>" --type task --priority 2 --external-ref gh-<number> --description "..." --acceptance "..."`
- **Claim**: `bd update <id> --claim`
- **Add a blocker**: `bd dep add <ticket> <blocker>` (the ticket depends on the blocker)
- **Complete**: `bd close <id> --reason "Completed"`

Create tickets in dependency order, and ensure each ticket describes an independently verifiable implementation outcome. Keep implementation status and dependencies in Beads rather than duplicating them in GitHub task lists.

## Wayfinding

Wayfinder maps and decision tickets are high-level work and live in GitHub Issues. A map is a single issue labelled `wayfinder:map`; decision tickets are its child issues and use `wayfinder:<type>` labels (`research`, `prototype`, `grilling`, or `task`). Use GitHub's sub-issue relationship where available, otherwise task lists and `Part of #<map>` references. Use GitHub native issue dependencies for decision-ticket blockers where available; otherwise record `Blocked by: #<n>` in the issue body. Implementation work that follows a resolved map is tracked as Beads tickets linked to the relevant GitHub issue.
