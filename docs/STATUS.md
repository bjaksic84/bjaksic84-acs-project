# Status

Updated 2026-11-09.

## Current slice

**S0 Skeleton and CLI** (`docs/PLAN.md`). Tests: `cli`. Nothing in `src/` exists yet.

## Score

0 of 56 locked tests pass (31 tier 1, 25 tier 2); 3 tier 3 drafts aren't counted. `make score`
prints the table; `docs/metrics/score.csv` keeps the history.

## Done

- 2026-11-09: the seed pack, written before any code: the spec, the plan, the locked acceptance
  suite (59 scripts, fixtures, golden files), the agent instructions and permissions, the
  Makefiles, CI and these docs. Written by Claude Opus 5.5 with the owner (`docs/AI-USE.md`).

## Next

1. Owner: finish the set-up (LM Studio, Tailscale, OpenCode) and fill in the TBD fields in
   `docs/STACK.md` and the model id in `opencode.json`.
2. Owner: `make hooks`, then a smoke session on S0 with the standard prompt.

## Blocked

Nothing.

## Questions

The coding agent writes here when a test or the spec looks wrong or unclear: which test, which
spec section, and why.

None yet.
