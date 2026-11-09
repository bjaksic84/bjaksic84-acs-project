# flagwire: specification

Version 1.0, 2026-11-09. Written before any code, by Claude Opus 5.5 (`claude-opus-5-5`) with
Bojan Jakšić. This file is locked (`tests/LOCKED`): the coding agent reads it and never edits it. A
change goes through a person or the reviewing model, with a `DECISIONS.md` entry and a
`Locked-change:` commit.

The acceptance tests in `tests/scripts/` are the executable form of this file. Where a sentence
here and a test disagree, the test decides and the disagreement is a bug to report in
`docs/OPEN_QUESTIONS.md`.

## 1. What flagwire is

flagwire is a command-line tool for Go codebases. It finds the features an existing app already
has (HTTP routes, background jobs) and the switches it already uses (environment variables that
turn things on or off), then wires those features behind
[OpenFeature](https://openfeature.dev) feature flags. It does the edits itself, on a branch,
and only keeps them if the app still builds and its tests still pass. The result is a pull request
after which every gated feature can be switched off without editing code.

Flags default to **on**: applying flagwire never changes what the app does until someone turns a
flag off. Flags are read through the OpenFeature Go SDK, so any backend works. flagwire ships two:
environment variables (no infrastructure), and [flagd](https://flagd.dev) over OFREP (switch
without a restart).

## 2. Scope and tiers

Work happens in tiers. A tier starts only when the one before it passes its locked tests
(`docs/PLAN.md`).

| Tier | Feature (test directory) | Commands |
|---|---|---|
| 1 | `cli` | `version`, `help`, exit codes |
| 1 | `init` | `init` |
| 1 | `scan-env`, `scan-routes`, `scan-flags` | `scan` |
| 1 | `propose` | `propose --no-llm` |
| 1 | `apply`, `apply-verify` | `apply` |
| 1 | `adopt` | `adopt` |
| 1 | `status` | `status` |
| 2 | `scan-jobs`, `apply-jobs` | `scan`, `apply` for `go` statements |
| 2 | `propose-llm` | `propose` with a model |
| 2 | `auth` | `auth login`, `auth status`, `auth logout` |
| 2 | `pr` | `pr` |
| 2 | `provider`, `flagd`, `deploy` | `provider add ofrep`, `sync flagd`, `toggle`, `deploy snippet` |
| 2 | `routers` | chi and gin routes |
| 2 | `e2e-glas` | the whole flow on the Glas app (runs only with a Glas checkout) |
| 3 | `cleanup`, `issues` | drafts, section 11 |

**Out of scope for the course:** TypeScript and Next.js targets, Vercel Flags and Vercel env vars,
other deployment platforms, a dashboard, a hosted service. They are the plan after the course.

## 3. Terms

- **Module**: the Go module flagwire works on, found from `--dir` (default `.`), which must hold
  `go.mod`. All paths flagwire prints or stores are relative to the module root, with `/`.
- **Toggle**: a switch the code already reads, such as `os.Getenv("FEATURE_X") == "true"`.
- **Candidate**: code that could sit behind a flag: an HTTP route registration or a `go`
  statement (a background job).
- **Feature**: a named group of candidates and toggles, proposed by `propose`.
- **Flag**: an entry in `flags.json`, plus the code that checks it.
- **Flags package**: the Go package flagwire generates in the module (section 8).
- **Provider**: where flag values come from at run time: `env` or `ofrep`.

## 4. Behaviour of every command

- Invocation: `flagwire <command> [flags] [args]`. Flags use Go's `flag` syntax (`--dir x` or
  `--dir=x`) and may come before or after the arguments.
- `--dir DIR` selects the module (default: the current directory). Every command that reads a module
  accepts it.
- Output goes to stdout; errors and warnings go to stderr as `flagwire: <message>` and
  `flagwire: warning: <message>`. Output never contains colour codes when stdout is not a terminal,
  or when `NO_COLOR` is set.
- `--json` (where listed) prints one JSON document to stdout and nothing else there.
- flagwire never writes outside the module, except its credentials (section 7.4) and the temporary
  directory.
- flagwire never prints or logs a token or API key.
- Exit codes:

| Code | Meaning |
|---|---|
| 0 | Success, including "nothing to do" |
| 1 | Runtime failure (a file, git, the network, a refused precondition) |
| 2 | Usage error: unknown command or flag, wrong arguments, unknown feature key |
| 3 | Verification failed; every change was rolled back |
| 4 | Not a Go module (`no go.mod in DIR`), or flagwire is not initialised there (`flagwire is not initialised in DIR; run flagwire init`) |

## 5. Files

### 5.1 `flagwire.toml` (written by `init`)

```toml
# flagwire configuration. Edit freely; flagwire init --force rewrites it.
module = "example.com/shop"
flags_package = "internal/flags"
manifest = "flags.json"
providers = ["env"]
env_prefix = "FLAG_"
flagd_file = "flags.flagd.json"

[verify]
commands = ["go build ./...", "go vet ./...", "go test ./..."]

# [llm]
# base_url = "http://127.0.0.1:1234/v1"
# model = "qwen3.8-27b-gsq-rco"
```

- `module`: copied from `go.mod`.
- `flags_package`: `internal/flags` when the module root has an `internal/` directory, else `flags`.
  `init --flags-package PATH` overrides it.
- `providers`: `env` always; `provider add ofrep` appends `ofrep`.
- `verify.commands`: run in order from the module root. Each is split on spaces into a program and
  its arguments (no shell).
- `[llm]`: commented out by `init`. A model is configured when `[llm]` is present or
  `FLAGWIRE_LLM_BASE_URL` is set; then `propose` uses it (section 7.3). `FLAGWIRE_LLM_BASE_URL`,
  `FLAGWIRE_LLM_MODEL` and `FLAGWIRE_LLM_API_KEY` override the file; the key is never stored in
  the file.

### 5.2 `flags.json`: the manifest

