# Written proposal (due Wed 18 Nov)

Draft. Adjust to the format the course asks for.

## flagwire: what a local 27B model can build against a locked specification

**Student:** Bojan Jakšić. **Repository:** https://github.com/bjaksic84/bjaksic84-acs-project

### The problem and the user

Feature flags let a team switch a feature off without a deploy, but adding them to an existing
codebase is manual work: find the right places, wrap them, wire a flag provider, and make sure
nothing broke. Tools exist to remove finished flags; I found none that add flags to an existing
codebase. The user is a developer of a Go web service who wants a kill switch for its features.

### What I will build

flagwire, a command-line tool for Go modules:

- `scan` lists routes (`net/http`, route tables, chi, gin), background jobs, existing environment
  toggles and flag-SDK calls, with file and line;
- `propose` groups them into named features, by a heuristic or by a local model;
- `apply` gates a feature behind an OpenFeature flag by editing the syntax tree, generates a
  small flags package, runs the build and tests, and commits on a branch only if they pass;
  `adopt` turns existing `FEATURE_X=true` checks into flags without changing behaviour;
- `pr` opens a GitHub pull request; `sync flagd` and `toggle` switch flags live through flagd.

### Goals

The seven checkable goals are in the repository's README: tier 1 complete (31 tests), at least
20 of 25 tier 2 tests, a real run on my app Glas, a live switch in the demo, at least 80% of the
code written by the local model, every intervention logged and measured, and two recorded runs of
manual UX tests.

### The agentic stack and method

- **Coding:** OpenCode with Qwen3.8 27B (GSQ RCO IQ3_S) in LM Studio on my Windows PC
  (RX 7800 XT, 16 GB), reached from my Mac over Tailscale. Exact versions are pinned in
  `docs/STACK.md`.
- **Specification and tests:** Claude Opus 5.5 wrote the specification, the plan and 59 locked
  acceptance tests before any code, grouped by feature and by level (basic, standard, advanced),
  with fixture apps and expected outputs that were verified to build and run.
- **Process:** one slice per fresh session with the same starting prompt; the tests are locked by
  hashes, a commit hook, the harness's permissions and CI; `make score` records progress; every
  intervention by me or the reviewing model is logged with its kind and cost.

### Schedule

Set-up and a smoke session this week; tier 1 (scanning, proposing, applying with rollback,
adopting, status) in Sprint 1 to 13 Dec; tier 2 (live flags with flagd, GitHub pull requests,
the model-backed `propose`, jobs and routers, Glas end to end) from 14 Dec to 3 Jan; demo,
report and freeze 5 to 10 Jan.

### Risks

- The local model may stall on the harder features (type-aware rewriting): the levels and the
  intervention log turn that into a result, and tier 2 is ordered so the demo needs only part
  of it.
- Context (32K now) limits each session: slices are small and every session starts fresh.
- The demo depends on Docker and Glas: the acceptance suite runs without them, and the UX tests
  are rehearsed twice.
