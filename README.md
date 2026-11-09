# flagwire

**flagwire** adds feature flags to an existing Go codebase. It finds the features an app already
has (HTTP routes, background jobs) and the switches it already reads (environment variables such
as `FEATURE_X=true`), wires them behind [OpenFeature](https://openfeature.dev) flags, checks that
the app still builds and its tests still pass, and opens a pull request. After that, any gated
feature can be switched off, by an environment variable or live through
[flagd](https://flagd.dev), without editing code.

This is also an experiment. It is the individual project for *Agentic Computer Science* (UL FRI,
2026/27), and its subject is what a **local** model can build: flagwire's code is written by
Qwen3.8 27B, quantised to about 3 bits, on one consumer GPU, through OpenCode. Before any code
existed, a stronger model (Claude Opus 5.5) wrote the specification and a locked acceptance
suite that defines "done" and measures progress. Every human or Opus intervention is logged.

Status: **seed pack only, no flagwire code yet.** See [`docs/STATUS.md`](docs/STATUS.md).

## Goals

Each goal is checkable from this repository.

1. **G1 Tier 1 works.** All 31 tier 1 acceptance tests pass (`make score`): `init`, `scan`,
   `propose`, `apply` with verification and rollback, `adopt`, `status`.
2. **G2 Tier 2 mostly works.** At least 20 of the 25 tier 2 tests pass: jobs, chi and gin, the
   model-backed `propose`, GitHub sign-in and pull requests, the OFREP provider, flagd, deploy
   snippets, and Glas end to end.
3. **G3 It works on a real app.** On Glas v0.1.0, `flagwire apply admin` gates its six admin
   routes, Glas's own tests still pass, and a pull request is opened (the `e2e-glas` tests and UX
   test U15).
4. **G4 Live switch.** In the demo, switching `admin` off with `flagwire toggle` makes Glas's admin
   pages answer 404 within seconds, without a restart (UX test U13 or U15).
5. **G5 The local model wrote it.** At least 80% of the commits that change `src/` carry the
   coding agent's `Model:` trailer, and every other change to `src/` is in
   [`docs/INTERVENTIONS.md`](docs/INTERVENTIONS.md).
6. **G6 The result is measured.** The report gives the pass rate per tier and level over time,
   interventions per slice by kind, and tokens and time per session, all reproducible from
   `docs/metrics/`, `docs/sessions/` and the locked suite.
7. **G7 People can use it.** Both runs of the UX tests are recorded in
   [`docs/UX-TESTS.md`](docs/UX-TESTS.md), and the tier 1 tests average a clarity of at least 3.5
   of 5.

## The agentic stack

Every pinned version is in [`docs/STACK.md`](docs/STACK.md); TBD fields are filled at set-up,
before the first coding session.

| Tool | Model (pinned) | What it does |
|---|---|---|
| OpenCode (version: `docs/STACK.md`) | Qwen3.8 27B GSQ RCO IQ3_S, in LM Studio on the owner's PC (RX 7800 XT, 16 GiB), over Tailscale | writes `src/` and its unit tests, one slice per session (`AGENTS.md`) |
| Claude Code (web) | Claude Opus 5.5 (`claude-opus-5-5`) | wrote the spec, the plan, the locked tests and these docs; reviews sessions; any change it makes later is a logged intervention |
| flagwire `propose` (tier 2) | the same local model, through its OpenAI-compatible API | groups routes and toggles into named features |

How the work is directed and measured: [`docs/EXPERIMENT.md`](docs/EXPERIMENT.md). What each
model produced, with dates: [`docs/AI-USE.md`](docs/AI-USE.md).

## Install

Needs Go 1.27, git, and (for some features) Docker and the GitHub CLI.

```sh
git clone https://github.com/bjaksic84/bjaksic84-acs-project.git
cd bjaksic84-acs-project
make build                       # bin/flagwire (once src/ exists)
export PATH="$PWD/bin:$PATH"
```

## Run

In the root of a Go module, inside a git repository with no uncommitted changes:

```sh
flagwire init                    # flagwire.toml, flags.json
flagwire scan                    # toggles, routes and jobs, with file and line
flagwire propose --no-llm        # group them into features
flagwire apply admin             # gate the feature, verify, commit on flagwire/admin
FLAG_ADMIN=false go run ./cmd/app   # the admin routes now answer 404
flagwire status                  # flags in the manifest and where the code checks them
```

The full command reference is [`docs/SPEC.md`](docs/SPEC.md), sections 6 and 7.

## Demo

The Demo Day script is UX test **U15** in [`docs/UX-TESTS.md`](docs/UX-TESTS.md): flagwire on Glas
v0.1.0, from `scan` to a pull request, then switching the admin pages off and on live through
flagd. The project's own demo is the score: `make score` prints how much of the specification the
local model has built.

## Tests

```sh
make check                       # lint, unit tests, the test harness, the lock
make test T=scan-env             # one feature's acceptance tests
make score                       # all of them, as a table by feature, tier and level
```

The acceptance suite, its format and its rules: [`tests/README.md`](tests/README.md).

## Repository

| Path | What |
|---|---|
| `src/` | flagwire's code (written by the coding agent) |
| `tests/` | the locked acceptance suite: scripts, fixtures, golden files, harness |
| `AGENTS.md` | the coding agent's instructions (locked) |
| `docs/SPEC.md`, `docs/PLAN.md` | what to build and in which order (locked) |
| `docs/STATUS.md` | where the work is now |
| `docs/DECISIONS.md`, `docs/OPEN_QUESTIONS.md`, `docs/LICENSES.md` | why, what's unknown, which licences |
| `docs/STACK.md`, `docs/EXPERIMENT.md`, `docs/INTERVENTIONS.md`, `docs/AI-USE.md` | the experiment's record |
| `docs/sessions/`, `docs/metrics/` | one file per coding session; scores over time |
| `docs/UX-TESTS.md`, `docs/screenshots/` | the manual tests and their evidence |
| `docs/course/`, `docs/report/` | the pitch, the proposal and the ERK report |

## Licence

Not chosen yet; the dependencies flagwire may use are listed in [`docs/LICENSES.md`](docs/LICENSES.md).
