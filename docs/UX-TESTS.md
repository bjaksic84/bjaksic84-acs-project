# UX tests

Manual tests for the owner. The acceptance suite checks that flagwire does the right thing; these
check that a person can use it: whether the messages make sense, whether a mistake is easy to
recover from, how long things take, and whether the demo works on real infrastructure (a real
GitHub, real Docker, real Glas).

Run them twice: **run 1** before the checkpoint (Mon 14 Dec, tier 1 tests only) and **run 2**
before Demo Day (all of them). Record every result, including failures: a failed UX test is a
result for the report.

## How to record

For each test, fill in its row in the results table and keep notes below it.

- **Result:** pass (everything under "Expect" happened), partial (worked, with a problem worth
  noting), fail, or skipped (with the reason).
- **Minutes:** wall-clock time for the test, including reading output.
- **Clarity (1 to 5):** would someone who hasn't read the spec understand the output and know what
  to do next? 1 = confusing, 3 = understandable with effort, 5 = obvious.
- **Screenshot:** save to `docs/screenshots/<run>-<test>-<what>.png` (for example
  `run1-U5-diff.png`) where the test says so.
- **Bug:** if flagwire does something wrong, note it in `docs/STATUS.md` under "Questions" (for
  the coding agent) or as an open question, and record the test as fail or partial.

## Results

| Test | Tier | Run 1 result | Min | Clarity | Run 2 result | Min | Clarity |
|---|---|---|---|---|---|---|---|
| U1 First contact | 1 | | | | | | |
| U2 init | 1 | | | | | | |
| U3 scan | 1 | | | | | | |
| U4 propose | 1 | | | | | | |
| U5 apply, and the flag works | 1 | | | | | | |
| U6 apply refuses and rolls back | 1 | | | | | | |
| U7 adopt | 1 | | | | | | |
| U8 status and drift | 1 | | | | | | |
| U9 Error message tour | 1 | | | | | | |
| U10 propose with the local model | 2 | | | | | | |
| U11 Sign in to GitHub | 2 | | | | | | |
| U12 Open a real pull request | 2 | | | | | | |
| U13 Switch a flag live with flagd | 2 | | | | | | |
| U14 Switch a flag on Kubernetes | 2 | | | | | | |
| U15 Glas end to end (the demo) | 2 | | | | | | |

## Preparation (once per run)

1. Update the repository and build:
   ```sh
   cd ~/path/to/bjaksic84-acs-project
   git pull
   make build
   export PATH="$PWD/bin:$PATH"
   flagwire version
   ```
   Note the version and the commit (`git rev-parse --short HEAD`) at the top of your notes.
2. Make a scratch area outside the repository and copy the fixture apps into it:
   ```sh
   export UX=~/flagwire-ux/run1
   mkdir -p "$UX"
   cp -R tests/fixtures/shop tests/fixtures/lectures tests/fixtures/worker "$UX"/
   ```
3. Make each app a git repository, as a real project would be:
   ```sh
   for d in shop lectures worker; do (cd "$UX/$d" && git init -q -b main && git add -A && git commit -qm initial); done
   ```

---

## U1 First contact

**Goal:** someone new can find out what flagwire does and how to start.

1. `flagwire` (no arguments). **Expect:** a usage text, exit code 2 (`echo $?`).
2. `flagwire help`. **Expect:** every command with a one-line description; exit code 0.
3. `flagwire scna`. **Expect:** `flagwire: unknown command "scna"`, exit code 2.
4. `cd "$UX/shop" && flagwire scan`. **Expect:** an error saying flagwire isn't initialised and
   to run `flagwire init`, exit code 4.
5. Judge: from the help text alone, would you know to run `init`, then `scan`, then `propose`?

## U2 init

1. `cd "$UX/shop" && flagwire init`. **Expect:** `initialised flagwire in .` and the files it
   wrote: `flagwire.toml`, `flags.json`, `.gitignore`.
2. `cat flagwire.toml`. **Expect:** readable, with comments; `module = "example.com/shop"`.
3. `flagwire init` again. **Expect:** exit 1, saying it's already initialised and naming `--force`.
4. `git status`. **Expect:** only the new files; no Go file changed.
5. Commit: `git add -A && git commit -qm "flagwire init"`.