The [OpenFeature CLI manifest format](https://raw.githubusercontent.com/open-feature/cli/refs/heads/main/schema/v0/flag-manifest.json):

```json
{
  "$schema": "https://raw.githubusercontent.com/open-feature/cli/refs/heads/main/schema/v0/flag-manifest.json",
  "flags": {
    "admin": {
      "flagType": "boolean",
      "defaultValue": true,
      "description": "Routes under /admin (2 routes)"
    }
  }
}
```

flagwire writes it with two-space indentation, keys sorted, and a final newline. It keeps every
flag it didn't create, as is.

### 5.3 `.flagwire/`

flagwire's working directory, ignored by git (`init` adds `.flagwire/` to `.gitignore`, creating
the file if needed, unless a line `.flagwire/` is already there). It holds `scan.json` (last scan),
`proposals.json` (last proposals) and backups during `apply`. Git cleanliness checks ignore it.

### 5.4 The flags package

Generated by `apply`, `adopt` and `provider add` from templates (section 8). Every file starts with
`// Code generated by flagwire. DO NOT EDIT.` and is rewritten from `flags.json` each time.

## 6. Commands, tier 1

### 6.1 `version`, `help`

- `flagwire version` prints `flagwire <version>` (`dev` unless set at build time).
- `flagwire help`, `flagwire --help` and `flagwire -h` print a usage text that names every
  command available in this build, one per line as `  <command>  <one-line description>`, and exit 0.
- No command prints the usage text to stdout and exits 2.
- An unknown command exits 2 with `flagwire: unknown command "<name>"`. An unknown flag exits 2.

### 6.2 `init`

`flagwire init [--dir DIR] [--flags-package PATH] [--force]`

- Needs `go.mod` (else exit 4). Checks everything below before writing anything.
- Writes `flagwire.toml` (section 5.1). If it exists: exit 1 with
  `flagwire: already initialised (flagwire.toml exists); use --force to rewrite it`, unless
  `--force`.
- Writes `flags.json` with an empty `flags` object if it doesn't exist. If it exists it must be a
  valid manifest (JSON with a `flags` object), else exit 1 with
  `flagwire: flags.json is not a valid manifest: <reason>`; it is kept as it is.
- Adds `.flagwire/` to `.gitignore` (section 5.3).
- Changes no Go code and commits nothing.
- Prints `initialised flagwire in <DIR>` and then one line per file it wrote, as `  wrote <path>`.

### 6.3 `scan`

`flagwire scan [--dir DIR] [--json]`

Reads the module and reports toggles, candidates, and flags already in use. It changes nothing in
the module except `.flagwire/scan.json` (only when `.flagwire/` exists).

**What is read.** Every package of the module, excluding `_test.go` files, `vendor/` and `testdata/`
directories, files whose first comment line matches `^// Code generated .* DO NOT EDIT\.$`, and the
flags package. Type information is used when the module type-checks. If it doesn't (for example a
dependency is missing), scan still runs from syntax alone, prints
`flagwire: warning: N packages did not type-check; some findings may be missing` to stderr, and
exits 0. Every tier 1 detection works on modules that type-check; the detections marked
*syntactic* also work when they don't.

**Common fields.** `file` is the module-relative path. `line` is 1-based. `func` names the
enclosing function: `<package name>.<Func>`, or `<package name>.<Receiver type>.<Method>` with the
receiver's `*` and type parameters dropped; code inside a function literal belongs to the
enclosing named function; package-level code is `<package name>.init`.

**Constant strings.** A name argument counts only if it is a constant string expression: a
literal, a named constant, or a constant expression of them (`prefix + "QA"`).

#### 6.3.1 Environment toggles

A *read* is one of:

- `os.Getenv(NAME)` (via `os.Getenv`), `os.LookupEnv(NAME)` (via `os.LookupEnv`);
- a call through a **variable** (a parameter, local variable, or struct field, never a declared
  function or method) whose type's underlying type is `func(string) string` or
  `func(string) (string, bool)`, with a constant `NAME` matching `^[A-Z][A-Z0-9_]*$` (via
  `lookup`). This catches loaders that take `os.LookupEnv` as a parameter.

A read is **boolean** when any of these hold:

1. it is compared with `==` or `!=` to one of the strings `true`, `TRUE`, `True`, `1`, `yes`, `on`,
   `enabled`, `enable`, `false`, `FALSE`, `False`, `0`, `no`, `off`, `disabled`, `disable`;
2. it is the argument of `strconv.ParseBool`;
3. its string result is assigned to a local variable that, in the same function, is compared as in
   (1) or passed to `strconv.ParseBool`. "The same variable" means the same declaration (the same
   `types.Object`), not the same name: two `v`s declared in two `if` statements are different;
4. the name, split on `_`, contains one of the words `ENABLE`, `ENABLED`, `DISABLE`, `DISABLED`,
   `FEATURE`, `FLAG`, `FF`.

A toggle is reported per name, with every read of it; it is `boolean` if any read is.

```json
{
  "id": "env:FEATURE_RECOMMENDATIONS",
  "kind": "env",
  "name": "FEATURE_RECOMMENDATIONS",
  "boolean": true,
  "reads": [
    {"file": "internal/config/config.go", "line": 22, "func": "config.Load", "via": "os.Getenv"}
  ]
}
```

**Command-line flags.** `flag.Bool`, `flag.BoolVar`, and the same methods on a `*flag.FlagSet`,
with a constant flag name, are toggles of kind `cli-flag`: id `flag:<name>`, `boolean` true, `via`
one of `flag.Bool`, `flag.BoolVar`, `FlagSet.Bool`, `FlagSet.BoolVar`.

#### 6.3.2 Route candidates

A route is registered through:

