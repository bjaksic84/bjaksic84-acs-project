# The acceptance suite

This directory decides when flagwire is done. It was written before any flagwire code, from
`docs/SPEC.md`, and it is **locked**: the coding agent never edits it (see "Locking" below).

## Layout

| Path | What it is |
|---|---|
| `scripts/<feature>/<level>-<name>.txtar` | One acceptance test: a script plus the files it needs |
| `fixtures/` | Small Go apps the scripts copy and run flagwire on |
| `golden/` | Exact expected output: generated code, rewritten files, deploy snippets |
| `internal/script/` | The script runner (standard library only) and its self-tests |
| `acceptance_test.go` | Builds flagwire from `../src`, lints every script, runs them in parallel |
| `cmd/score/` | Turns a run into the score table, `docs/metrics/score.csv` rows and the ratchet check |
| `cmd/lock/` | Keeps the locked files honest |
| `tests.mk` | The make targets for all of the above, included by the root `Makefile` |
| `LOCKED`, `SHA256SUMS` | Which files are locked, and their hashes |

## Features, levels and tiers

Tests are grouped **by feature**, not by slice: `scripts/scan-env/` holds everything about
environment toggles, whichever session builds it. Each feature has up to three **levels**:

- **basic**: the main path, on the simplest input. The first thing to make pass.
- **standard**: the shapes real code has: several files, more syntax, plain-text output, options.
- **advanced**: edge cases, error handling, and the cases that need care (type information,
  idempotence, rollback).

Every script also has a **tier** (spec section 2). Tier 1 is the core product; tier 2 starts once
tier 1 passes; tier 3 scripts are `status: draft` and are neither run nor counted unless
`FLAGWIRE_TEST_DRAFTS=1`. The plan's slices (`docs/PLAN.md`) point at features and levels, so
the score shows both how far the work is and where the model struggles.

## Running

From the repository root:

```sh
make test                  # every script
make test T=scan-env       # one feature
make test T=apply/basic    # a regular expression over script names
make score                 # every script, then the score table, a CSV row and the ratchet check
make test-harness          # the runner's own tests, and a lint of every script
make lock-verify           # locked files match tests/SHA256SUMS
```

Environment variables:

| Variable | Effect |
|---|---|
| `FLAGWIRE_BIN` | Test this binary instead of building `../src` |
| `FLAGWIRE_TEST_OFFLINE=1` | Skip `[net]` scripts (the ones that run `go get`) |
| `FLAGWIRE_GLAS_DIR` | A checkout of Glas at tag `v0.1.0`; enables the `[glas]` scripts |
| `FLAGWIRE_TEST_DRAFTS=1` | Also run tier 3 drafts |
| `FLAGWIRE_RESULTS` | Write per-script results as JSON (`make score` sets it) |

A script that can't run here (no network, no Glas, no POSIX shell) is **skipped**, and the score
counts skips separately from failures.

## Script format

