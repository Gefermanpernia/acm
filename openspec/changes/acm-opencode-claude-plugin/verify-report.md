```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:253438698519d414f9f2626e8f3a2a3f29b104fd2e0623c42a6a3a3d2cec8576
verdict: fail
blockers: 3
critical_findings: 3
requirements: 8/11
scenarios: 17/21
test_command: go test -count=1 ./... && go test -count=1 -race ./... && node --test integrations/opencode/test/*.test.js
test_exit_code: 0
test_output_hash: sha256:332a4c96297e96ace43ff8d2d10a0c1b7f720b1a93431532bb3eddf73bb3accf
build_command: go build ./... && gofmt -l . && go vet ./... && sh -n install.sh
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verification Report

- **Change**: `acm-opencode-claude-plugin`
- **Round**: 4 (independent final verification)
- **Tip**: `06b9ab6` on `feat/opencode-plugin-r6-compat-policy`, 15 commits from `main`
- **Mode**: full spec-driven verification (proposal + 3 specs + design + tasks present)
- **Artifact store**: hybrid (OpenSpec + Engram, project `acm`)
- **Verdict**: **FAIL** — 3 CRITICAL

## Executive Summary

Every mandated command exits `0`. All 24 Node tests and the entire Go suite pass, with and
without `-race`. All 15 commits build and pass their own tests. Round-3 C1 is **closed**: the
production entry point now loads on the installed runtime.

The change nevertheless fails, and it fails **in the same shape as rounds 1, 2, and 3**: a
documented state that no test exercises, while the suite stays fully green. This round the
defect was introduced *by the round-3 fix itself*. Commit `06b9ab6` deleted
`integrations/opencode/compatibility.json` but left `install.sh:66` fetching it. Under
`set -eu` with `curl -fsSL`, the installer aborts on a 404 and never bundles the plugin — so
the capability that round 3 restored at the *load* boundary is now unreachable at the
*distribution* boundary. `sh -n install.sh` cannot see it, and no test touches `install.sh`.

A second CRITICAL is a spec scenario whose implementation and test assert the **opposite** of
the specification, and a third is a specification that the accepted ADR silently invalidated
without amending the spec file.

## Completeness

| Dimension | Status | Evidence |
|---|---|---|
| Tasks checked | 30/30 | `tasks.md` phases 1–11, all `[x]` |
| Proposal | present | `proposal.md` |
| Specs | 3 present | auth, failover, lifecycle |
| Design | present | `design.md` |
| Requirements | 11 total | 4 auth + 4 failover + 3 lifecycle |
| Scenarios | 21 total | 7 auth + 8 failover + 6 lifecycle |

## Command Evidence

| Command | Exit | Result |
|---|---|---|
| `gofmt -l .` | 0 | no output |
| `go vet ./...` | 0 | clean |
| `go test -count=1 ./...` | 0 | `ok github.com/Gefermanpernia/acm 3.261s` |
| `go test -count=1 -race ./...` | 0 | `ok github.com/Gefermanpernia/acm 5.222s` |
| `ACM_OPENCODE_CONFIG_HOME=$(mktemp -d) go test -count=1 -run TestOpenCodeMigration .` | 0 | `ok ... 0.050s`; temp home left empty |
| `node --test integrations/opencode/test/*.test.js` | 0 | tests 24, pass 24, fail 0 |
| `sh -n install.sh` | 0 | syntax only — see C1 |
| Entry-point load (bare import) | 0 | `import_ok typeof default = function` |
| Entry-point load (invoke factory) | 0 | `hooks = auth,chat.headers`, `auth.provider = anthropic` |
| Per-commit build × 15 | 0 | all 15 `build=0` |
| Per-commit `go test` × 15 | 0 | all 15 `test=0` |
| Per-commit `node --test` × 12 | 0 | commits 4–15; commits 1–3 predate the JS suite |

Node v26.7.0. `bun` never invoked. `gentle-ai review` and `gentle-ai sdd-attempt` never invoked.

### Round-3 C1 closure (verified independently)

```
$ env -i PATH=/usr/bin:/bin HOME=/tmp node -e 'import(".../integrations/opencode/index.js")...'
import_ok typeof default = function
$ ... await m.default()
ACM diagnostics: record_failed
invoke_ok hooks = auth,chat.headers
auth.provider = anthropic
```

`compatibility.json` and `scripts/check-compat.js` are absent from the tree; `resolveVersions`
has no residual reference in any source file; `package.json` declares
`"@opencode-ai/plugin": "^1.18.18"`. ADR `docs/03-architecture/decisions/0001-...md` records the
accepted risk. **C1 closed.**

## PRIMARY OBLIGATION — Coverage-State Matrix

Legend: **Real-binary test** = a test that spawns the compiled `acm` and exercises this state.
**Live** = verified by my own probe against a freshly built binary in `/tmp` (deleted after).

### Machine dispatcher branches (`machine.go`)

| # | State | Specified in | Real-binary test | Live probe | Verdict |
|---|---|---|---|---|---|
| 1 | `credential.select` success | auth R1 S1 | yes — `machine-contract:75-79` | ok | COVERED |
| 2 | `credential.select` cooling (`no_available_profile`, exit 75, `reset_at`) | failover R8 S1 | yes — `machine-contract:144-150`, `quota-integration:53-57` | ok | COVERED |
| 3 | `credential.select` all-quarantined (exit 69) | failover R8 S2 | yes — `machine-contract:152-160` | ok | COVERED |
| 4 | `credential.select` mixed (reset from cooling only) | failover R8 S3 | yes — `machine-contract:175-179` | ok | COVERED |
| 5 | `credential.select` no-unattempted (exit 69, non-retryable) | design (implicit) | shape only — `machine-contract:80-81`; never mapped | ok | **UNEXERCISED (mapping)** — W3 |
| 6 | `credential.select` `invalid_profile_path` | design §Interfaces | `machine_test.go:228` (in-process) | — | COVERED (unit) |
| 7 | `oauth.refresh.begin` success | auth R2 S1 | yes — `machine-contract:83-89` | ok | COVERED |
| 8 | `oauth.refresh.begin` `stale_generation` | failover R7 S2 | yes — `machine-contract:181-183` | ok | COVERED |
| 9 | `oauth.refresh.begin` `lease_busy` | design §Data Flow | `machine_test.go:81` (in-process) | — | COVERED (unit) |
| 10 | **`oauth.refresh.begin` `credential_quarantined`** (`machine.go:344-347`) | auth R3 S1 | **none** | exit 69, correct | **UNEXERCISED** — W2 |
| 11 | `oauth.refresh.commit` success | auth R2 S1 | yes — `machine-contract:100-105` | ok | COVERED |
| 12 | `oauth.refresh.commit` stale/expired/persistence failure | auth R2 S2 | `machine_test.go:97,133,167` (in-process) | — | COVERED (unit) |
| 13 | **`oauth.refresh.commit` / `.abort` `invalid_lease`** (`machine.go:372,421`) | design §Interfaces | **none** | exit 75, retryable | **UNEXERCISED** — W7 |
| 14 | `oauth.refresh.abort` → `aborted` | auth R3 S1 | yes — `machine-contract:91-95` | ok | COVERED |
| 15 | `oauth.refresh.abort` → `quarantined` (terminal reasons) | auth R3 S1 | no — `machine_test.go:149-153` in-process only | quarantined, state updated | COVERED (unit); real-binary gap |
| 16 | `quota.exhaust` cooling, `replacement_available: true` | design §Interfaces | yes — `machine-contract:111-120`, `quota-integration:73-85` | ok | COVERED |
| 17 | **`quota.exhaust` cooling, `replacement_available: false`** | design §Interfaces | shape only — `machine-contract:127-133`; never mapped end-to-end | 429 + `Retry-After: 3600` | **UNEXERCISED (mapping)** — W1 |
| 18 | `quota.exhaust` `stale_generation` (non-replayed) | failover R7 S2 | `machine_test.go:406` (in-process) | exit 75 | COVERED (unit) |
| 19 | **`quota.exhaust` replayed with a stale generation** | failover R7 S2 | `machine_test.go:368-370` **asserts acceptance** | exit 0, `ok:true` | **CONTRADICTS SPEC** — W4 |
| 20 | **`quota.exhaust` `unknown_operation`** (`machine.go:447`) | design §Interfaces | **none** | exit 2, non-retryable | **UNEXERCISED** — W7 |
| 21 | `diagnostics.record` valid + rejected | auth R4 | yes — `machine-contract:135-141` | ok | COVERED |
| 22 | `diagnostics.status` valid + `invalid_request` + `state_unavailable` | auth R4 | yes — `machine-contract:137-141,180,186-187` | ok | COVERED |
| 23 | **`state_busy`** (`machine.go:201,318`) | design §Interfaces (exit 75) | **none** | exit 75, retryable | **UNEXERCISED** — W7 |
| 24 | **`invalid_operation`, Go dispatcher `default:`** (`machine.go:106`) | design §Interfaces | **none** (the JS test hits `machine.js`'s own allowlist, never the binary) | exit 2 | **UNEXERCISED** — W7 |
| 25 | `unsupported_version` (`v2`) | design §Interfaces | `machine_test.go:207` (in-process) | exit 2 | COVERED (unit) |
| 26 | 64 KiB stdin overflow | design §Threat Matrix | `machine_test.go:330` | exit 2 | COVERED |
| 27 | **`output_too_large`** (`machine.go:111-115`) | design §Interfaces | **none** | **unreachable**: full 256-event ring truncates to 64 → 6,658 B | **DEAD BRANCH** — S4 |
| 28 | Login clears only its own quarantine | auth R3 S1 | yes — `machine-contract:162-173` | ok | COVERED |

### Adapter response mapping (`quota.js:mapMachineResponse`)

Complete surface, driven with envelopes captured verbatim from the real binary:

| Machine outcome | HTTP | `Retry-After` | Body | In `design.md`? | Real-binary test |
|---|---|---|---|---|---|
| cooling, replacement available | 429 | *absent* | `{outcome:cooling,retryable:true}` | yes | yes |
| cooling, no replacement | 429 | `3600` | `{outcome:cooling,retryable:true}` | yes | **no** (W1) |
| `no_available_profile` cooling (select) | 429 | `90` | `{outcome:cooling,retryable:true}` | yes | yes |
| `credential_quarantined` | 401 | *absent* | `{action:"acm login",outcome:quarantined,retryable:false}` | yes | yes |
| `no_available_profile` non-retryable | **503** | absent | `{code,outcome:unavailable,retryable:false}` | **no** | **no** (W3) |
| `state_busy` (exit 75, **retryable**) | **503** | absent | `{code,outcome:unavailable,retryable:true}` | **no** | **no** (W3/W7) |
| `state_unavailable` (exit 74) | **503** | absent | `{code,outcome:unavailable,retryable:false}` | **no** | yes — `quota-integration:100-107` |
| `unknown_operation` (exit 2) | **503** | absent | `{code,outcome:unavailable,retryable:false}` | **no** | **no** (W3/W7) |
| `invalid_lease` (exit 75, **retryable**) | **503** | absent | `{code,outcome:unavailable,retryable:true}` | **no** | **no** (W3/W7) |
| unconfirmed 401 / 429 / 529 | passthrough | unchanged | unchanged object identity | yes | yes — `quota-integration:91-98` |

Four codes collapse into one undocumented 503 shape. Two of them (`state_busy`,
`invalid_lease`) are **retryable at the machine layer** but surface with no `Retry-After`,
losing the retry signal at the HTTP layer OpenCode's `SessionRetry` actually reads.

## Spec Compliance Matrix

| Req | Scenario | Status | Evidence |
|---|---|---|---|
| auth R1 | Selected profile supplies authentication | PASS | `quota-integration:43-50,73-88` (real binary) |
| auth R1 | Non-ACM or unsupported host refused | PASS | `compat.test.js:23-27` |
| auth R2 | Normal expiry refresh succeeds | PASS | `machine-contract:97-105`; `quota-integration:69-82` |
| auth R2 | Stale or failed refresh commit | PASS | `machine_test.go:97-131` |
| auth R3 | Refresh credentials are revoked | PASS | `machine_test.go:149-165`; adapter `quota.test.js:108-122` |
| auth R3 | **Unsupported version attempts a sensitive operation** | **FAIL** | **C3** — matrix removed by ADR; spec never amended; no gate, no test |
| auth R4 | Doctor collects a failed refresh event | PASS | `machine_test.go:259`; `main_test.go:12`; live `acm doctor` |
| failover R5 | Confirmed exhaustion selects another profile | PASS | `quota-integration:73-88` (real binary) |
| failover R5 | Generic rate-limit-like response | PASS | `quota-integration:91-98` (401/429/529, identity preserved) |
| failover R6 | OpenCode retries after transition | PASS | `quota-integration:86-88`; no timer/replay/queue in adapter |
| failover R7 | Multiple retries consume candidates | PASS | `machine-contract:125-133`; `machine_test.go:239` |
| failover R7 | Concurrent stale transition arrives | **PARTIAL** | rejected when not replayed; **accepted on replay** — W4 |
| failover R8 | Cooling profile supplies retry metadata | PASS | `machine-contract:144-150`; `quota-integration:53-57` |
| failover R8 | Only quarantined profiles remain | PASS | `machine-contract:152-160`; `quota-integration:59-63` |
| failover R8 | Cooling and quarantined mixed | PASS | `machine-contract:175-179`; `quota-integration:65-67` |
| lifecycle R9 | **Fresh ACM installation** | **FAIL** | **C1** — installer aborts; plugin never bundled |
| lifecycle R9 | User explicitly enables the experiment | PASS | `opencode_lifecycle_test.go:64-73` (blocked in the field by C1) |
| lifecycle R10 | Confirmed migration from upstream plugin | PASS | `opencode_lifecycle_test.go:64-73` |
| lifecycle R10 | **Plugin conflict is detected** | **FAIL** | **C2** — proceeds silently, exit 0; test asserts the opposite |
| lifecycle R11 | Rollback after experimental use | PASS | `opencode_lifecycle_test.go:71-73` |
| lifecycle R11 | Backup is missing or invalid | PASS | `opencode_lifecycle_test.go:58-61,74-78` |

**17/21 PASS · 1 PARTIAL · 3 FAIL** · Requirements fully satisfied: **8/11**.

## Guard Audit by Mutation

Baseline SHA-256 recorded, mutations applied, files restored and re-verified byte-identical.

| # | Mutation | Expected | Observed | Verdict |
|---|---|---|---|---|
| 1 | `machine.go:556` `replacement_available` → `replacement_ready` | guard fails | **fail 1** — `+ 'replacement_ready' / - 'replacement_available'` | guard is LIVE |
| 2a | negative control field → `bogus_field`, `message: /retry_after/` **kept** | guard fails | **fail 1** — `message: /retry_after/` unmatched | strengthening WORKS |
| 2b | same bogus field, regex binding **removed** | guard passes (round-2 weakness) | **pass 1** | weakness was real; task 10.3 closes it |

Restoration verified:
```
526a725ba0eb8bd56300513e3f10ce8aea027c2513c686adbbb08d4d36c6643f  machine.go
7866a0f0ecde3719901079039283a3e322fbe7f255e244d6803f23088a4d2325  machine-contract.test.js
```
identical to baseline; `git diff --stat HEAD` empty.

## Additional Verification

**Fixture fidelity** — no synthetic fixture asserts a field the real binary does not emit.
`quota.json.selection` is a strict subset of the real `credential.select` keys.
`machine-stub.js` emits an extra `args` field, but it is a Node stub for
`machine-process.test.js` boundary tests and is never claimed as binary output. One weakness:
`quota.test.js` stubs omit `replacement_available`, so the `Retry-After` expectation silently
rests on `undefined !== true` (S5).

**Ownership boundaries** — clean. No `setTimeout`/`setInterval`/`setImmediate`/sleep/delay
anywhere in `integrations/opencode/*.js`. Exactly one provider call per attempt
(`index.js:64`), plus the OAuth token endpoint (`oauth.js:10`). No queue, replay, stream
resumption, or agent continuation. The only two `while` loops are a bounded array shrink
(`diagnostics.js:21`) and the 4 KiB-capped evidence reader (`quota.js:14`).

**Secret safety** — with tokens `TOKEN-SECRET-AAA/BBB` in a live profile: 0 hits in
`acm doctor` stdout, 0 in stderr, 0 in the persisted state file. State file `0600`, credential
file `0600`. `machine.go:266-275` allowlists diagnostic component/event/outcome to fixed sets,
collapsing anything else to `"unknown"` — verified live: a request carrying a full profile
path, a token, and `private-identifier` persisted as `{component:"unknown", event:"unknown",
outcome:"unknown"}`. Machine subprocess output is never exposed
(`machine-process.test.js:24-33`). **One exception: W6.**

**Bounded state** — verified live against the real binary:

| Bound | Injected | Persisted | Cap |
|---|---|---|---|
| Diagnostics ring, count | 900 | **256** | `machineDiagnosticMax` |
| Diagnostics ring, age | 50 aged 25 h | **1** (only the fresh event) | 24 h TTL |
| Operation ledger, count | 3,000 | **1024** | `machineLedgerMax` |

Both structures are bounded by count **and** age, as `design.md` requires.

**Per-commit coherence** — all 15 commits build and pass their own Go tests; the 12 commits
that ship JS tests pass those too. Test count drops 33 → 24 at `06b9ab6`; the delta is
entirely the deleted compatibility-matrix cases, consistent with the ADR.

**Safety rule compliance** — `~/.config/opencode/` was never written. No
`.acm-opencode-backup.json` and no `*.acm-backup` exist there. Every lifecycle test used
`ACM_OPENCODE_CONFIG_HOME` pointing at `mktemp -d`. No credentials read, no OAuth login,
`install.sh` never executed (only simulated offline in `/tmp` with a fake `curl`). Probe
binaries built in `/tmp` and deleted; the per-commit worktree was created outside the repo and
removed. Read-only apart from this report.

`~/.config/opencode/` contains **both** `opencode.json` and `opencode.jsonc`, so
`findOpenCodeConfig` (`opencode_lifecycle.go:127-144`) would correctly abort as ambiguous —
confirmed by inspection, not by execution.

---

## Findings

### CRITICAL

#### C1 — `install.sh` fetches the deleted `compatibility.json`; the installer aborts and never bundles the plugin

**File**: `install.sh:66` · **Introduced by**: `06b9ab6` · **Breaks**: lifecycle R9 S1

Commit `06b9ab6` deleted `integrations/opencode/compatibility.json` but did not update the
installer's fetch list:

```sh
install.sh:14   set -eu
install.sh:66   for file in index.js machine.js oauth.js compat.js quota.js diagnostics.js compatibility.json package.json; do
install.sh:68     curl -fsSL "$raw" -o "$plugin_tmp/$file"
```

`curl -f` fails on HTTP 404 and `set -e` aborts the script. Offline simulation of lines 63–77
with the HEAD file list, using a fake `curl` that 404s only on the missing name:

```
=== simulating install.sh lines 63-77 with the HEAD file list ===
curl: (22) The requested URL returned error: 404
loop_exit=22
share dir contents: <<< NOT CREATED — plugin was never bundled >>>

=== control: same loop with compatibility.json removed from the list ===
BUNDLED OK
loop_exit=0
share dir contents: compat.js diagnostics.js index.js machine.js oauth.js package.json quota.js
```

Impact: `acm` itself installs (line 59), then the script dies before `$SHARE_DIR/opencode` is
created (line 74), before the shell aliases are added (line 108), and before `acm init`
adopts existing logins (line 118). `acm opencode enable` then fails at
`opencode_lifecycle.go:60-61` with *"el adaptador OpenCode de ACM no está instalado"*, so the
entire `acm-opencode-plugin-lifecycle` capability is unreachable in the field — the exact
failure class round 3 rejected, relocated from the load boundary to the distribution boundary.

Why the suite stayed green: `sh -n install.sh` validates syntax only, and
`grep -rn 'install.sh' --include='*.go' --include='*.js'` returns **nothing** — `install.sh`
has no behavioural test at all. The round-3 frozen-boundary discipline that kept out-of-scope
files byte-identical is precisely what let this through.

#### C2 — Plugin-conflict detection does not stop; the test asserts the opposite of the spec

**Files**: `opencode_lifecycle.go:185-195`, `opencode_lifecycle_test.go:52-56` ·
**Breaks**: lifecycle R10 S2 · **Round-1 W5, still open and now sharper**

The spec requires:

> #### Scenario: Plugin conflict is detected
> - GIVEN both ACM and upstream plugin entries are present
> - THEN the lifecycle command MUST stop with a conflict until guided resolution is confirmed.

`editOpenCode` instead filters **both** entries and appends the ACM URL unconditionally:

```go
opencode_lifecycle.go:187   upstream := value == "opencode-anthropic-login-via-cli" || strings.HasPrefix(...)
opencode_lifecycle.go:188   acm := value == pluginURL || strings.HasSuffix(value, "/acm/opencode/index.js")
opencode_lifecycle.go:189   if !acm && (!enable || !upstream) { result = append(result, value) }
opencode_lifecycle.go:194   result = append(result, pluginURL)
```

`enableOpenCode` returns `nil`, so `runOpenCodeLifecycle` prints success and returns `0`. There
is no conflict detection and no conflict-specific confirmation — `--confirm` is a blanket flag
required for *every* invocation (`opencode_lifecycle.go:22`), not guided resolution.

The existing test constructs exactly the both-present conflict and asserts the contradiction:

```go
opencode_lifecycle_test.go:52  "...{\"plugin\":[\"opencode-anthropic-login-via-cli@1.6.1\",\"file:///tmp/acm/opencode/index.js\"],}"
opencode_lifecycle_test.go:55  check(t, code == 0 && ...)
opencode_lifecycle_test.go:56  check(t, ... && !strings.Contains(string(changed), "opencode-anthropic-login-via-cli"), ...)
```

The scenario has no covering test, and the test that touches the state locks in the opposite
behavior. Either the code must stop on conflict or the spec must be amended — the current pair
cannot both be right.

#### C3 — The compatibility requirement was invalidated by ADR 0001 but the spec was never amended

**File**: `specs/acm-opencode-claude-auth/spec.md:44,52-56` · **Breaks**: auth R3 S2

The spec still mandates:

> Compatibility-sensitive operations MUST block outside the explicit supported OpenCode/Claude
> CLI matrix.
>
> #### Scenario: Unsupported version attempts a sensitive operation
> - GIVEN the OpenCode or Claude CLI version is outside the supported matrix
> - THEN the plugin MUST block safely and report the incompatibility.

`06b9ab6` deliberately removes that matrix. `compat.js:22-25` now only *observes* the Claude CLI
version, and `assertCompatibility` (`compat.js:27-30`) gates on platform and ACM-managed status
only. My live probe confirms the plugin loads against a `9.9.9` CLI and against no CLI at all.

The removal is correct and well argued in `docs/03-architecture/decisions/0001-use-ecosystem-plugin-compatibility.md`,
which is Accepted and user-approved. The defect is that the **authoritative spec file was not
updated as part of the change**, so verification is measured against a requirement the change
intentionally abandoned. This is a real SDD coherence failure: it leaves the requirement
permanently unsatisfiable and guarantees this exact contradiction resurfaces in a future round.
Fix by amending the spec, not the code.

### WARNING

- **W1 — `quota.exhaust` cooling with `replacement_available: false` is never mapped end-to-end
  against the real binary.** `machine-contract.test.js:127-133` asserts only the machine-side
  shape; `quota-integration.test.js:83` exercises only the replacement-available branch. My
  live probe shows the mapping is correct (`429` + `Retry-After: 3600`), but nothing guards it.
  *Round-3 W1: still open.*

- **W2 — `oauth.refresh.begin` returning `credential_quarantined` has zero coverage.**
  `machine.go:344-347`; live probe confirms exit `69`, `retryable:false`. Round-3 W2 partially
  closes: abort→quarantine **is** covered (`machine_test.go:149-153`, all three terminal
  reasons) and select→quarantine **is** covered (`machine_test.go:297-311`); the `begin` branch
  is the residual. *Partially closed.*

- **W3 — The 503 shape is undocumented and mostly unexercised.** Four codes collapse into it;
  only `state_unavailable` is exercised (`quota-integration:100-107`). `design.md` §Interfaces
  enumerates exactly four outcomes and none is a 503. Worse, `state_busy` and `invalid_lease`
  carry exit `75` / `retryable:true` at the machine layer yet surface as 503 with no
  `Retry-After`. `state_busy` is the *expected* outcome of two concurrent OpenCode instances —
  the very concurrency `design.md` §Data Flow is built around. *Round-3 W3: partially closed,
  substantively open.*

- **W4 — Stale generation accepted on a replayed `quota.exhaust`.** `machine.go:451` precedes
  `machine.go:459`. Live-proven with a working control:
  ```
  replay,     profile alpha, generation 1 (actual 2) -> exit 0,  ok:true
  non-replay, profile beta,  generation 1 (actual 3) -> exit 75, stale_generation
  ```
  `machine_test.go:368-370` asserts the accepting behavior, so this is deliberate idempotency,
  not an oversight. It is **non-corrupting** — the branch returns before `saveMachineState`, so
  newer state is never overwritten and the response carries the *current* generation. But it
  contradicts failover R7's literal "MUST … reject stale generations" and is undocumented in
  `design.md`. Document the idempotency exception or move the generation check above line 451.
  *Round-3 W4: still open, now precisely characterised.*

- **W5 — escalated to C2.**

- **W6 — ENOENT leaks a filesystem path out of `auth.fetch`.** `index.js:25` reads the
  credential file; `index.js:54-57` maps only errors carrying `.machine` and rethrows everything
  else raw. Live-proven with control:
  ```
  THROWN message: ENOENT: no such file or directory, open '/tmp/.../VERY-PRIVATE-PROFILE-DIR/.credentials.json'
  LEAKS PATH?   : YES — full profile path escaped
  control (valid credential): status 200 body ok-control
  ```
  Not a credential leak, but it discloses the ACM directory layout and profile name into
  OpenCode's error surface and logs, against the security baseline's "hide internals in errors".
  The machine boundary is correctly hardened; this read boundary is not. No test.
  *Round-3 W6: still open.*

- **W7 (new) — Four machine error codes have zero coverage anywhere.** `unknown_operation`
  (`machine.go:447`), `state_busy` (`machine.go:201,318`), `invalid_lease`
  (`machine.go:372,421`), and the dispatcher `default:` `invalid_operation` (`machine.go:106`).
  All four verified correct by live probe; none is guarded by any Go or JS test. The JS
  `invalid_operation` test (`machine-process.test.js:21`) hits `machine.js`'s own allowlist and
  never reaches the binary. `unknown_operation` is reachable in production once the 1024-record
  ledger evicts an in-flight operation, and it turns a genuine provider 429 into a
  non-retryable 503.

### SUGGESTION

- **S1 — `fsync` misses the directory entry.** `atomicWriteMachineFile` (`machine.go:623-641`)
  syncs the temp file (`machine.go:632`) but never opens and syncs `filepath.Dir(path)` after
  `os.Rename` (`machine.go:640`). A crash can lose the rename even though the data was durable.
  Affects both the state file and the committed credential file. *Round-3 S1: still open.*

- **S2 — `compat.test.js:18-21` is self-referential.** It reads `package.json` and asserts that
  `package.json` says `^1.18.18` — the same structural shape as round-3 C1 (comparing the pinned
  file to itself). Harmless now that nothing gates on it, but it proves nothing about the
  ecosystem. There is no lockfile and no `node_modules`, so no installed-range evidence exists.

- **S3 — `ACM diagnostics: record_failed` on stderr is correct but noisy.** Assessment: this is
  **acceptable best-effort behavior, not a defect.** The string is a fixed, redacted code with
  no path, token, or identifier; it fires on plugin *invocation* (not bare import — my probe
  confirms a bare `import` is silent); and the plugin still returns fully functional hooks
  (`index.js:19-22` never throws). It is the correct observable signal for a diagnostics sink
  that could not reach `acm`. The only real objection is that it is unconditional and
  unsilenceable, and in normal operation `acm` is always on PATH — so if it fires, something is
  genuinely wrong and the user should see it. Recommend keeping it; optionally emit once per
  process rather than per load.

- **S4 — `output_too_large` is dead code.** `machine.go:111-115` cannot trigger:
  `diagnostics.status` truncates to the last 64 events (`machine.go:249-251`), producing 6,658
  bytes with a completely full 256-event ring — far under the 16 KiB cap. Verified live. Keep as
  defence-in-depth or delete, but do not count it as coverage.

- **S5 — `quota.test.js` stubs omit `replacement_available`.** Lines 28, 50, 56, 100 return
  `{outcome, generation, reset_at}`. The `Retry-After: 45` expectation at line 35 therefore
  depends on `undefined !== true` rather than a modelled state. Add
  `replacement_available: false` so the stub states which of the four design outcomes it models.

- **S6 — `validMachineRequest` is lax for read operations.** `machine.go:159` does not constrain
  `Profile`, `Generation`, `LeaseID`, or `ExpiresAt` on `credential.select`,
  `diagnostics.status`, `oauth.refresh.begin`, or `oauth.refresh.abort`. The fields are ignored,
  so there is no exploit, but the strict-boundary posture stated in `design.md` §Interfaces
  ("unknown fields/versions fail") is only partially realised.

---

## Per-Finding Closure Verdicts

| Round | Item | Verdict | Note |
|---|---|---|---|
| R3 | **C1** pinned matrix matches no shipping release | **CLOSED** | entry point loads; matrix/resolver/checker removed; ADR 0001 recorded |
| R3 | W1 cooling via `quota.exhaust` (`replacement_available:false`) | **OPEN** | machine shape asserted; adapter mapping still unexercised |
| R3 | W2 quarantine via `oauth.refresh.begin` | **PARTIALLY CLOSED** | abort→quarantine and select→quarantine covered; `begin` branch still unexercised |
| R3 | W3 undocumented `503` for `no_available_profile` | **PARTIALLY CLOSED** | a 503 is now exercised, but for `state_unavailable`; shape still undocumented and 3 of 4 codes unexercised |
| R3 | W4 stale generation on replayed `quota.exhaust` | **OPEN** | live-proven; deliberately asserted at `machine_test.go:368`; non-corrupting |
| R3 | W5 plugin conflict returns `0` and silently rewrites | **OPEN → ESCALATED to C2** | test asserts the contradiction |
| R3 | W6 ENOENT leaks a filesystem path | **OPEN** | live-proven with control |
| R3 | S1 `fsync` misses the directory entry | **OPEN** | `machine.go:640` |
| R3 | `opencode.json` + `opencode.jsonc` both present → ambiguous abort | **CONFIRMED CORRECT** | `opencode_lifecycle.go:127-144`; covered by `TestOpenCodeMigrationRollbackOnJSONCConflict` |
| R2 | `machineQuotaResponse` always cooling + `Retry-After` | **CLOSED** | `replacement_available` added; mutation-proven guard; live 429-without-header |
| R2 | negative control passes on a bogus field name | **CLOSED** | mutation 2a fails, 2b proves the prior weakness |
| R1 | `quota.exhaust` in JS but not Go | **CLOSED** | real-binary contract test spawns compiled `acm` |
| R1 | `retry_after` vs `reset_at` key drift | **CLOSED** | mutation 1 proves the guard detects live key drift |
| R1 | compatibility gate compared the pinned file to itself | **CLOSED** | gate removed entirely (see C3 for the spec-side residue) |
| R1 | diagnostics never ran in production | **CLOSED** | `index.js:11-22` production sink; `quota-integration:79,85` asserts the real `diagnostics.record` call |
| R1 | 3 of 4 `design.md` outcomes implemented | **CLOSED** | all four present in one mapping path (`quota.js:46-67`) |

## Required Before This Change Can Pass

1. **C1** — remove `compatibility.json` from `install.sh:66`, and add a test that asserts the
   installer's file list equals the set of files actually shipped in
   `integrations/opencode/`. Without that test this class recurs on the next file rename.
2. **C2** — either implement conflict detection that stops with a non-zero exit until guided
   resolution, or amend lifecycle R10 S2. Then make `opencode_lifecycle_test.go:51-62` assert
   the chosen contract instead of the current contradiction.
3. **C3** — amend `specs/acm-opencode-claude-auth/spec.md` R3 so the compatibility requirement
   matches ADR 0001. Code change not required.
4. **W1–W7** — close the coverage states above. Four rounds in a row the failure has been an
   unexercised documented state; adding these guards is what breaks the pattern.

## Final Verdict

**FAIL** — 3 CRITICAL, 6 WARNING, 6 SUGGESTION.

Round-3 C1 is genuinely closed and the round-1/round-2 defects hold under mutation. But the
fix that closed C1 opened C1' at the installer boundary, and two specification scenarios remain
contradicted by the code and by their own tests. The change is not ready to merge.

*No file was fixed. All mutated files restored byte-identical and checksum-verified. Read-only
apart from this report.*
