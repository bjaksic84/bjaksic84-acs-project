# Licences

Check here before adding a dependency, then add a row. Allowed in flagwire's binary and in the
code it generates: **MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, MPL-2.0**. Not allowed:
AGPL, GPL, BUSL, SSPL, non-commercial licences. Unmodified tools that only run beside flagwire
(flagd in a container, `git`, `gh`, `kubectl`) are listed but are not linked into anything.

"Checked" means someone read the licence file at that version; "to check" means it must be read
before the dependency is added.

## flagwire's own dependencies (`src/go.mod`)

| Module | Version | Licence | Status | Why |
|---|---|---|---|---|
| Go standard library | 1.27 | BSD-3-Clause | checked | everything by default |
| `golang.org/x/tools` (`go/packages`, `go/ast/astutil`) | pin when added | BSD-3-Clause | to check | loading packages with types (spec 9.2) |
| `github.com/BurntSushi/toml` | pin when added | MIT | to check | reading `flagwire.toml` |
| `github.com/zalando/go-keyring` | pin when added | MIT | to check | the OS keychain (tier 2, `auth`) |

## Added to the user's module by `apply` (and so present in their binary)

| Module | Version | Licence | Status |
|---|---|---|---|
| `github.com/open-feature/go-sdk` | v1.20.0 | Apache-2.0 | checked 2026-11-09 (LICENSE at v1.20.0) |
| `github.com/open-feature/go-sdk-contrib/providers/ofrep` | v0.1.7 | Apache-2.0 | checked 2026-11-09 (LICENSE at providers/ofrep/v0.1.7) |
| `golang.org/x/sync`, `golang.org/x/text` (required by go-sdk) | as go-sdk pins | BSD-3-Clause | to check |
| `github.com/go-logr/logr` (required by ofrep) | as ofrep pins | Apache-2.0 | to check |

The go-sdk's other requirements (`godog`, `testify`, `goleak`, `mock`, and through `godog` the
HashiCorp MPL-2.0 modules) are used by its own tests and are not compiled into an app.

## Test fixtures only

| Module | Version | Licence | Status |
|---|---|---|---|
| `github.com/go-chi/chi/v5` | v5.5.0 | MIT | checked 2026-11-09 |
| `github.com/gin-gonic/gin` | not downloaded (the fixture doesn't type-check on purpose) | MIT | n/a |

## Tools beside flagwire

| Tool | Version | Licence | Use |
|---|---|---|---|
| flagd (`ghcr.io/open-feature/flagd`) | v0.18.2 | Apache-2.0 | the flag server in the demo |
| OpenCode | see `docs/STACK.md` | MIT (to check at the pinned version) | the coding agent's harness |
| LM Studio | see `docs/STACK.md` | proprietary (terms to check) | serves the local model; not shipped |
| Qwen3.8 27B weights | see `docs/STACK.md` | to check (must allow this use) | the coding model; not shipped |
