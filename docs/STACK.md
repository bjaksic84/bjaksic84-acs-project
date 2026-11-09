# The agentic stack

Every tool and model used, pinned, with dates. When something changes, add a row with the date
instead of editing the old one: the sessions before and after are compared separately
(`docs/EXPERIMENT.md`). "TBD" fields are filled from the owner's setup (step 8) before the first
session.

## Coding agent (writes `src/`)

| Part | Pin | Since | Notes |
|---|---|---|---|
| Model | Qwen3.8 27B, GSQ RCO quantisation, IQ3_S GGUF (12.70 GB) | TBD | LM Studio model identifier: TBD |
| Model server | LM Studio TBD, llama.cpp runtime TBD (Vulkan) | TBD | full GPU offload; Require Authentication on; Serve on Local Network off |
| Context length | 32768 tokens | TBD | 64K is an open question (`docs/OPEN_QUESTIONS.md`, Q3) |
| Sampling | LM Studio's defaults for the model | TBD | record any change here |
| Generation speed | TBD tok/s (one chat reply, as LM Studio reports it) | TBD | |
| Harness | OpenCode TBD | TBD | config: `opencode.json` (locked) |
| Commit trailer | `Model: local-qwen3.8-27b` | | how commits by the agent are counted |

## Machines and network

| Part | What | Notes |
|---|---|---|
| Model host | The owner's Windows gaming PC, AMD Radeon RX 7800 XT (16 GiB VRAM) | runs LM Studio only |
| Workstation | The owner's Mac (arm64) | runs OpenCode, Go, git and the tests |
| Link | Tailscale: `tailscale serve --bg 1234` on the PC, HTTPS inside the tailnet only | LM Studio listens on the PC's loopback only |
| Endpoint | `LM_BASE_URL` = `https://<pc-name>.<tailnet>.ts.net/v1` | kept out of the repository (`opencode.json` reads it from the environment) |
| Token | `LM_API_TOKEN`, an LM Studio API token named `flagwire-mac`, stored in Bitwarden, entered with `read -rs` | never committed or logged |

## Reviewing model (spec, tests, reviews; no `src/` except logged interventions)

| Part | Pin | Since | What it did |
|---|---|---|---|
| Model | Claude Opus 5.5 (`claude-opus-5-5`) | 2026-11-09 | the spec, the locked tests and fixtures, the plan, these docs |
| Harness | Claude Code (web), on the owner's subscription | 2026-11-09 | |

## Toolchain

| Tool | Pin | Notes |
|---|---|---|
| Go | 1.27.3 | CI pins it in `.github/workflows/ci.yml` |
| GitHub Actions | `actions/checkout` v7.0.1, `actions/setup-go` v7.0.0, by commit SHA | |
| Make | GNU Make 3.81 or newer | the Makefiles avoid newer features |

## Inside the product

| Part | Pin | Notes |
|---|---|---|
| OpenFeature Go SDK | v1.20.0 | added to the user's module by `apply` |
| OFREP provider | v0.1.7 | added by `provider add ofrep` |
| flagd | v0.18.2 | the flag server in the demo |
| Model for `propose` (tier 2) | the same local model, through any OpenAI-compatible endpoint | `FLAGWIRE_LLM_*` |

## Changes

| Date | What changed | Why |
|---|---|---|
| 2026-11-09 | First version | |
