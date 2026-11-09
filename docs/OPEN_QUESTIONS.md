# Open questions

Unknowns to settle. Each says who can answer it and what it blocks. When one is answered, move it
to "Answered" with the date and where the answer is recorded.

## Open

**Q1. Docker Compose or Kubernetes for the flagd demo on Glas?** (owner) `deploy snippet` supports
both. Compose is simpler to show live; Kubernetes (k3d) shows the ConfigMap path. Blocks: S19 and
the demo script (UX tests U13 to U15).

**Q2. Where does the Glas pull request go?** (owner, with EloRSL) A demo branch in a fork, or a
real pull request to Glas that EloRSL reviews. A real one also needs Glas's own rules: a
`LICENSES.md` entry for the OpenFeature SDK, and its `make check`. Blocks: S19, U15.

**Q3. Can the coding set-up use a 64K context?** (owner, measured) The model runs at 32K now.
OpenCode's own instructions take a large share of that. Measure LM Studio at 65536 with full GPU
offload: does it fit in 16 GiB, and how fast is it? Blocks: nothing; changes `docs/STACK.md` and
`opencode.json` if yes.

**Q4. The GitHub OAuth app for the device flow.** (owner) `auth login github` needs an OAuth app
with device flow enabled; its client id goes in `DECISIONS.md` and becomes the default. Tests use
a fake GitHub, so this blocks only the real UX test (U11) and the demo.

**Q5. The digest of `ghcr.io/open-feature/flagd:v0.18.2`.** (anyone, from the registry) Glas pins
images by digest. Blocks: the Glas deploy (S19).

**Q6. The model's licence.** (owner) Record the licence of the Qwen3.8 27B weights and of the GSQ
RCO quantisation, and check that this use is allowed (`docs/LICENSES.md`).

**Q7. Does `opencode.json` match the installed OpenCode?** (owner, at set-up) The file follows
OpenCode's current (V2) configuration: `providers`, `permissions` rules with
`action`/`resource`/`effect`, last match wins. Check that OpenCode starts with it, that an edit
under `tests/` is refused, and that `{env:...}` works in `baseURL`.

**Q8. What do `opencode stats` and `opencode export` record?** (owner, after the smoke session)
Tokens, tool calls, timings? This decides which "Measurements" fields are automatic.

## Answered

None yet.
