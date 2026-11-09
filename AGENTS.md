# flagwire: instructions for the coding agent

You are building **flagwire**, a Go command-line tool that adds OpenFeature feature flags to
existing Go codebases. The code goes in `src/` (module `github.com/bjaksic84/bjaksic84-acs-project/src`).
This file, `docs/SPEC.md`, `docs/PLAN.md` and everything in `tests/` are locked: read them, never
change them.

## Every session

1. Read `docs/STATUS.md`. It names the current slice.
2. Read that slice's card in `docs/PLAN.md`, and only the spec sections the card names.
3. Copy `docs/sessions/TEMPLATE.md` to `docs/sessions/<date>-<slice>-<n>.md` (for example
   `2026-11-20-S2-1.md`) and fill in "Plan": which tests you will work on, in what order.
4. Run the slice's tests first, to see what fails: `make test T=<feature>` (for example
   `make test T=scan-env`, or `make test T=scan-env/basic` for one level).
5. Work on the **basic** test first, then **standard**, then **advanced**.
6. Small steps: change a little, `make build`, run the test again. Read the failure message: it
   shows the script line, the command and flagwire's real output.
7. Before every commit: `make check` must pass. Then commit (format below). Keep each commit
   under about 300 changed lines.
8. At the end: run `make score`. Fill in "Result" in your session file: what passes now, what you
   tried, what is blocking you. Update `docs/STATUS.md` to match. Commit both.

## Rules

- **Never change a test, a fixture, a golden file, the spec or the plan to make something pass.**
  If a test looks wrong, or the spec and a test disagree, write it under "Questions" in
  `docs/STATUS.md`, explain why, and work on something else.
- When the spec says an output line is exact, print exactly that line.
- If you are stuck on the same failure after three attempts, stop. Write in `docs/STATUS.md` what
  you tried and what you think is wrong, commit, and end the session.
- Standard library first. A new dependency needs an entry in `docs/LICENSES.md` first, and only
  these licences: MIT, Apache-2.0, BSD, ISC, MPL-2.0. Spec section 9.2 lists the expected ones.
- `gofmt` every file. Wrap errors with context. No panics on bad input: print
  `flagwire: <message>` to stderr and use the exit codes in spec section 4.
- Write unit tests for your own code in `src/` (table-driven, `go test`). They don't replace the
  acceptance tests; they help you find bugs faster.
- Never print, log or commit a token or API key. Never read `.env` files.
- Don't push. Don't use `git commit --no-verify`.

## Commands

| Command | What it does |
|---|---|
| `make build` | Builds `bin/flagwire` from `src/` |
| `make test T=<feature>[/<level>]` | Runs acceptance scripts (`tests/scripts/<feature>/`) |
| `make unit` | Runs the unit tests in `src/` |
| `make check` | gofmt, vet, unit tests, the test harness and the lock check |
| `make score` | Runs every acceptance script and prints the score table |

Each acceptance script is a text file you can read: `tests/scripts/<feature>/<level>-<name>.txtar`.
`tests/README.md` explains its commands. Expected files are in `tests/golden/`; the apps the
scripts run on are in `tests/fixtures/`.

## Commit messages

```
feat(scan): find os.Getenv toggles

What changed and why, in one or two sentences.

Slice: S2
Tests: scan-env/basic-getenv now passes
Model: local-qwen3.8-27b
```

Use `feat`, `fix`, `test` (unit tests in `src/`), `refactor` or `docs`, with the package as the
scope. One logical change per commit. Copy the `Model:` line exactly as written here: it is how
the project counts which commits you wrote.