- `(*http.ServeMux).HandleFunc` / `.Handle` (via `ServeMux.HandleFunc`, `ServeMux.Handle`);
- `http.HandleFunc` / `http.Handle` (via `http.HandleFunc`, `http.Handle`);
- a **registrar**: any other function or method that returns nothing, whose first parameter is a
  string and whose second parameter's type is `http.Handler`, `http.HandlerFunc`, or
  `func(http.ResponseWriter, *http.Request)` (via `registrar`). Wrappers like Glas's `srv.Handle`
  are registrars; `http.StripPrefix`, which returns a handler, is not;
- a **route table** entry: a composite literal of a struct type with exactly one field holding a
  pattern-shaped constant string and at least one field whose type is one of the three handler types
  above, keyed or not (via `route-table`). The handler is the first such field.

In every case the pattern must be a constant string shaped like
`^((GET|HEAD|POST|PUT|PATCH|DELETE|OPTIONS|CONNECT|TRACE) )?/\S*$`; anything else is skipped.

```json
{
  "id": "route:GET /admin/orders",
  "kind": "route",
  "pattern": "GET /admin/orders",
  "method": "GET",
  "path": "/admin/orders",
  "handler": "s.adminOrders",
  "via": "ServeMux.HandleFunc",
  "file": "internal/web/routes.go",
  "line": 15,
  "func": "web.Server.Routes"
}
```

- `id` is `route:` plus the pattern with single spaces. If two registrations have the same pattern,
  the later ones (in file, then line order) get `#2`, `#3`, ...
- `method` is `""` for a pattern without one.
- `handler` is the handler argument as `go/types.ExprString` prints it.
- `line` is the line of the call, or for a table entry the line of the entry's opening brace.

#### 6.3.3 Flags in use

Reported under `flags`, grouped by id, each with its `uses` (`file`, `line`, `func`):

- `openfeature`: method calls `BooleanValue`, `StringValue`, `IntValue`, `FloatValue`,
  `ObjectValue` and their `...Details` forms, in a file that imports
  `github.com/open-feature/go-sdk/openfeature`, with a constant key as the second argument.
  `type` is `boolean`, `string`, `int`, `float` or `object`.
- `launchdarkly`: method calls `BoolVariation`, `IntVariation`, `Float64Variation`,
  `StringVariation`, `JSONVariation` and their `...Detail` forms, in a file that imports
  `github.com/launchdarkly/go-server-sdk/v6` or `/v7`, with a constant key as the first argument.
  `type` is `boolean`, `int`, `float`, `string` or `object`.
- `unleash`: calls `IsEnabled` (the package function or a method), in a file that imports
  `github.com/Unleash/unleash-go-sdk/v5` or `/v6`, with a constant key as the first argument.
  `type` is `boolean`.
- `flagwire`: calls to `Enabled`, `Gate` or `GateFunc` of this module's flags package (from
  `flagwire.toml`, whatever name the file imports it under), with a constant key as the second
  argument of `Enabled` and the first of `Gate` and `GateFunc` (usually `flags.Admin`). `type` is
  `boolean`.

All four are *syntactic*, so they work in a module whose dependencies are missing (flagwire's own
gates right after an `apply` whose `go get` hasn't run, a vendor SDK that isn't in `go.mod`). Here a
constant key is a string literal, or a name (`bannerKey`, `flags.Admin`) declared by a `const`
with a string literal value in the same package or in the package it is selected from, found by
reading that package's source. The `...Details` and `...Detail` forms are uses of the same id.

```json
{"id": "openfeature:new-checkout", "sdk": "openfeature", "key": "new-checkout", "type": "boolean",
 "uses": [{"file": "internal/shop/checkout.go", "line": 12, "func": "shop.Checkout"}]}
```

#### 6.3.4 Output

`--json` prints:

```json
{"schema": "flagwire.scan/v1", "module": "example.com/shop", "toggles": [], "candidates": [], "flags": []}
```

Each list is sorted by `id`; `reads` and `uses` by file, then line. Every list is present, even
when empty. Without `--json`:

```
Toggles (3)
  env  FEATURE_RECOMMENDATIONS  boolean  internal/config/config.go:22
  ...
Candidates (7)
  route  GET /admin/orders  internal/web/routes.go:15
  ...
Flags in use (0)
```

The three heading lines are exact; the rows below them are free in format but one per item.

### 6.4 `propose`

`flagwire propose [--dir DIR] [--json] [--no-llm]`

Needs `init`. Scans, then groups candidates and boolean toggles into features. Without a model
configured (section 5.1), or with `--no-llm`, it uses the heuristic below. The model path is tier 2
(section 7.3).

**Heuristic.**

1. Route candidates are grouped by the first segment of their path (`/admin/orders` → `admin`).
   Routes whose first segment is empty (`/`, `/{$}`), contains `{`, or is one of `static`,
   `assets`, `healthz`, `livez`, `readyz`, `metrics`, `version`, `favicon.ico` are left out.
2. Each named job candidate (tier 2) is its own feature, keyed by its target's last name
   (`runCleanup` → `run-cleanup`). Anonymous jobs are left out.
3. Each boolean `env` toggle is its own feature, keyed by its name with a leading `FEATURE_`,
   `ENABLE_`, `FF_` or `FLAG_` and a trailing `_ENABLED` or `_ENABLE` removed
   (`FEATURE_RECOMMENDATIONS` → `recommendations`).
4. Keys are slugs: lower case, every run of characters outside `[a-z0-9]` becomes one `-`, no `-`
   at either end, camelCase split (`adminPanel` → `admin-panel`). Features with the same key merge.
5. Descriptions (no final full stop):
   - routes with one common first segment `S`: `Routes under /S (N routes)`, or `(1 route)`;
   - routes without one: `N routes`;
   - one job: `Background job <target>`; several: `N background jobs`;
   - toggles only: `Existing toggle NAME`, or `Existing toggles A, B`;
   - mixed: the parts joined with `; `, routes first, then jobs, then toggles.

**Output.** Writes `.flagwire/proposals.json` and prints, with `--json`, the same document:

