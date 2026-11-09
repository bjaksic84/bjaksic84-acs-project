# Session <date>-<slice>-<n>

Copy this file to `docs/sessions/<date>-<slice>-<n>.md`. The coding agent writes "Plan" and
"Result"; the owner or the reviewing model writes "Measurements" and "Verdict".

## Plan (agent, at the start)

- Slice:
- Tests I will work on, in order:
- What I expect to be hard:

## Result (agent, at the end)

- Tests that pass now (and that didn't before):
- Tests still failing, and why I think they fail:
- What I tried that didn't work:
- Questions about the spec or the tests:
- Commits:

## Measurements (owner or reviewing model)

| Item | Value |
|---|---|
| Date, start and end time | |
| OpenCode session id | |
| Transcript | `docs/sessions/exports/<id>.json` |
| Model and context (from `docs/STACK.md`) | |
| Commit before, commit after | |
| Score before, after (locked tests passing) | |
| Tokens in, out (`opencode stats`) | |
| Tool calls, failed tool calls | |
| Typical generation speed (tok/s) | |
| Why it ended | finished / stuck / time limit / crash |
| Interventions | IDs from `docs/INTERVENTIONS.md` |

## Verdict (owner or reviewing model)

- Did the work match the spec, or only the tests?
- Failure kinds seen (taxonomy in `docs/EXPERIMENT.md`):
- Code quality notes (from reading the diff):
- One thing to change before the next session:
