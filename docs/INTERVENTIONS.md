# Interventions

Every time someone other than the coding agent changes what the agent does or what it produced,
it is logged here: the owner, the reviewing model (Claude Opus 5.5), or anyone else. There is no
hands-off rule; the point is that every intervention is visible and counted. The coding agent
doesn't edit this file.

## What counts

- Typing anything into a session after the standard prompt (a hint, a correction, "stop", "try
  X"). Answering a permission prompt doesn't count, unless the answer redirects the work.
- Changing, reverting or fixing anything in `src/`, or its commits.
- Changing a locked file (tests, spec, plan, `AGENTS.md`, `opencode.json`, CI). These also need a
  `Locked-change:` trailer.
- Restarting a session early, changing a setting (context length, model, sampling) mid-slice, or
  repairing the environment (LM Studio crashed, Tailscale down).
- Deciding to move on with tests still failing.

## Kinds

| Kind | Meaning |
|---|---|
| `hint` | Pointed the agent at something: a spec section, a file, a function, an error it missed |
| `correct` | Told it its approach is wrong, or what to do instead |
| `unstick` | Broke a loop: stopped it, restarted the session, narrowed the task |
| `fix` | Wrote or changed code in `src/` by hand (or had the reviewing model do it) |
| `revert` | Undid the agent's commits |
| `locked` | Changed a locked file: a wrong test, a spec gap, a plan change |
| `env` | Fixed the environment: the model server, the network, the machine, the harness config |
| `scope` | Changed what the agent works on: skipped a test, moved to the next slice |

## Log

One row per intervention. `Who` is `owner` or `opus-5.5`. `Minutes` is the person's time, rounded.
`Failure` uses the taxonomy in `docs/EXPERIMENT.md`. `Ref` is a commit or a session file.

| ID | Date | Session | Slice | Who | Kind | Failure | What happened, and what was done | Minutes | Ref |
|---|---|---|---|---|---|---|---|---|---|
