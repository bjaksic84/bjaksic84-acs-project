# Plan

Locked (`tests/LOCKED`). The coding agent reads this; the owner or the reviewing model changes it,
as a logged intervention. Dates are the course's (Agentic Computer Science, UL FRI 2026/27).

## How the work is cut

- A **slice** is one piece of work with its own acceptance tests. A slice may take several
  sessions; a session works on one slice only and starts fresh (no carried-over chat).
- Tests are grouped by **feature** (`tests/scripts/<feature>/`) and **level** (basic, standard,
  advanced). A slice card names the features it must make pass. Inside a slice: basic first.
- A slice is **done** when its tests pass and `make check` passes. The ratchet then keeps them
  passing.
- A tier starts when the tier before it is done, or when the owner decides to move on (a logged
  decision, with the failing tests named).
- Size: each commit stays under about 300 changed lines; a slice that needs more is split across
  sessions, never squeezed into one giant commit.

## Starting a session

The owner opens a new OpenCode session in the repository and sends exactly this, with the slice
filled in, so every session starts the same way and the experiment compares like with like:

```
Work on slice S2 from docs/PLAN.md. Follow AGENTS.md.
```

Anything else the owner types during a session is an intervention and goes in
`docs/INTERVENTIONS.md`.

## Schedule

| When | What |
|---|---|
| Mon 9 Nov to Sun 15 Nov | Setup (owner): LM Studio, Tailscale, OpenCode, pins in `docs/STACK.md`. Smoke session on S0. Pitch slide uploaded before the lecture |
| Mon 16 Nov | Pitch (3 minutes, one slide) |
| Wed 18 Nov | Written proposal |
| 16 Nov to Sun 13 Dec | **Sprint 1: tier 1**, S0 to S10. Milestone self-submitted on Moodle by Sun 13 Dec: the score table, the interventions so far, one finding |
| Mon 14 Dec | Checkpoint: 3-minute lightning talk and the written sprint deliverable |
| 14 Dec to Sun 3 Jan | **Sprint 2: tier 2**, in the priority order below. Sprint review Mon 21 Dec |
| Mon 4 Jan | Exam (not about the project) |
| 5 to 10 Jan | Glas demo, report, freeze. No new features after Thu 7 Jan |
| Mon 11 Jan | Demo Day (10 minutes); report and code due |

Targets for each week of Sprint 1: S0 to S2 by Sun 22 Nov; S3 to S5 by Sun 29 Nov; S6 to S8 by
Sun 6 Dec; S9, S10 and catch-up by Sun 13 Dec.

**Tier 2 priority** (the demo needs the first four): S16 provider, S17 flagd, S18 deploy, S19
Glas; then S15 pr and S14 auth; then S13 propose-llm; then S11 jobs and S12 routers. If time runs
short, cut from the end of this list.

## Tier 1 slices

### S0 Skeleton and CLI
- **Goal:** `src/go.mod`, `cmd/flagwire`, command dispatch, `version`, `help`, exit codes,
  `--dir`, the `flagwire: ` error format.
- **Spec:** 4, 6.1, 9.1.
- **Tests:** `cli` (3).
- **Notes:** dispatch with the standard `flag` package, one `FlagSet` per command. Flags may come
  after arguments (spec 4): parse, take the first argument, parse the rest again. `help` lists
  only commands that exist in this build. Go version: `go 1.27.0` in `src/go.mod`.

### S1 init
- **Goal:** `init`: `flagwire.toml`, `flags.json`, `.gitignore`, checks before writing.
- **Spec:** 5.1 to 5.3, 6.2.
- **Tests:** `init` (3).
- **Notes:** write the TOML from a fixed template, not a library. The manifest is written with
  two-space indentation and sorted keys (`encoding/json` with `MarshalIndent` sorts map keys).

### S2 Loading and environment toggles
- **Goal:** load the module's packages with type information, fall back to syntax when it doesn't
  type-check; find `os.Getenv`, `os.LookupEnv`, lookup-function reads and `flag.Bool` toggles;
  `scan --json` and the plain listing.
