# The experiment

flagwire is two results: the tool, and a measured account of how a local 27B model, quantised to
about 3 bits and running on one consumer GPU, builds it. This file says what is measured and how,
so the numbers in the report can be traced to files in this repository. The owner and the
reviewing model keep it; the coding agent doesn't edit it.

## Questions

1. **How far does it get?** The share of locked acceptance tests that pass, per tier and per
   level, over time and over sessions.
2. **Where does it fail?** Which features and which levels resist, and what kind of failure it is
   (the taxonomy below). The levels are designed so that "basic passes, advanced doesn't" is a
   finding, not noise.
3. **What does supervision cost?** Interventions per slice, by kind, and the owner's minutes.
4. **What does it cost to run?** Tokens, wall-clock time, and generation speed per session.
5. **Does the product's own model step work?** `propose` with the same local model (tier 2): how
   often its answer is valid on the first try, and how it compares with the heuristic on Glas.

## Set-up that stays fixed

- The model, its quantisation, the context length, the runtime and the harness version are pinned
  in `docs/STACK.md`. A change to any of them starts a new row there, with the date, and the
  sessions after it are compared separately.
- Every session starts fresh with the same prompt (`docs/PLAN.md`, "Starting a session").
- The definition of done (spec, plan, tests, agent instructions, permissions, CI) is locked
  before the first session. When it has to change, that is an intervention with a reason.
- The model writes `src/`. The owner and the reviewing model write nothing in `src/` except as a
  logged intervention.

## What is recorded, and where

| Data | Where | Made by |
|---|---|---|
| Score per run (passed, failed, skipped, per tier) | `docs/metrics/score.csv` | `make score` |
| Tests that have ever passed | `docs/metrics/ratchet.txt` | `make score-update` |
| Each session: plan, result, measurements, verdict | `docs/sessions/<date>-<slice>-<n>.md` | the agent (plan, result); the owner or the reviewing model (the rest) |
| The session transcript | `docs/sessions/exports/<session id>.json` | `opencode export <id> --sanitize` |
| Tokens and tool calls | the session file's "Measurements" | `opencode stats` |
| Interventions | `docs/INTERVENTIONS.md` | the owner or the reviewing model |
| Who wrote each commit | commit trailers: `Model: local-qwen3.8-27b` (the agent), `Co-Authored-By: Claude ...` (the reviewing model), none (the owner) | everyone |
| Manual UX results | `docs/UX-TESTS.md` and `docs/screenshots/` | the owner |

The score of any past commit can be recomputed: check it out and run the locked suite. So
`score.csv` is a convenience, not the only evidence.

## Failure taxonomy

Each failing test at the end of a session, and each intervention, gets one main kind:

| Kind | Meaning |
|---|---|
| `spec-miss` | Ignored or misread the spec: wrong output text, wrong JSON field, wrong exit code |
| `go-api` | Misused a Go API or library (`go/types`, `go/packages`, the OpenFeature SDK) |
| `logic` | Ran and printed the right shape, but the result is wrong (missed a site, wrong line) |
| `build` | Left the code not compiling, or `make check` failing |
| `regression` | Broke a test that passed before (the ratchet catches these) |
| `loop` | Repeated the same failed attempt without new information |
| `context` | Lost track after the context filled up: forgot the slice, re-read files, contradicted itself |
| `tooling` | Misused the harness: wrong tool calls, malformed edits, commands that hang |
| `scope` | Did work outside the slice, or changed things it was told not to |
| `gaming` | Tried to make a test pass without doing the work: special-casing test input, editing locked files, skipping checks |
| `env` | Not the model's fault: the machine, the network, LM Studio, a flaky test |

## Protocol per session

1. Before: note the score (`make score`, last row of `score.csv`) and the commit.
2. Start OpenCode fresh, send the standard prompt, and leave it to work. Answering a permission
   prompt is not an intervention; typing anything that changes what it does is.
3. Stop when the agent ends, when it has looped for about 15 minutes without progress, or after
   2 hours. Record which.
4. After: `make score`, `opencode stats`, `opencode export <id> --sanitize`, then fill in
   "Measurements" and "Verdict" in the session file and add any interventions.
5. Review the diff before the next session (the reviewing model may do this). A problem found in
   review and fixed by someone other than the agent is an intervention.

## Comparison runs (optional)

If time allows, one or two slices are redone on a throwaway branch by a frontier model through
Claude Code on the owner's subscription, from the same starting commit and the same prompt. The
result is a reference point for the report, not part of the product. Nothing from those branches
is merged.

## Threats to validity

- One model, one machine, one project: the results describe this set-up, not local models in
  general.
- The tests and spec were written by a much stronger model, which shapes what "done" means and
  how much help the agent gets from them.
- The owner's interventions are judgement calls; the log makes them visible but not objective.
- LM Studio, OpenCode and the model may update during the course; the pins and the session
  dates show when.
- Quantisation and the shorter context differ from the conditions of the model's published
  benchmarks (BF16 weights, 256K context), so those numbers don't transfer.

## In the report

| Report section | Comes from |
|---|---|
| Agentic stack used | `docs/STACK.md` |
| Method | this file, `docs/PLAN.md`, `AGENTS.md`, the session protocol |
| Results | `docs/metrics/`, the session files, `docs/UX-TESTS.md`, the demo |
| Limitations | the failure taxonomy counts, the threats above |
| AI-use declaration | `docs/STACK.md`, `docs/AI-USE.md`, commit trailers, `docs/INTERVENTIONS.md` |