## U3 scan

1. `flagwire scan`. **Expect:** three headings, `Toggles`, `Candidates`, `Flags in use`, with
   `FEATURE_RECOMMENDATIONS` and `SHOP_BETA_CHECKOUT` as boolean toggles, `SHOP_ADDR` as not
   boolean, and the shop's seven routes (`propose` later leaves out `GET /{$}` and
   `GET /static/`).
2. Time it: `time flagwire scan > /dev/null`.
3. `flagwire scan --json | python3 -m json.tool | head -40`. **Expect:** file and line for every
   item; open one in your editor and check the line is right.
4. Repeat 1 in `"$UX/lectures"` (after `flagwire init` there). **Expect:** the admin routes from
   the route table and the `LECTURES_*` toggles read through the lookup function.
5. Judge: is the plain listing enough to decide what to flag, or do you need `--json`?
   Screenshot: `U3-scan.png`.

## U4 propose

1. `cd "$UX/shop" && flagwire propose --no-llm`. **Expect:** five features: `admin`, `cart`,
   `products`, `recommendations`, `shop-beta-checkout`, each with a description.
2. Judge: are these the features a developer of this shop would name? Which would you rename?
3. `ls .flagwire/`. **Expect:** `proposals.json` (and it's ignored by git: `git status` is clean).

## U5 apply, and the flag works

1. `flagwire apply admin`. **Expect:** `gated admin: 2 routes` with the two sites, the generated
   files, `added flags.Setup() to cmd/shop/main.go`, `verified: ...`, and
   `committed on branch flagwire/admin`. Note how long verification took.
2. `git log -1` and `git show --stat`. **Expect:** a clear commit message with `Flags:`,
   `Verified:` and `Generated-by:` lines; only the files apply changed.
3. `git show internal/web/routes.go`. **Expect:** two lines changed, each wrapping the handler in
   `flags.GateFunc(flags.Admin, ...)`, nothing else touched. Screenshot: `U5-diff.png`.
4. Run the shop with the flag on (the default):
   ```sh
   SHOP_ADDR=127.0.0.1:8081 go run ./cmd/shop &
   sleep 3; curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8081/admin/orders   # 200
   kill %1
   ```
5. Run it with the flag off:
   ```sh
   FLAG_ADMIN=false SHOP_ADDR=127.0.0.1:8081 go run ./cmd/shop &
   sleep 3; curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8081/admin/orders   # 404
   curl -s http://127.0.0.1:8081/products                                                 # still works
   kill %1
   ```
6. Judge: would you merge this diff in a real project?

## U6 apply refuses and rolls back

1. In `"$UX/shop"`, make the tree dirty: `echo "// note" >> internal/web/server.go`, then
   `flagwire apply products`. **Expect:** exit 1, nothing changed, the message says to commit,
   stash or pass `--allow-dirty`. Then `git checkout internal/web/server.go`.
2. Break a test on purpose:
   ```sh
   printf 'package web\n\nimport "testing"\n\nfunc TestBroken(t *testing.T) { t.Fatal("broken on purpose") }\n' > internal/web/broken_test.go
   git add -A && git commit -qm "a broken test"
   flagwire apply products
   ```
   **Expect:** exit 3, `verification failed: go test ./...`, the test's output, and
   `git status` clean afterwards, `internal/web/routes.go` unchanged, no new branch.
3. Clean up: `git rm -q internal/web/broken_test.go && git commit -qm "remove the broken test"`.
4. Judge: after the failure, did you know what had happened and that nothing was left behind?

## U7 adopt

1. `flagwire adopt FEATURE_RECOMMENDATIONS`. **Expect:** `adopted FEATURE_RECOMMENDATIONS as
   recommendations: 1 read`, a commit, and in `internal/config/config.go` the old expression kept
   as the fallback.
2. Behaviour is unchanged without a flag value:
   ```sh
   FEATURE_RECOMMENDATIONS=true SHOP_ADDR=127.0.0.1:8081 go run ./cmd/shop &
   sleep 3; curl -s http://127.0.0.1:8081/products; kill %1      # shows "recommended: tea"
   ```
3. The flag overrides it:
   ```sh
   FEATURE_RECOMMENDATIONS=true FLAG_RECOMMENDATIONS=false SHOP_ADDR=127.0.0.1:8081 go run ./cmd/shop &
   sleep 3; curl -s http://127.0.0.1:8081/products; kill %1      # no "recommended" line
   ```
4. Judge: is it clear from the output and the diff that nothing changes until the flag is set?

## U8 status and drift

1. `flagwire status`. **Expect:** one line per flag (`admin`, `recommendations`) with its
   default and how many places check it.
2. Add a flag nobody checks: edit `flags.json` and add
   `"old-banner": {"flagType": "boolean", "defaultValue": false, "description": "unused"}`.
3. `flagwire status; echo $?` then `flagwire status --check; echo $?`. **Expect:**
   `unused: old-banner`; exit 0 without `--check`, 1 with it.
4. Undo: `git checkout flags.json`.

## U9 Error message tour

For each, note the message and whether it tells you what to do next.

1. Outside a Go module: `cd /tmp && flagwire init`. **Expect:** exit 4, `no go.mod`.
2. A manifest that isn't valid: in a fresh copy of `shop`, `echo '{' > flags.json && flagwire init`.
   **Expect:** exit 1 naming `flags.json`; no files written.
3. No network during apply: turn Wi-Fi off, then `flagwire apply cart` in `"$UX/lectures"` or a
   fresh copy (after `init`, `propose --no-llm`). **Expect:** exit 1 from `go get`, and a clean
   tree afterwards. Turn Wi-Fi back on.
4. An unknown feature: `flagwire apply nosuch`. **Expect:** exit 2.
5. Judge the tour as a whole: 1 to 5.

---

## U10 propose with the local model (tier 2)

**Needs:** LM Studio running with the model loaded; `LM_BASE_URL` and `LM_API_TOKEN` set.

1. In `"$UX/lectures"`:
   ```sh
   export FLAGWIRE_LLM_BASE_URL="$LM_BASE_URL" FLAGWIRE_LLM_API_KEY="$LM_API_TOKEN"
   export FLAGWIRE_LLM_MODEL=<model id from docs/STACK.md>
   time flagwire propose
   ```
   **Expect:** features with descriptions written by the model; `source` is `llm` in
   `.flagwire/proposals.json`. Note the time.
2. Compare with `flagwire propose --no-llm`. Judge: which grouping would you rather apply? Note
   any feature the model invented that makes no sense, and whether it needed retries (LM Studio's
   server log shows the requests).
3. Model down: `FLAGWIRE_LLM_BASE_URL=http://127.0.0.1:9/v1 flagwire propose`. **Expect:** exit 1,
   the message names the URL and suggests `--no-llm`; the token doesn't appear anywhere.

## U11 Sign in to GitHub (tier 2)

**Needs:** the OAuth app client id (Q4).

1. `flagwire auth login github`. **Expect:** a one-time code and the GitHub URL. Open it, enter
   the code, approve. **Expect:** `logged in to github as bjaksic84`.
2. `flagwire auth status`. **Expect:** `github: logged in as bjaksic84`; no token printed.
3. `flagwire auth logout github`, then `flagwire auth login github --with-gh` (if `gh` is signed
   in). **Expect:** logged in without a browser.
4. Judge: was it obvious what to do at each step? Screenshot: `U11-device-code.png`.

## U12 Open a real pull request (tier 2)

**Needs:** U11; a private sandbox repository on GitHub, for example `bjaksic84/flagwire-sandbox`.

1. Push a fresh, initialised copy of `shop` to the sandbox's `main`.
2. `flagwire propose --no-llm && flagwire apply admin && flagwire pr`. **Expect:**
   `opened https://github.com/bjaksic84/flagwire-sandbox/pull/1`.
3. Open the pull request. **Expect:** the title from the commit, the flags table, the sites,
   how to switch a flag, the verification line. Screenshot: `U12-pr.png`.
4. Judge: would a reviewer understand the pull request without running flagwire?

## U13 Switch a flag live with flagd (tier 2)

**Needs:** Docker. Uses `"$UX/shop"` after U5 and U7.

1. `flagwire provider add ofrep`. **Expect:** `added provider ofrep`, and a commit.
2. `flagwire sync flagd`. **Expect:** `flags.flagd.json` with `admin` and `recommendations`.
3. `flagwire deploy snippet compose`. Read it. Judge: could you add this to a Compose file
   without the docs?
4. Start flagd with the file mounted, and the shop pointed at it:
   ```sh
   docker run --rm -d --name flagd-ux -p 127.0.0.1:8016:8016 \
     -v "$PWD/flags.flagd.json:/etc/flagd/flags.flagd.json:ro" \
     ghcr.io/open-feature/flagd:v0.18.2 start --uri file:/etc/flagd/flags.flagd.json
   FLAGWIRE_PROVIDER=ofrep FLAGWIRE_OFREP_URL=http://127.0.0.1:8016 SHOP_ADDR=127.0.0.1:8081 go run ./cmd/shop &
   sleep 3; curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8081/admin/orders   # 200
   ```
5. Without restarting anything: `flagwire toggle admin off`, wait 5 seconds, curl again.
   **Expect:** 404. Note how long the change took to arrive. If it never arrives, note it: the
   file watch through Docker's bind mount is the suspect, and that is a finding.
6. `flagwire toggle admin on`; curl again. **Expect:** 200.
7. Stop flagd: `docker stop flagd-ux`; curl again. **Expect:** 200 (flags fall back to on when
   flagd is unreachable). `kill %1`.

## U14 Switch a flag on Kubernetes (tier 2, if Q1 chooses Kubernetes)

**Needs:** k3d and kubectl; a local cluster, for example `k3d cluster create flagwire-ux`.

1. `flagwire deploy snippet k8s --app shop > shop-flagd.yaml` and read it.
2. Apply the ConfigMap part and a flagd Deployment using it (the snippet's sidecar can run on its
   own for this test).
3. `flagwire toggle admin off --k8s-configmap shop-flags --context k3d-flagwire-ux`. **Expect:**
   `applied configmap shop-flags in namespace default`.
4. Check what flagd serves:
   `kubectl port-forward deploy/<name> 8016:8016` and
   `curl -s -X POST http://127.0.0.1:8016/ofrep/v1/evaluate/flags/admin -d '{}'`.
   **Expect:** `"value":false`, after the time Kubernetes takes to refresh a mounted ConfigMap
   (up to about a minute). Note the delay.
5. Delete the cluster.

## U15 Glas end to end: the demo (tier 2)

**Needs:** a Glas checkout at `v0.1.0` (not Glas's `main`), Docker, the answers to Q1 and Q2.
This is the Demo Day script; time it and aim for under 6 minutes of the 10.

1. Copy Glas: `git clone <glas> "$UX/glas" && cd "$UX/glas" && git checkout -b flagwire-demo v0.1.0`.
2. `flagwire init && git add -A && git commit -qm "flagwire init"`.
3. `flagwire scan`. **Expect:** the six `/admin` routes from the route table, the login routes,
   the host and participant routes. Screenshot: `U15-scan.png`.
4. `flagwire propose --no-llm` (and, for the demo, `flagwire propose` with the local model).
5. `flagwire apply admin`. **Expect:** `gated admin: 6 routes`, verification with Glas's own tests
   passing, a commit on `flagwire/admin`. Note the verification time.
6. Run Glas (`make run`, as in Glas's README), create an admin user, sign in, open
   `/admin/users`. Stop it, start it again with `FLAG_ADMIN=false`, and open `/admin/users`.
   **Expect:** 404.
7. The flagd part, as in U13 (or U14), with `--app glas`: switch `admin` off and on without
   restarting Glas.
8. `flagwire pr` to the target chosen in Q2. Screenshot: `U15-pr.png`.
9. Judge: is this the demo? What would you cut for time?