- **Spec:** 6.3 (intro), 6.3.1, 6.3.4.
- **Tests:** `scan-env` (3).
- **Notes:** `golang.org/x/tools/go/packages` with `NeedSyntax|NeedTypes|NeedTypesInfo|NeedFiles|NeedName`,
  `-mod=readonly`. Count packages with errors for the warning. Keep the JSON types in one file;
  every list is present even when empty.

### S3 Route candidates
- **Goal:** `ServeMux` and `http.Handle*` registrations, registrars, route tables.
- **Spec:** 6.3.2.
- **Tests:** `scan-routes` (3).
- **Notes:** registrars and route tables need type information. `line` for a table entry is its
  opening brace. Duplicate patterns get `#2`.

### S4 Flags in use
- **Goal:** OpenFeature, LaunchDarkly, Unleash and flagwire's own flag checks, from syntax.
- **Spec:** 6.3.3.
- **Tests:** `scan-flags` (3).
- **Notes:** no type information here: match imports by path and resolve constant keys by reading
  `const` declarations.

### S5 propose (heuristic)
- **Goal:** features from routes and boolean toggles; `.flagwire/proposals.json`.
- **Spec:** 6.4.
- **Tests:** `propose` (3).
- **Notes:** the slug rules (camelCase, runs of other characters) are easy to get subtly wrong;
  unit-test them.

### S6 The flags package
- **Goal:** generate `flags.go` and `setup.go` (and later `setup_ofrep.go`) from the manifest.
- **Spec:** 8.
- **Tests:** none of its own; `apply/basic-shop-admin` compares the result with
  `tests/golden/flags/`. Write unit tests that render the golden cases.
- **Notes:** embed the templates (`embed`, `text/template`), then `go/format`.

### S7 apply
- **Goal:** gate routes behind flags: rewrite sites, add imports, add `flags.Setup()` to main,
  `go get`, commit on a branch.
- **Spec:** 6.5.
- **Tests:** `apply` (5).
- **Notes:** splice text at node positions, back to front, then reformat (spec 9.3). Never print
  a whole rewritten tree. Most `apply` scripts need the network (`go get`).

### S8 Verification and rollback
- **Goal:** run `verify.commands`; on failure restore every file, remove created ones, exit 3.
- **Spec:** 6.6.
- **Tests:** `apply-verify` (3).
- **Notes:** back up before the first write; restore `go.mod` and `go.sum` too.

### S9 adopt
- **Goal:** turn boolean env toggles into flags with the old expression as fallback.
- **Spec:** 6.7.
- **Tests:** `adopt` (3).

### S10 status
- **Goal:** compare the manifest with the flags the code checks.
- **Spec:** 6.8.
- **Tests:** `status` (2).

## Tier 2 slices

| Slice | Goal | Spec | Tests |
|---|---|---|---|
| S11 Jobs | `go` statements as candidates; gate them | 7.1 | `scan-jobs` (3), `apply-jobs` (2) |
| S12 Routers | chi and gin routes; gate chi | 7.2 | `routers` (2) |
| S13 propose with a model | OpenAI-compatible call, validation, retries, fallback | 7.3 | `propose-llm` (3) |
| S14 auth | device flow, token from stdin or `gh`, storage | 7.4 | `auth` (3) |
| S15 pr | push the branch, open the pull request with the table | 7.5 | `pr` (3) |
| S16 provider | `provider add ofrep`, `setup_ofrep.go` | 7.6, 8.3 | `provider` (1) |
| S17 flagd | `sync flagd`, `toggle`, the Kubernetes ConfigMap | 7.7, 6.8 | `flagd` (4) |
| S18 deploy | Compose and Kubernetes snippets | 7.8 | `deploy` (2) |
| S19 Glas | the whole flow on Glas v0.1.0, and the demo | all | `e2e-glas` (2), UX tests U13 to U15 |

S13's prompt (`prompts/propose.v1.md`) is written by the agent and reviewed by the owner; the
tests check the request and the validation, not the prompt's wording.

## Tier 3 (only if tier 2 is done early)

`cleanup` and `issues` (spec 11). Their scripts are drafts; before work starts they are reviewed,
changed if needed, and locked like the others.

## After the course

TypeScript and Next.js targets, Vercel Flags and environment variables, other platforms, a
dashboard (spec section 2).