```json
{
  "schema": "flagwire.proposals/v1",
  "source": "heuristic",
  "features": [
    {"key": "admin", "description": "Routes under /admin (2 routes)",
     "candidates": ["route:GET /admin/orders", "route:POST /admin/orders/{id}/refund"], "toggles": []}
  ]
}
```

Features are sorted by key; `candidates` and `toggles` by id. `source` is `heuristic` or `llm`;
`model` is present when it is `llm`. Without `--json`, one line per feature:
`<key>  <description>`.

### 6.5 `apply`

`flagwire apply [--dir DIR] [--json] [--no-verify] [--no-commit] [--allow-dirty] KEY...`
`flagwire apply --key KEY --candidate ID [--candidate ID]... [same flags]`

Gates candidates behind flags. The first form takes features from `.flagwire/proposals.json`; the
second gates the given candidate ids (from a fresh scan) under one key, with the description built
as in 6.4 rule 5.

**Preconditions**, checked before any change:

1. `init` done (else exit 4).
2. Every `KEY` is in the proposals (else exit 2: `flagwire: unknown feature "<key>"; run flagwire propose`).
   Every `--candidate` id is in a fresh scan (else exit 2: `flagwire: unknown candidate "<id>"`).
   A feature with toggles but no candidates is an error (exit 2) that names the `adopt` command to
   use instead. For a feature with both, apply gates the candidates and prints
   `note: <key> also has toggles; run flagwire adopt --key <key> <NAME>...`.
3. A git work tree with no uncommitted changes, untracked files included, outside `.flagwire/` (else exit 1:
   `flagwire: working tree has uncommitted changes; commit or stash them, or pass --allow-dirty`).
   Outside a git repository, apply needs `--no-commit`.

**Steps.**

1. Back up every file it will change, and remember every file it will create.
2. Add each key to `flags.json`: `flagType` boolean, `defaultValue` true, the description.
3. Regenerate the flags package (section 8) from the manifest.
4. Rewrite each site (below), and add the import of the flags package to the file. The import's name
   is `flags`, unless that name is already taken in the file; then it is `featureflags`.
5. Add `flags.Setup()` as the first statement of `func main()` in every `main` package of the module
   that imports, directly or through other packages of the module, a package with a gated site,
   unless that `main` already calls it. If no `main` package does, print
   `note: no main package imports the gated code; call flags.Setup() at startup`.
6. Add the dependencies: `go get github.com/open-feature/go-sdk@v1.20.0` (and, when `ofrep` is a
   provider, `github.com/open-feature/go-sdk-contrib/providers/ofrep@v0.1.7`), then `go mod tidy`.
   A failure here rolls back and exits 1.
7. Verify (section 6.6) unless `--no-verify`, which prints `flagwire: warning: verification skipped`.
8. Commit unless `--no-commit` (below).

**Site rewrites.** Code is edited by splicing text at the positions of the syntax tree, then running
`gofmt` on the file; untouched code keeps its formatting and comments.

| Site | Before | After |
|---|---|---|
| handler is `http.Handler` (`Handle`, registrar, table field) | `mux.Handle(p, h)` | `mux.Handle(p, flags.Gate(flags.Admin, h))` |
| handler is a function (`HandleFunc`, `HandlerFunc` parameter or field) | `mux.HandleFunc(p, h)` | `mux.HandleFunc(p, flags.GateFunc(flags.Admin, h))` |
| route table entry | `{"GET /admin/users", h}` | `{"GET /admin/users", flags.Gate(flags.Admin, h)}` |
| job (tier 2) | `go w.cleanup(ctx)` | `if flags.Enabled(ctx, flags.Cleanup, true) {` / `go w.cleanup(ctx)` / `}` |

The flag is named by the generated constant (section 8.1), never by a string literal. For a job,
the context is `ctx` if an identifier `ctx` of type `context.Context` is in scope, else
`r.Context()` if a parameter `r` of type `*http.Request` is in scope, else
`context.Background()` (adding the `context` import).

**Idempotence.** A site already wrapped for the same key is left alone. If nothing changes, apply
prints `nothing to apply: <keys> already gated`, commits nothing, and exits 0.

**Commit.** If the current branch starts with `flagwire/`, commit on it; else create and switch to
`flagwire/<key1>-<key2>...` (keys in argument order; `-2`, `-3` ... appended if the branch
exists). Stage only the files apply changed or created. The message:

```
feat(flags): gate admin behind a feature flag

Flags: admin
Verified: go build ./... && go vet ./... && go test ./...
Generated-by: flagwire dev
```

With several keys the subject is `feat(flags): gate admin, reports behind feature flags`.
`Verified:` lists the commands joined by ` && `, or says `skipped`.

**Output.**

```
gated admin: 2 routes
  internal/web/routes.go:15  GET /admin/orders
  internal/web/routes.go:16  POST /admin/orders/{id}/refund
generated internal/flags (flags.go, setup.go)
added flags.Setup() to cmd/shop/main.go
verified: go build ./... && go vet ./... && go test ./...
committed on branch flagwire/admin
```

The `gated <key>: N route(s)` / `N job(s)` (or `N routes, M jobs`), `verified: ...` and
`committed on branch <name>` lines are exact. `--json` prints
`{"schema": "flagwire.apply/v1", "flags": [{"key", "sites": [{"id", "file", "line"}]}],
"generated": [paths], "bootstrap": [paths], "verified": bool, "branch": name, "commit": sha}`.

### 6.6 Verification and rollback

Each `verify.commands` entry runs in the module root with flagwire's environment. On the first
failure flagwire:

1. restores every backed-up file and deletes every file and directory it created, including
   `go.mod` and `go.sum` changes;
2. prints to stderr `flagwire: verification failed: <command>` and the last 30 lines of that
   command's combined output;
3. exits 3, creating no branch and no commit.

After a rollback, `git status --porcelain` shows nothing outside `.flagwire/`.

