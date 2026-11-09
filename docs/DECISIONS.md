# Decisions

Why things are the way they are. Newest last. Anyone may add an entry (the coding agent too, for
choices inside `src/`); an entry is never edited afterwards, only superseded by a later one.

## D-001 Go is both the implementation language and the only target (2026-11-09)

flagwire is written in Go and edits Go code. Go's standard library parses and type-checks Go
(`go/ast`, `go/types`), so finding routes and rewriting them needs no extra tooling; the result
is one static binary; the demo app, Glas, is Go. TypeScript and Next.js targets, Vercel Flags and
platform environment variables are kept in the plan for after the course (Vercel Flags has no Go
SDK, so for a Go app it would only mean environment variables).

## D-002 OpenFeature, with flags defaulting to on (2026-11-09)

Generated code calls the OpenFeature Go SDK (v1.20.0), so any backend works and the flags file is
OpenFeature's `flags.json` manifest. Every gate falls back to "on", so applying flagwire never
changes behaviour until someone switches a flag off.

## D-003 Two providers: environment variables and flagd over OFREP (2026-11-09)

The env provider is generated (an in-memory provider filled from `FLAG_<KEY>` variables at
startup), because the contrib `from-env` provider expects JSON values in its variables. The
runtime provider is OFREP (`providers/ofrep` v0.1.7) pointed at flagd v0.18.2, so a flag can be
switched without a restart and without a vendor account.

## D-004 OpenCode is the harness; no harness comparison (2026-11-09)

Antigravity can't use a custom or local model in its IDE or app, so it is dropped. OpenCode talks
to any OpenAI-compatible server, loads `AGENTS.md`, and has per-path permissions, which is what
locks the tests from the agent's side. Only one harness is used, so the results are about the
model, not a harness comparison.

## D-005 The model runs in LM Studio on the Windows PC, reached over Tailscale (2026-11-09)

Kept separate from the owner's Homebase set-up on purpose. LM Studio listens on the PC only;
`tailscale serve` exposes it over HTTPS to the tailnet; an LM Studio API token is required. The
endpoint and token come from environment variables (`LM_BASE_URL`, `LM_API_TOKEN`), so the
tailnet's name and the token stay out of this public repository.

## D-006 Tests are grouped by feature and level, and tiered (2026-11-09)

Each feature has basic, standard and advanced scripts; each script has a tier. Feature grouping
keeps a test's meaning stable however the work is sliced; levels show where the model's ability
runs out; tiers order the work. Slices in `docs/PLAN.md` point at features.

## D-007 A test harness of our own instead of `testscript` (2026-11-09)

The runner in `tests/internal/script/` uses only the standard library and borrows testscript's
txtar format. It adds what flagwire needs: Go-aware comparison (`cmpgo`), JSON subset matching,
fake OpenAI-compatible and GitHub servers, Glas and network conditions, and a lint that catches a
broken script before any flagwire exists. No third-party code sits between the agent and its
definition of done.

## D-008 The definition of done is locked, and changes are interventions (2026-11-09)

`tests/`, the spec, the plan, `AGENTS.md`, `opencode.json` and CI are hashed (`tests/SHA256SUMS`),
guarded by a pre-commit hook and by OpenCode's permissions, and audited in CI (`Locked-change:`
trailers). There is no hands-off rule, but every intervention is logged
(`docs/INTERVENTIONS.md`), by the owner or by the reviewing model.

## D-009 Golden files are checked by building and running them (2026-11-09)

Every expected rewrite was built, vetted and tested inside its fixture, with a behaviour test that
turns the flag off; the same behaviour test was run against the ungated fixture and failed. So a
golden file can't encode a rewrite that doesn't work.

## D-010 The demo app is Glas v0.1.0 (2026-11-09)

Glas is a real Go app (route tables, a registrar, an injected `LookupEnv`), owned by the owner and
EloRSL. Its end-to-end scripts run only with a checkout (`FLAGWIRE_GLAS_DIR`), so this repository
holds no Glas code. Where the resulting pull request goes is open (Q2).

## D-011 Comparisons, if any, use the owner's Claude subscription (2026-11-09)

The course's class model (dingo) is not used. An optional reference run redoes a slice with a
frontier model through Claude Code on the owner's subscription, on a throwaway branch.

## D-012 Vendor SDKs are detected from syntax, for v5 and v6 of Unleash and v6 and v7 of LaunchDarkly (2026-11-09)

Apps that already use a flag vendor often don't type-check in a fresh checkout (private modules,
missing `go.sum`). Detection by import path and method name works anyway. Unleash's docs show both
`unleash-go-sdk/v5` and `/v6` in use in 2026.