A script is a [txtar](https://pkg.go.dev/golang.org/x/tools/txtar) archive. The header comes first:

```
# feature: scan-env
# level: basic
# tier: 1
# title: os.Getenv reads are found, with boolean comparisons marked
```

(`# status: draft` marks a tier 3 draft.) Then one command per line, and then the archive's
files (`-- name --` lines), which are written into the script's work directory `$WORK` before the
first command runs.

A line may start with conditions, each of which must hold for the line to run: `[net]`, `[glas]`,
`[drafts]`, `[linux]`, `[darwin]`, `[windows]`, `[exec:NAME]` (a program is on `PATH`), and their
negations such as `[!net]`. `[!net] skip 'why'` skips the whole script.

`! cmd` means the command must fail; `? exec` means it may fail. Words split on spaces; single
quotes group (`''` is a literal quote); `$VAR` and `${VAR}` expand from the script's environment.

| Command | Does |
|---|---|
| `exec PROGRAM [ARGS...]` | Runs a program (`flagwire` is the binary under test) and keeps its stdout, stderr and exit code |
| `stdout RE`, `stderr RE` | The last exec's output matches the regular expression (multi-line mode: `^` and `$` match at lines) |
| `exitcode N` | The last exec exited with N |
| `stdin FILE` | The next exec reads this file on stdin |
| `cmp A B` | Files are byte-for-byte equal; `stdout` and `stderr` name the last exec's output |
| `cmpgo GOT WANT` | Go files are equal after `gofmt` and after merging all imports into one sorted block |
| `jsonmatch GOT WANT` | GOT contains WANT (see below) |
| `grep RE FILE` | A file matches |
| `exists PATH...` | Paths exist |
| `fixture NAME [DIR]` | Copies `fixtures/NAME` into the work directory |
| `glascopy DIR` | Copies the Glas checkout (needs `[glas]`) |
| `gitinit [DIR]` | `git init -b main`, add everything, commit "initial" |
| `cd`, `cp`, `mv`, `rm`, `mkdir`, `chmod` | As in a shell, inside `$WORK` |
| `env KEY=VALUE...`, `unset KEY...` | Changes the environment of later commands |
| `fakellm RESPONSES.json` | Starts an OpenAI-style chat server answering with scripted replies, and sets `FLAGWIRE_LLM_*` |
| `fakegithub` | Starts a fake GitHub (REST and device flow), and sets `FLAGWIRE_GITHUB_API_URL` and `_OAUTH_URL` |
| `skip MESSAGE` | Skips the script |

**`jsonmatch`** compares a subset: every key in WANT must be in GOT with a matching value, and
GOT may have more keys. Arrays must have the same length, in order, unless WANT's first element is
`"...contains"`, in which case each remaining element must match some element of GOT. A string
`"...any"` matches any value, and `"re:..."` matches a string against a regular expression.

**The fakes** log every request, one JSON object per line, to `$FAKELLM_LOG` and
`$FAKEGITHUB_LOG`, so scripts can `grep` what flagwire sent. The fake GitHub accepts any token
except an empty one or one starting with `bad`, numbers pull requests from 7 and issues from 101,
and writes each body to `$WORK/.fakegithub/pull-<n>.md` or `issue-<n>.md`.

Every script runs with a fresh `HOME`, `FLAGWIRE_CONFIG_DIR`, `FLAGWIRE_KEYRING=file`, a git
identity of its own, no git hooks, `NO_COLOR=1`, `TZ=UTC` and `LC_ALL=C`. The host's Go settings
(`GOPATH`, `GOMODCACHE`, `GOCACHE`, `GOPROXY`, ...) pass through, so `go get` uses the normal
module cache.

## How the expected results were made

- `golden/flags/` is rendered from the templates in spec section 8.
- Every rewritten file in `golden/` was checked before it was locked: the fixture with the golden
  files in place builds, passes `go vet` and its own tests, and passes a behaviour test that turns
  the flag off (the gated route answers 404, the job doesn't start). The same behaviour test fails
  on the fixture without the gate. Several scripts carry those behaviour tests as `zz_*_test.go`
  files and run them after flagwire has done its work.
- Line numbers in expected JSON were computed from the fixture files, not typed by hand.
- The flagd digest in `golden/deploy/compose-glas.yaml` is a placeholder for `--image`, not a real
  image.

## Locking

`tests/LOCKED` lists the files that define "done": this directory, the spec, the plan, the agent
instructions, the agent's permissions and CI. `make lock-verify` checks them against
`tests/SHA256SUMS`, and the pre-commit hook (`make hooks`) refuses a commit that changes one.

A locked file changes only when a person or the reviewing model decides it must (a test that
contradicts the spec, a broken fixture). That change is an intervention: it is logged in
`docs/INTERVENTIONS.md`, committed with `FLAGWIRE_ALLOW_LOCKED_EDIT=1` and a
`Locked-change: <why>` trailer, and the hashes are refreshed with `make lock-update`. CI's
`lock-audit` fails on any commit that changes a locked file without that trailer.

## The ratchet

`docs/metrics/ratchet.txt` lists every locked test that has passed. `make score` fails if one of
them fails now, so progress can't silently slip back. `make score-update` adds newly passing
tests; removing a line is an intervention.