### 6.7 `adopt`

`flagwire adopt [--dir DIR] [--key KEY] [--all] [--no-verify] [--no-commit] [--allow-dirty] NAME...`

Turns existing boolean environment toggles into flags without changing behaviour: the original
expression becomes the flag's fallback, so with no provider value the app does exactly what it did.
Preconditions, backups, dependencies, verification, rollback and `flags.Setup()` work as in
`apply`. `--all` adopts every boolean `env` toggle. The key is `--key` (one name only) or derived
as in 6.4 rule 3.

**Shapes.**

| Shape | Before | After |
|---|---|---|
| A: comparison | `x := os.Getenv("FEATURE_X") == "true"` | `x := flags.Enabled(C, flags.X, os.Getenv("FEATURE_X") == "true")` |
| B: `strconv.ParseBool` | `v, err := strconv.ParseBool(os.Getenv("X"))` | a new statement `v = flags.Enabled(C, flags.X, v)` |

- Shape A wraps the whole comparison expression that contains the read.
- Shape B inserts the new statement after the assignment, or, when the next statement is
  `if err != nil { ... }` without `else` (using that assignment's error variable), after that `if`.
- `C` is chosen as for jobs (section 6.5).
- Any other shape is left alone and reported on stdout as
  `skipped <NAME> at <file>:<line>: unsupported shape`, one line per read. A name with any
  skipped read is not adopted at all: it gets no manifest entry and is left out of the branch name
  and the commit subject. Adopt exits 1 if it adopted nothing.
- The manifest entry: `defaultValue` false, description `Adopted from NAME`. (The real default is
  the fallback expression; `false` is what flagd serves when told nothing.)

Commit subject: `feat(flags): adopt FEATURE_X as a feature flag` (several:
`feat(flags): adopt A, B as feature flags`), on a branch `flagwire/adopt-<key1>-<key2>...` (names
and keys in argument order; with `--all`, in toggle id order), with
the same trailers as `apply`. Output: `adopted FEATURE_X as <key>: N read(s)` per name, then the
same `generated`, `verified` and `committed` lines as `apply`.

### 6.8 `status`

`flagwire status [--dir DIR] [--json] [--check]`

Needs `init`. Compares `flags.json` with the flags the code checks (section 6.3.3, `flagwire` and
`openfeature` uses).

```json
{
  "schema": "flagwire.status/v1",
  "flags": [
    {"key": "admin", "default": true, "description": "Routes under /admin (2 routes)",
     "env": "FLAG_ADMIN", "uses": [{"file": "internal/web/routes.go", "line": 15}]}
  ],
  "unused": [],
  "undeclared": []
}
```

- `flags`: every manifest flag, sorted by key; `env` is the provider variable (section 8.2).
- `unused`: manifest keys no code checks. `undeclared`: keys the code checks that the manifest
  lacks. Both sorted.
- From tier 2, each flag also has `"flagd": "on" | "off" | "missing"` when the flagd file exists.
- Without `--json`: one line per flag, `<key>  default <on|off>  <N> uses`, then
  `unused: a, b` and `undeclared: c` lines when those lists aren't empty.
- `--check` exits 1 when `unused` or `undeclared` is not empty.

## 7. Commands, tier 2

### 7.1 Jobs

`scan` reports every `go` statement outside tests as a candidate:

```json
{"id": "job:jobs.Worker.Cleanup", "kind": "job", "target": "jobs.Worker.Cleanup",
 "anonymous": false, "file": "cmd/worker/main.go", "line": 18, "func": "main.main"}
```

- `target` names the called function in the `func` format of 6.3.
- An anonymous `go func() {...}()` has `"anonymous": true`, target `<enclosing func>.func<N>` with
  `N` counting anonymous `go` statements in that function from 1, and the same in its id.
- The same target started in two places gets `#2` as routes do.

`apply` gates jobs as in 6.5. `apply` output counts them: `gated cleanup: 1 job`.

### 7.2 chi and gin routes

*Syntactic*, in files importing `github.com/go-chi/chi/v5` or `github.com/gin-gonic/gin`:

- chi: `X.Get/Post/Put/Patch/Delete/Head/Options/Connect/Trace(pattern, h)` (method from the name),
  `X.Handle(pattern, h)` and `X.HandleFunc(pattern, h)` (no method), with prefixes from enclosing
  `X.Route(prefix, func(r chi.Router) {...})` calls. via `chi`.
- gin: `X.GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS(pattern, h...)`, with prefixes from
  `g := X.Group(prefix)` assignments in the same function; a group of a group adds both prefixes.
  via `gin`. gin's `:id` and `*path` segments are kept as written.
- `handler` is the last handler argument as written; `func` and `line` as in 6.3.2.
- A call matched here is reported once, with `via` `chi` or `gin`, never also as a registrar
  (chi's `Get` has a registrar's signature when the module type-checks).

`apply` gates chi routes like `net/http` ones (`h` is an `http.HandlerFunc`, so `GateFunc`). Gin
gating is tier 3.

### 7.3 `propose` with a model

When a model is configured (section 5.1) and `--no-llm` is absent:

- POST `<base_url>/chat/completions` with `Authorization: Bearer <key>` when a key is set, and:

```json
{"model": "<model>", "temperature": 0,
 "messages": [{"role": "system", "content": "<prompt>"}, {"role": "user", "content": "<scan JSON>"}],
 "response_format": {"type": "json_schema", "json_schema": {"name": "flagwire_proposals", "strict": true,
   "schema": {"...": "the features array of 6.4: key, description, candidates, toggles"}}}}
```

- The prompt is a versioned file embedded in the binary (`prompts/propose.v1.md`). The user message
  is the scan document with only the candidates and boolean toggles.
- The reply's message content must be JSON `{"features": [...]}` as in 6.4, after removing
  surrounding whitespace and one surrounding Markdown code fence (```` ```json ```` ... ```` ``` ````),
  which local models often add. It is **valid** when
  every key matches `^[a-z][a-z0-9-]{0,40}$` and is unique, every id it names is in the scan, no id
  is in two features, and every feature names at least one id.
- When it's invalid, flagwire asks again, adding a user message that lists each problem on its own
  line (for example `unknown candidate "route:GET /nope"`). At most 3 requests in all.
- After 3 invalid replies it uses the heuristic, prints
  `flagwire: warning: the model's proposals were invalid 3 times; using the heuristic`, and exits 0
  with `source` `heuristic`.
- If the model can't be reached, or answers with an HTTP error, flagwire exits 1 with
  `flagwire: cannot reach the model at <base_url>: <reason>; run with --no-llm to use the heuristic`.
- With a valid reply, `source` is `llm` and `model` is the model name. Descriptions come from the
  model.

### 7.4 `auth`

`flagwire auth login github [--token-stdin] [--with-gh]`, `flagwire auth status`,
`flagwire auth logout github`

- **Endpoints**: API `FLAGWIRE_GITHUB_API_URL` (default `https://api.github.com`), OAuth
  `FLAGWIRE_GITHUB_OAUTH_URL` (default `https://github.com`), OAuth app client id
  `FLAGWIRE_GITHUB_CLIENT_ID` (default: the id in `docs/DECISIONS.md`).
- **Device flow** (default): POST `<oauth>/login/device/code` (form `client_id`, `scope=repo`,
  header `Accept: application/json`), print `First copy your one-time code: <user_code>` and
  `Then open <verification_uri>`, and poll POST `<oauth>/login/oauth/access_token` (form
  `client_id`, `device_code`, `grant_type=urn:ietf:params:oauth:grant-type:device_code`, same
  header) every `interval` seconds. GitHub answers HTTP 200 with an `error` field while waiting:
  `authorization_pending` means keep polling, `slow_down` means add 5 seconds,
  `expired_token` and `access_denied` end with exit 1.
- `--token-stdin` reads the token from the first line of stdin. `--with-gh` takes the output of
  `gh auth token`.
- Every login checks the token with GET `<api>/user`. A 401 exits 1 with
  `flagwire: github rejected the token`. Success prints `logged in to github as <login>`.
- **Storage**: service `flagwire`, account `github`, in the OS keychain. With
  `FLAGWIRE_KEYRING=file`, in `$FLAGWIRE_CONFIG_DIR/credentials.json` (default config dir:
  `os.UserConfigDir()/flagwire`), mode 0600, as `{"github": {"token": "...", "login": "..."}}`.
- `FLAGWIRE_GITHUB_TOKEN`, when set, is used instead of the stored token everywhere.
- `auth status` reads storage only: `github: logged in as <login>`, `github: not logged in`, or
  `github: using FLAGWIRE_GITHUB_TOKEN`.
- `auth logout github` deletes the stored token: `logged out of github`.

### 7.5 `pr`

`flagwire pr [--dir DIR] [--remote NAME] [--repo OWNER/NAME] [--base BRANCH] [--draft]`

- Preconditions (exit 1 each): a token (`flagwire: not logged in to github; run flagwire auth login github`);
  the current branch starts with `flagwire/` (`flagwire: not on a flagwire branch`); a clean tree.
- `--remote` defaults to `origin`. `--repo` defaults to the owner and name parsed from the remote's
  fetch URL (`https://github.com/OWNER/NAME(.git)` or `git@github.com:OWNER/NAME(.git)`); without
  either, exit 1. `--base` defaults to the repository's `default_branch` (GET `<api>/repos/OWNER/NAME`),
  else `main`.
- Pushes with `git push -u <remote> <branch>`, then POST `<api>/repos/OWNER/NAME/pulls` with
  `title`, `head` (the branch), `base`, `body`, `draft`.
- The branch's commits are those of `git log <base>..HEAD`, with `<remote>/<base>` when that ref
  exists and the local `<base>` otherwise. Its *flagwire commits* are those with a
  `Generated-by: flagwire <version>` trailer. The title is the subject of the newest one. The flags
  are the union of their `Flags:` trailers; the sites come from a scan (`flagwire` uses of those
  keys).
- The body is exactly this text, with a final newline. The table has one row per flag and the
  Sites list one line per flag, both sorted by key; `Default` is the manifest's `defaultValue` as
  `on` or `off`; `Sites` counts the flag's uses; the example variable in "Switching a flag" is the
  first row's.

```
## Feature flags added by flagwire

| Flag | Default | Variable | Sites |
|---|---|---|---|
| `admin` | on | `FLAG_ADMIN` | 2 |

### Sites
- `admin`: `internal/web/routes.go:15`, `internal/web/routes.go:16`

### Switching a flag
Set a flag's variable, for example `FLAG_ADMIN=false`, and restart; or run flagd and set `FLAGWIRE_PROVIDER=ofrep` to switch without a restart. With no value set, a gated flag is on and an adopted flag follows the variable it replaced.

### Verification
go build ./... && go vet ./... && go test ./...

Generated by flagwire dev.
```

  The Verification line is the `Verified:` trailer of the newest flagwire commit, and the last
  line names the version in its `Generated-by:` trailer.
- Prints `opened <html_url>`.

### 7.6 `provider`

`flagwire provider add ofrep [--no-verify] [--no-commit] [--allow-dirty]`, `flagwire provider list`

- `add ofrep` appends `ofrep` to `providers`, regenerates the flags package (adding
  `setup_ofrep.go`), adds the dependency, verifies and commits as `apply` does, with subject
  `feat(flags): add the ofrep flag provider` on the current branch, or `flagwire/provider-ofrep`.
  Prints `added provider ofrep`. If it's already there: `provider ofrep is already enabled`, exit 0.
- `list` prints one line per provider: `env`, `ofrep`.

### 7.7 `sync flagd` and `toggle`

`flagwire sync flagd [--file PATH] [--dry-run]`

Writes the flagd flag file (default `flagd_file`) from the manifest:

```json
{
  "$schema": "https://flagd.dev/schema/v0/flags.json",
  "flags": {
    "admin": {"state": "ENABLED", "variants": {"on": true, "off": false}, "defaultVariant": "on"}
  }
}
```

- The file is written like the manifest: two-space indentation, keys sorted (so each flag reads
  `defaultVariant`, `state`, `variants`), and a final newline.
- Boolean manifest flags only; others are skipped with a note.
- A new flag's `defaultVariant` is `on` if its `defaultValue` is true, else `off`. A flag already in
  the file keeps its `defaultVariant` (so toggles survive). Flags not in the manifest are removed.
- Prints one line per flag, `added admin (on)`, `kept reports (off)`, `removed old-flag`, then
  `wrote <path>`. `--dry-run` prints the same lines with `would add`, `would keep`, `would remove`
  and writes nothing.

`flagwire toggle KEY on|off [--file PATH] [--k8s-configmap NAME [--namespace NS] [--context CTX]]`

- Sets the flag's `defaultVariant` and prints `<key>: <on|off> in <path>`.
- Unknown key: exit 1, `flagwire: flag "<key>" is not in <path>; run flagwire sync flagd`.
  Missing file: exit 1, same advice. A value other than `on`/`off`: exit 2.
- **The file is rewritten in place** (truncate and write, same inode), never replaced by rename:
  Docker Compose bind-mounts single files, and a renamed file would not reach the container.
- With `--k8s-configmap NAME`, it then runs
  `kubectl [--context CTX] --namespace NS create configmap NAME --from-file=flags.flagd.json=<path> --dry-run=client -o yaml`
  and pipes the output into `kubectl [--context CTX] --namespace NS apply -f -`. `NS` defaults to
  `default`. Prints `applied configmap NAME in namespace NS`.

### 7.8 `deploy snippet`

`flagwire deploy snippet compose|k8s [--app NAME] [--image IMAGE] [--file PATH]`

Prints configuration to run flagd beside the app. `--app` defaults to `app`, `--image` to
`ghcr.io/open-feature/flagd:v0.18.2`, `--file` to `flagd_file` (printed as `./<file>` in the
Compose volume). The exact text is in `tests/golden/deploy/` and is part of this spec. Compose
mounts the file and points the app at `http://flagd:8016`. Kubernetes adds a ConfigMap
`<app>-flags` holding the file's current content (each line indented four spaces), a flagd
sidecar, and points the app at `http://localhost:8016`; it needs the file (else exit 1,
`flagwire: <file> does not exist; run flagwire sync flagd`). Pin the image by digest with
`--image` when the deployment requires it (Glas does).

## 8. The generated flags package

Placed at `flags_package`. Files and content are exact up to `gofmt` and import order. The tests
compare them with `tests/golden/`.

### 8.1 `flags.go`

For each manifest key, a constant: the key split on `-`, each part with its first letter upper
case, joined (`admin` → `Admin`, `qr-code` → `QrCode`). Constants are sorted by key, in one
parenthesised block even when there's one, each with the description as a trailing comment (no
comment when the description is empty).

```go
// Code generated by flagwire. DO NOT EDIT.

// Package flags holds this module's feature flags. flagwire regenerates it
// from flags.json: change the manifest, not this file.
package flags

import (
	"context"
	"net/http"

	"github.com/open-feature/go-sdk/openfeature"
)

// Flag keys, as listed in flags.json.
const (
	Admin   = "admin"   // Routes under /admin (2 routes)
	Reports = "reports" // Routes under /reports (2 routes)
)

// client evaluates every flag of this module.
var client = openfeature.NewDefaultClient()

// Enabled reports whether the boolean flag key is on. When no provider knows
// the flag, or evaluation fails, it returns def.
func Enabled(ctx context.Context, key string, def bool) bool {
	v, _ := client.BooleanValue(ctx, key, def, openfeature.EvaluationContext{})
	return v
}

// Gate returns h guarded by flag key: while the flag is off, requests get
// 404 Not Found. The flag defaults to on.
func Gate(key string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Enabled(r.Context(), key, true) {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// GateFunc is Gate for handler functions.
func GateFunc(key string, f func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return Gate(key, http.HandlerFunc(f)).ServeHTTP
}
```

### 8.2 `setup.go`

`envNames` maps each constant to `env_prefix` + the key in upper case with `-` as `_`
(`qr-code` → `FLAG_QR_CODE`), sorted by key.

```go
// Code generated by flagwire. DO NOT EDIT.

package flags

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// envNames maps each flag to the environment variable the env provider reads.
var envNames = map[string]string{
	Admin:   "FLAG_ADMIN",
	Reports: "FLAG_REPORTS",
}

// providers builds the providers that FLAGWIRE_PROVIDER can name.
var providers = map[string]func(context.Context) (openfeature.FeatureProvider, error){
	"env": newEnvProvider,
}

// Setup registers the provider named by FLAGWIRE_PROVIDER: "env" when unset,
// and "none" leaves every flag at its default. Setup never stops the program:
// on any problem it logs a warning and the flags keep their defaults.
func Setup() {
	name := os.Getenv("FLAGWIRE_PROVIDER")
	if name == "" {
		name = "env"
	}
	if name == "none" {
		return
	}
	newProvider, ok := providers[name]
	if !ok {
		slog.Warn("flagwire: unknown FLAGWIRE_PROVIDER; flags keep their defaults", "provider", name)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := newProvider(ctx)
	if err == nil {
		err = openfeature.SetProviderWithContextAndWait(ctx, p)
	}
	if err != nil {
		slog.Warn("flagwire: flag provider not ready; flags keep their defaults", "provider", name, "error", err)
	}
}

// newEnvProvider reads each flag's environment variable once, at startup. A
// flag whose variable is unset keeps its default.
func newEnvProvider(context.Context) (openfeature.FeatureProvider, error) {
	flags := map[string]memprovider.InMemoryFlag{}
	for key, name := range envNames {
		raw, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		on, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("%s=%q is not a boolean", name, raw)
		}
		variant := "off"
		if on {
			variant = "on"
		}
		flags[key] = memprovider.InMemoryFlag{
			Key:            key,
			State:          memprovider.Enabled,
			DefaultVariant: variant,
			Variants:       map[string]any{"on": true, "off": false},
		}
	}
	return memprovider.NewInMemoryProvider(flags), nil
}
```

### 8.3 `setup_ofrep.go` (only when `ofrep` is a provider)

```go
// Code generated by flagwire. DO NOT EDIT.

package flags

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk/openfeature"
)

func init() {
	providers["ofrep"] = newOFREPProvider
}

// newOFREPProvider evaluates flags remotely over the OpenFeature Remote
// Evaluation Protocol, at FLAGWIRE_OFREP_URL (for example a flagd sidecar at
// http://flagd:8016).
func newOFREPProvider(context.Context) (openfeature.FeatureProvider, error) {
	url := os.Getenv("FLAGWIRE_OFREP_URL")
	if url == "" {
		return nil, errors.New("FLAGWIRE_OFREP_URL is not set")
	}
	return ofrep.NewProvider(url, ofrep.WithTimeout(2*time.Second)), nil
}
```

These templates were compiled, vetted and run against go-sdk v1.20.0 and the ofrep provider v0.1.7
on 2026-11-09: defaults on, `FLAG_ADMIN=false` turns `/admin` into 404, an invalid value or an
unreachable flagd leaves the defaults, and a fake OFREP server answering `false` turns the route off.

## 9. Architecture (guidance, not tested)

### 9.1 Layout

```
src/
  go.mod                  module github.com/bjaksic84/bjaksic84-acs-project/src
  cmd/flagwire/main.go    calls cli.Main(os.Args[1:]) and exits with its code
  internal/cli/           command dispatch, flags, output, exit codes
  internal/config/        flagwire.toml
  internal/manifest/      flags.json
  internal/load/          loading packages (golang.org/x/tools/go/packages), with the syntax fallback
  internal/scan/          env.go, routes.go, jobs.go, sdks.go, model.go (the JSON types)
  internal/propose/       heuristic.go, llm.go, validate.go, prompts/propose.v1.md (embedded)
  internal/rewrite/       splice.go, gate.go, adopt.go, bootstrap.go, imports.go
  internal/gen/           the flags package templates (embedded text/template files)
  internal/verify/        running commands, backups, rollback
  internal/gitx/          git through the git command
  internal/github/        REST client and device flow
  internal/auth/          credential storage
  internal/flagd/         sync, toggle, snippets
  internal/llm/           OpenAI-compatible client
```

### 9.2 Libraries

Prefer the standard library. Allowed when they earn their place, recorded in `docs/LICENSES.md`
first: `golang.org/x/tools` (packages, `ast/astutil`), `github.com/BurntSushi/toml`,
`github.com/zalando/go-keyring`, `golang.org/x/oauth2` (only if it fits the GitHub behaviour in
7.4). Command dispatch uses the standard `flag` package. Git and `gh` are run as commands, not
libraries.

### 9.3 Editing code

Find sites with `go/ast` and `go/types`, then edit the original bytes at the node positions,
back to front in each file. Add imports by re-parsing and using `astutil.AddNamedImport`, then
`go/format`. Never print a whole file from a rewritten tree: comments move. Load packages with
`-mod=readonly`, so that reading a module never edits its `go.mod` or `go.sum`.

### 9.4 Determinism

The same module and flags give byte-identical output: sort every list, avoid map order, write
JSON with sorted keys, and keep timestamps out of files flagwire writes.

## 10. Pins

| What | Version |
|---|---|
| Go toolchain | 1.27 (the same as Glas) |
| OpenFeature Go SDK | `github.com/open-feature/go-sdk` v1.20.0 |
| OFREP provider | `github.com/open-feature/go-sdk-contrib/providers/ofrep` v0.1.7 |
| flagd image | `ghcr.io/open-feature/flagd:v0.18.2` |
| Manifest schema | OpenFeature CLI `schema/v0/flag-manifest.json` |
| flagd schema | `https://flagd.dev/schema/v0/flags.json` |

## 11. Tier 3 (drafts)

Drafted now so the plan is visible; their tests are `status: draft` until the tier starts and they
are reviewed and locked.

### 11.1 `cleanup`

`flagwire cleanup KEY --keep on|off [--no-verify] [--no-commit] [--allow-dirty]`

Removes a flag that has done its job. `--keep on`: `flags.Gate(flags.K, h)` and
`flags.GateFunc(flags.K, h)` become `h`; a job's `if flags.Enabled(...) { go ... }` becomes the
`go` statement; an adopted `flags.Enabled(C, flags.K, E)` becomes `true`. `--keep off`: the
registration statement, table entry, or job wrapper is deleted; an adopted expression becomes
`false`. The key leaves `flags.json` and the package is regenerated. If no flag remains, the package,
its imports and the `flags.Setup()` calls are removed. Verified and committed as
`refactor(flags): remove flag admin (kept on)` on `flagwire/cleanup-<key>`.

### 11.2 `issues`

`flagwire issues sync [--repo OWNER/NAME]` opens one GitHub issue per manifest flag that has none:
title ``Remove feature flag `<key>` once it is fully rolled out``, label `flagwire`, a body listing its
sites. It records issue numbers in `flagwire.toml` under `[issues]` (`admin = 101`) and prints
`opened #101 for admin`.

### 11.3 Also tier 3

Gin gating (a generated `gin.go` with a `GateGin` middleware), echo routes, adopting toggles at their
use sites rather than where config is read, and the after-course work in section 2.
