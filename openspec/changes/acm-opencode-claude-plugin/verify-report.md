```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:3172e9d98c9731f1e9e30c39403e7cf3cbc506a2709d86459804457dca8ab9c9
verdict: fail
blockers: 1
critical_findings: 1
requirements: 8/11
scenarios: 18/21
test_command: go test -count=1 ./... && node --test "integrations/opencode/test/*.test.js"
test_exit_code: 0
test_output_hash: sha256:2bedbb7245b57778fda32c88dfe20afc79abb5b3918093025b072ecb42220fb7
build_command: gofmt -l . && go vet ./... && git diff --check && sh -n install.sh
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verification Report

- **Change**: `acm-opencode-claude-plugin`
- **Round**: 5 (independent final verification of the complete R7–R11 remediation chain)
- **Tip**: `533555b` on `feat/opencode-plugin-r11-auth-error-containment`, 21 commits from `main`, worktree clean
- **Mode**: Strict TDD, full spec-driven verification (proposal + 3 specs + design + tasks present)
- **Artifact store**: hybrid (OpenSpec + Engram, project `acm`)
- **Verdict**: **FAIL** — 1 CRITICAL
- **Merge safety**: **NOT SAFE TO MERGE**

## Executive Summary

The R7–R11 chain is substantially real work, not cosmetic. Every command exits `0`; the Go
suite and all 29 Node tests pass. I independently confirmed that **all three round-4 CRITICALs
are genuinely closed**, and I proved four of the new guards are *live* by mutation. Most
importantly, I traced the whole user-facing capability myself — real `install.sh` in an
isolated `HOME`, real compiled `acm`, real `opencode enable`, real ESM load — and the default
path works end to end. That is the first time in five rounds this has been true.

The chain nevertheless fails, and it fails **in exactly the pattern this change keeps
repeating**: a slice fixed its own finding and silently removed adjacent behavior that no test
owned. Commit `dbd7115` (R9) closed C2 by moving backup creation under `if replaceUpstream`.
The consequence is that the **primary opt-in path** — `acm opencode enable --confirm` on a
configuration that has no upstream plugin, i.e. what every new user runs — now **mutates the
user's OpenCode configuration while creating no backup and no manifest**, after which
`acm opencode rollback --confirm` exits `2` and refuses to help. ACM performs an edit it
cannot undo. No test covers that path: every lifecycle test either seeds the upstream plugin
or asserts a *failure* branch, so the suite stayed green.

This is the same defect class as round-4 C1 (a slice's fix breaking a neighbouring boundary),
and the same class as C3 (behaviour changed without amending the specification). Task 14.3's
REFACTOR evidence explicitly claims "diff review found no out-of-scope behavior change"; the
diff contradicts it.

Two further latent instances of the same class survive: `ACM_SHARE_DIR` is honoured by the
installer but ignored by the lifecycle command, and `README.md` still advertises the pinned
compatibility matrix that ADR 0001 deleted.

## Completeness

| Dimension | Status | Evidence |
|---|---|---|
| Tasks checked | 54/54 | `tasks.md`, `grep -c '^- \[x\]'` = 54, `'^- \[ \]'` = 0 |
| Proposal | present | `proposal.md` |
| Specs | 3 present | auth, failover, lifecycle |
| Design | present | `design.md` |
| Requirements | 11 total | 4 auth + 4 failover + 3 lifecycle (counted from spec files) |
| Scenarios | 21 total | 7 auth + 8 failover + 6 lifecycle (counted from spec files) |

## Command Evidence

| Command | Exit | Result |
|---|---|---|
| `go test -count=1 ./...` | 0 | `ok github.com/Gefermanpernia/acm 15.622s` |
| `node --test "integrations/opencode/test/*.test.js"` | 0 | tests 29, pass 29, fail 0 |
| `gofmt -l .` | 0 | no output |
| `go vet ./...` | 0 | clean |
| `git diff --check` | 0 | clean |
| `sh -n install.sh` | 0 | **syntax only — insufficient; see "Why `sh -n` is not evidence"** |
| `go test -count=1 -cover ./...` | 0 | 45.1% of statements |

Node v26.7.0. `bun` never invoked.

### Why `sh -n` is not evidence

Round-4 C1 shipped while `sh -n install.sh` was green. I re-proved that blindness directly.
Removing `quota.js` from the installer's fetch list (mutation MUT-4 below) leaves
`sh -n install.sh` at exit `0`, while the new `install.test.js` correctly fails. `sh -n`
validates grammar and nothing else; it must never be cited as installer verification.

## PRIMARY OBLIGATION — End-to-End Capability Trace (independent)

Not a re-run of the implementer's tests. My own harness: `env -i`, temporary `HOME`, `TMPDIR`,
offline fake `curl` serving repository bytes, the real compiled `acm` as the release artifact.

```
STEP 1  real `sh install.sh`, isolated                     install_exit=0
STEP 2  staged at $HOME/.local/share/acm/opencode:
        compat.js diagnostics.js index.js machine.js oauth.js package.json quota.js   (7/7)
STEP 3  clean-process ESM load of the STAGED entry point
        bare import: OK, default is function
        factory invoke: OK, hooks = auth,chat.headers
STEP 4  `acm opencode enable --confirm` (default discovery)  enable_exit=0
        config: {"model":"anthropic/claude",
                 "plugin":["file:///.../.local/share/acm/opencode/index.js"]}
STEP 5  the path the config actually points at
        ENABLED PLUGIN LOADS: hooks = auth,chat.headers
host safety: real $HOME/.local/share/acm/opencode absent; sandbox removed
```

**All five links hold on the default path.** Round-3's "refuses to load" and round-4's "never
gets installed" are both genuinely gone. This is verified capability, not unit greenness.

### Cross-slice agreement (R7 contract × R8 assets)

R7 removed the version matrix and its resolver; R8 reduced the installer to seven assets. They
agree. The static import graph closes exactly over the shipped set:

| Module | Imports |
|---|---|
| `index.js` | `./compat.js`, `./machine.js`, `./oauth.js`, `./quota.js` |
| `quota.js` | `./diagnostics.js` |
| `compat.js`, `machine.js` | node builtins only |
| `oauth.js` | none |
| `package.json` | declares `@opencode-ai/plugin ^1.18.18`; never imported at runtime |

No asset the R7 contract needs is missing from the R8 fetch list. The C1 defect class is not
reintroduced — and `install.test.js` now imports the *staged* entry point, so an omitted
statically-imported module fails the suite (proved by MUT-4).

## Guard Audit by Mutation

Baseline SHA-256 recorded, mutation applied, suite run, file restored, checksums re-verified.

| # | Mutation | Expected | Observed | Verdict |
|---|---|---|---|---|
| 1 | `quota.js:54` ignore `replacement_available` (always set `Retry-After`) | fail | exit 1 | guard **LIVE** |
| 2 | `index.js` rethrow the raw credential error (undo W6) | fail | exit 1 | guard **LIVE** |
| 3 | `machine.go` drop the parent-directory fsync (undo S1) | fail | exit 1, `sync order = [file]` | guard **LIVE** |
| 4 | `install.sh` drop `quota.js` from the fetch list (C1 class) | fail | `sh -n`=0, `install.test.js` exit 1 | guard **LIVE** |

Restoration verified byte-identical:

```
f2a34ef22c6ddd0595150630cfb3a3b19666a94c116d8f06ae382332faf9a128  integrations/opencode/quota.js
9d55f94c691868d9246bc78eaf5c0a89a2190b904bd08fd5f748cc2f83f033e1  integrations/opencode/index.js
5171b7734e71666d917d008f5dc1b9083d65e9565bf6d7fded466e84e8965a66  machine.go
8d8621a1e8ee0e6daaaba667b53bc7778530372e82dbd687f999361d8b3c7b4a  install.sh
```
`git status --porcelain` empty after restoration.

## Regression Guards Required by This Round

| Guard | Result | Evidence |
|---|---|---|
| `Retry-After` omitted when `replacement_available === true` | ✅ HELD | `quota.js:54`; `quota-integration.test.js:118` asserts `null` against the real binary; MUT-1 proves the guard is live |
| Cooling and quarantine remain distinct outcomes | ✅ HELD | cooling → `429 {outcome:cooling,retryable:true}`; quarantine → `401 {action:"acm login",outcome:quarantined,retryable:false}`; live at `quota-integration.test.js:90-98` |
| Quarantine names `acm login` | ✅ HELD | `quota.js:51`; `machine-contract.test.js:231` asserts `/acm login/` in the binary's own message |
| Exit taxonomy 0/2/69/74/75 unchanged | ✅ HELD | `2` invalid/unsupported/unknown_operation; `69` quarantined/no_available_profile; `74` persistence/state_unavailable; `75` state_busy/lease_busy/invalid_lease/stale_generation. Verified live per code. |
| Responses and diagnostics secretless | ✅ HELD (scoped) | See below |

### Secretlessness probe (live, real binary, hostile input)

Injected `TOKEN-SECRET-AAA/BBB` and profile `MY-PRIVATE-ACCOUNT`, then submitted a
`diagnostics.record` carrying a full profile path, a token, and a private identifier:

```
persisted state: clean TOKEN-SECRET-AAA · clean TOKEN-SECRET-BBB · clean leak-me · clean /profiles/
diagnostics.status → {component:"unknown", event:"unknown", outcome:"unknown", retryable:true}
state file mode: 600
```

Tokens, paths, and hostile identifiers are all collapsed or absent. Adapter error bodies expose
only `code`/`outcome`/`retryable`/`action` — never a profile.

**Scoped exception, by design, not a defect**: `credential.select` returns `profile` and
`config_dir` in its success response. The adapter cannot read `.credentials.json` without them
(`design.md` §Data Flow), and auth R1 constrains only tokens. The design's path prohibition is
scoped to *diagnostics*, which is satisfied. W6's fix closed the one place this used to escape
into OpenCode's error surface.

## Spec Compliance Matrix

| Req | Scenario | Status | Evidence |
|---|---|---|---|
| auth R1 | Selected profile supplies authentication | ✅ COMPLIANT | `quota-integration.test.js:54-65,108-124` (real binary) |
| auth R1 | Non-ACM or unsupported host refused | ✅ COMPLIANT | `compat.test.js`; my probe: `darwin -> refused: unsupported platform` |
| auth R2 | Normal expiry refresh succeeds | ✅ COMPLIANT | `machine-contract.test.js:112-117`; `quota-integration.test.js:108-117` |
| auth R2 | Stale or failed refresh commit | ✅ COMPLIANT | `machine_test.go:113-129` (write/fsync failure, credential unchanged, no secret leak) |
| auth R3 | Refresh credentials are revoked | ✅ COMPLIANT | `machine-contract.test.js:154-161` → 401 `acm login`; abort-quarantine in `machine_test.go` |
| auth R3 | Claude CLI evidence is diagnostic only | ✅ COMPLIANT | **C3 closed.** `contract-coherence.test.js`; my probe: no CLI / `9.9.9` / garbage all load |
| auth R4 | Doctor collects a failed refresh event | ✅ COMPLIANT | `machine-contract.test.js:210-217`; live secretless probe |
| failover R5 | Confirmed exhaustion selects another profile | ✅ COMPLIANT | `quota-integration.test.js:108-124` (real binary rotation alpha→beta) |
| failover R5 | Generic rate-limit-like response | ✅ COMPLIANT | `quota-integration.test.js:135-142` (401/429/529, object identity preserved) |
| failover R6 | OpenCode retries after transition | ✅ COMPLIANT | `quota-integration.test.js:115,123`; one provider call per attempt; no timer/replay in adapter |
| failover R7 | Multiple retries consume candidates | ✅ COMPLIANT | `machine-contract.test.js:137-145` |
| failover R7 | Concurrent stale transition arrives | ⚠️ PARTIAL | rejected when not replayed; **accepted on replay** — see W5, re-proved live this round |
| failover R8 | Cooling profile supplies retry metadata | ✅ COMPLIANT | `quota-integration.test.js:88-92` (`retry-after: 90`) |
| failover R8 | Only quarantined profiles remain | ✅ COMPLIANT | `quota-integration.test.js:94-98` |
| failover R8 | Cooling and quarantined mixed | ✅ COMPLIANT | `quota-integration.test.js:100-102` (`retry-after: 150`, cooling-only derivation) |
| lifecycle R9 | Fresh ACM installation | ✅ COMPLIANT | **C1 closed.** `install.test.js`; my independent E2E STEP 1–3 |
| lifecycle R9 | **User explicitly enables the experiment** | ❌ **FAILING** | **C1(R5)** — no test covers the successful plain opt-in; real behaviour leaves no restorable backup |
| lifecycle R10 | Confirmed migration from upstream plugin | ✅ COMPLIANT | `opencode_lifecycle_test.go:74-104` (real binary); my scenario D |
| lifecycle R10 | Plugin conflict is detected | ✅ COMPLIANT | **C2 closed.** My scenario C: exit 2, bytes preserved, no backup |
| lifecycle R11 | Rollback after experimental use | ⚠️ PARTIAL | works for `--replace-upstream` (scenario D restored byte-identical); **unreachable** for plain enable |
| lifecycle R11 | Backup is missing or invalid | ✅ COMPLIANT | `opencode_lifecycle_test.go:117-124` |

**18/21 COMPLIANT · 2 PARTIAL · 1 FAILING** · Requirements fully satisfied: **8/11**.

## Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| Versioned stdin/stdout boundary, secrets never in argv/stderr/responses | ✅ Yes | Live probe clean |
| Lease-spanning refresh, generation-checked commit | ✅ Yes | Real-binary begin/commit/abort |
| `fsync` file **and parent directory** after rename | ✅ Yes | `machine.go:640-648`; MUT-3 proves the guard |
| All 503 outcomes documented; retryable carry bounded `Retry-After: 1` | ✅ Yes | `design.md:33`; `machine-contract.test.js:194-208` |
| One provider call per attempt; no replay/queue/stream continuation | ✅ Yes | `quota-integration.test.js:115`; no timers in `integrations/opencode/*.js` |
| Compatibility governed by package range, CLI diagnostic-only | ✅ Yes | ADR 0001 + amended auth R3 |
| Guided enable/rollback with checksummed backups | ❌ **No** | Backup only on `--replace-upstream`; see C1(R5) |

## Strict TDD Sections

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ | `apply-progress.md` has a TDD Cycle Evidence table for R7–R11 |
| All tasks have tests | ✅ | 54/54 tasks map to a named test file |
| RED confirmed (test files exist) | ✅ | every referenced file present |
| GREEN confirmed (tests pass now) | ✅ | 29/29 Node, full Go suite |
| Triangulation adequate | ✅ | e.g. R8 covers success + rejected-asset; R10 covers 6 machine outcomes |
| Safety net for modified files | ⚠️ | reported for each slice, but **R9's claim is contradicted** — see W4 |

**TDD compliance**: 5/6 checks passed.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit (in-process) | Go `machine_test.go`, `main_test.go`; `compat/quota/diagnostics` JS | 5 | `go test`, `node --test` |
| Integration (compiled binary / real process) | `machine-contract`, `quota-integration`, `machine-process`, `opencode_lifecycle_test.go` | 4 | spawned `acm` |
| E2E (real installer process) | `install.test.js` | 1 | `sh install.sh` + offline fakes |
| **Total Node** | **29** | **7** | |

The real-binary and real-installer layers are the strongest part of this chain.

### Changed File Coverage

Go: **45.1% of statements** (whole package; no per-file coverage profile configured).
Node: **no coverage tool configured** — analysis skipped, not a failure.

### Assertion Quality

No tautologies, no ghost loops, no render-only smoke assertions found.

```
compat 20 · contract-coherence 8 · install 12 · machine-contract 44
machine-process 4 · quota-integration 30 · quota 19        (assert.* per file)
```

The round-2 negative control (`machine-contract.test.js:133-135`) still binds
`message: /retry_after/`, so a bogus field name cannot pass silently.

**Assertion quality**: ✅ All assertions verify real behavior.

### Quality Metrics

**Linter/formatter**: ✅ `gofmt -l .` clean. **Vet**: ✅ clean. **Whitespace**: ✅ `git diff --check` clean.

---

## Findings

### CRITICAL

#### C1(R5) — `enable --confirm` mutates the OpenCode config with no backup; `rollback` then refuses, leaving no undo path

**Files**: `opencode_lifecycle.go:91-103` · **Introduced by**: `dbd7115` (R9) ·
**Breaks**: lifecycle R10 requirement text, lifecycle R9 S2; makes lifecycle R11 S1 unreachable

R9 closed C2 correctly, but it also moved backup creation under a condition:

```go
opencode_lifecycle.go:95    if replaceUpstream {
opencode_lifecycle.go:96      rollback, _ := editOpenCode(original, pluginURL, false)
opencode_lifecycle.go:97      record := []byte(filepath.Base(path) + ":" + checksumOpenCode(rollback))
opencode_lifecycle.go:98      if atomicWriteMachineFile(backup, rollback) != nil || atomicWriteMachineFile(manifest, record) != nil {
...
opencode_lifecycle.go:103   }
opencode_lifecycle.go:104   if err = atomicWriteMachineFile(path, updated); err == nil {   // config IS written regardless
```

Before `dbd7115` those two writes were unconditional. Live proof, temporary config home,
temporary plugin path, no host state touched:

```
--- before --- {"model":"anthropic/claude"}
$ acm opencode enable --confirm
✓ Configuración actualizada. Reinicia OpenCode para aplicar el cambio.
enable_exit=0
--- after  --- {"model":"anthropic/claude","plugin":["file:///.../opencode/index.js"]}
--- files in config home --- opencode.json          <<< no .acm-opencode-backup.json, no .acm-backup

$ acm opencode rollback --confirm
acm: no existe un respaldo válido
rollback_exit=2
--- config after rollback --- {"model":"anthropic/claude","plugin":["file:///.../index.js"]}
```

**Impact.** This is the path a new user takes: install ACM, opt in, no upstream plugin present.
ACM writes to the user's `opencode.json`, reports success, and then its own documented rollback
command exits `2` and leaves the edit in place. The only recovery is hand-editing JSON — the
exact remedy ADR 0001 criticises as unacceptable. `README.md:80` tells the user
`acm opencode rollback --confirm` is how to "volver a la configuración respaldada"; for this
path no backup was ever taken.

**Spec position.** Lifecycle R10's normative text is unconditional: *"Migration MUST require
explicit confirmation, **create a restorable backup of the OpenCode configuration**, and enforce
mutual exclusivity."* Enabling writes the configuration through the same edit/validate/atomic
machinery and the same `.acm-opencode-backup.json` manifest, so it is a configuration migration
in this system's own vocabulary. The behaviour was changed without amending R10 or R11 —
violating the same-slice rule R7 had just written into ADR 0001 and task 12.3.

**Why the suite stayed green.** Every lifecycle test either seeds the upstream plugin or asserts
a *failure* branch. There is **no test of a successful plain `enable --confirm`**:

```
opencode_lifecycle_test.go:32   {"plugin":["upstream-json"]}        → ambiguous origins, fails
opencode_lifecycle_test.go:46   same home, validateOpenCode stubbed to fail → restores, fails
opencode_lifecycle_test.go:53   upstream present                    → conflict, fails
opencode_lifecycle_test.go:74   upstream present (real binary)      → conflict, fails
opencode_lifecycle_test.go:107  upstream present                    → requires --replace-upstream
```

**Required fix (choose one, then test it).** Either restore unconditional backup + manifest
creation for any config-mutating `enable`, or amend lifecycle R10/R11 to state that a
non-migration enable takes no backup **and** give the user a supported `disable`/`rollback`
path. In both cases add a test that performs a *successful* plain `enable --confirm` and
asserts the resulting undo contract.

### WARNING

- **W1 — `ACM_SHARE_DIR` is honoured by the installer and ignored by the lifecycle command.**
  `install.sh:19` resolves `SHARE_DIR="${ACM_SHARE_DIR:-$HOME/.local/share/acm}"`, but
  `opencode_lifecycle.go:55-58` falls back to a hardcoded `$HOME/.local/share/acm/opencode/index.js`
  and never reads `ACM_SHARE_DIR`. Proved live: installing with `ACM_SHARE_DIR` set succeeds,
  then `acm opencode enable --confirm` fails with *"el adaptador OpenCode de ACM no está
  instalado"*, exit `2`. This is the round-4 C1 defect class (two slices disagreeing about where
  assets live) still latent, saved only by the default path. The undocumented
  `ACM_OPENCODE_PLUGIN_PATH` is the sole escape. Either honour `ACM_SHARE_DIR` in
  `enableOpenCode` or drop it from `install.sh`.

- **W2 — `README.md:63` still advertises the compatibility matrix ADR 0001 deleted.**
  > "Solo admite Linux, perfiles ACM y **la matriz fijada OpenCode 1.18.19 / SDK 1.17.12 /
  > Claude CLI 2.1.236**."

  R6 removed that matrix, R7 amended auth R3 to make CLI detection diagnostic-only, and my probe
  confirms the plugin loads against no CLI, `9.9.9`, and garbage output. R9 edited `README.md`
  in this same chain and left line 63 untouched. This is C3 surviving at the user-facing
  documentation boundary — the same "authoritative artefact not amended in the deciding slice"
  failure the ADR forbids.

- **W3 — `acm doctor` silently lost the state directory line and the profile listing.**
  `git show main:main.go` ends `cmdDoctor` with `fmt.Println("estado : " + acmDir)` … `return cmdLs()`;
  at HEAD both are gone (`grep -c cmdLs` inside `cmdDoctor` = 0, `'estado : '` = 0). Removed by
  `249633f` (R4a) while adding diagnostics aggregates. No requirement authorises the removal —
  auth R4 constrains what doctor MUST NOT *contain*, not that it must stop listing profiles.
  `acm ls` still exists, so this is recoverable, but it is out-of-scope behaviour deleted during a
  refactor and it went unflagged through four verification rounds.

- **W4 — Task 14.3's REFACTOR evidence is contradicted by the diff.**
  `apply-progress.md` (R9, task 14.3) states *"diff review found no out-of-scope behavior
  change"*. The diff shows `enableOpenCode` losing unconditional backup creation, which changes
  fresh-enable behaviour. This inaccurate self-assessment is the mechanism by which C1(R5)
  reached this round. Task honesty finding, not a code finding.

- **W5 — failover R7 S2 remains PARTIAL: a replayed `quota.exhaust` still accepts a stale
  generation.** Re-proved live against the compiled binary this round:
  ```
  first exhaust  (gen 1) -> exit 0, ok:true, generation:2
  REPLAY same op+profile, STALE gen 1 -> exit 0, ok:true, generation:2   <<< accepted
  ```
  `machine.go:451` precedes the generation check at `machine.go:459-460`. It is **non-corrupting**
  — it returns before `saveMachineState` and echoes the *current* generation — but it contradicts
  failover R7's literal "MUST reject stale generations" and is still undocumented in `design.md`.
  Round-4 W4; deliberately not planned for remediation. See the W4 note below.

- **W6 — No lifecycle test covers a successful plain `enable --confirm`.** Enumerated under
  C1(R5). This coverage hole is what allowed a config-mutating command to lose its undo path
  without a single test turning red, and it will allow the next such change too.

### SUGGESTION

- **S1 — `Retry-After` is unbounded.** `quota.js:32-40` accepts any 10–13-digit epoch strictly in
  the future from the provider-controlled `anthropic-ratelimit-unified-reset` header, and
  `quota.js:42-45` converts it to a raw delta. The suite's own diagnostic shows the consequence:
  `real binary no-replacement mapping: ... "retry-after":"212490117"` — roughly 6.7 years. A
  malformed or hostile upstream value can stall OpenCode indefinitely. Clamp to a sane maximum
  (the security baseline's "explicit limits at the boundary").

- **S2 — `install.test.js:11-14` hardcodes `runtimeAssets`** rather than deriving it from the
  files actually shipped in `integrations/opencode/`. The real protection is the staged
  `import()` at line 116, which only catches *statically* imported modules. A future
  lazily-imported or data asset dropped from `install.sh` would still slip through.

- **S3 — the installer harness stops one link short of the capability.** `install.test.js`
  proves the staged bundle loads, but never runs `acm opencode enable` against it. Extending it
  by that one step would have caught W1 automatically.

- **S4 — `cmdLogin` clears the cooldown on a path where no login occurred.** `main.go` now has
  `defer os.Remove(coolFile(t, name))` placed *before* the early `if recoveryExit != 0 { return
  recoveryExit }`, so a profile is marked available again even when `machineLoginState` aborted
  and the interactive login never ran. Previously the removal happened only after the login
  attempt.

- **S5 — coverage signal is thin.** Go statement coverage is 45.1% package-wide with no per-file
  profile; Node has no coverage configuration. Not blocking, but the chain's real assurance comes
  from the real-binary and real-installer layers, not from coverage.

---

## Per-Finding Closure Verdicts (round 4 → round 5)

| Round-4 item | Claimed by | Verdict | Independent evidence |
|---|---|---|---|
| **C1** installer fetches deleted `compatibility.json` | R8 `067a5c6` | ✅ **CLOSED** | My own isolated `install.sh` run stages 7/7 assets and the staged entry point loads; MUT-4 proves the new guard is live |
| **C2** plugin conflict does not stop | R9 `dbd7115` | ✅ **CLOSED** | Live: both-present + `--confirm` → exit 2, bytes preserved, no backup; `--replace-upstream` migrates and rollback restores byte-identical |
| **C3** compatibility spec invalidated by ADR 0001 | R7 `4a1101a` | ✅ **CLOSED** | auth R3 amended; probe shows load with no CLI / `9.9.9` / garbage, `darwin` still refused. *Residual at the README boundary → W2* |
| **W1** cooling with `replacement_available:false` unmapped | R10 `9809e02` | ✅ **CLOSED** | `quota-integration.test.js:126-133` real binary; MUT-1 proves the guard |
| **W2** `oauth.refresh.begin` quarantine uncovered | R10 `9809e02` | ✅ **CLOSED** | `machine-contract.test.js:154-161`, real binary → 401 `acm login` |
| **W3** 503 shape undocumented and unexercised | R10 `9809e02` | ✅ **CLOSED** | `design.md:33` enumerates every 503 class; retryable carry `Retry-After: 1`, asserted at `machine-contract.test.js:194-208` |
| **W6** ENOENT leaks a filesystem path | R11 `533555b` | ✅ **CLOSED** | `index.js:9-11,32-34`; `quota-integration.test.js:43-52` asserts neither temp path nor profile id appears; MUT-2 proves the guard |
| **W7** four machine codes with zero coverage | R10 `9809e02` | ✅ **CLOSED** | `invalid_lease`, `unknown_operation`, `invalid_operation`, `state_busy` all asserted against the real binary |
| **S1** `fsync` misses the directory entry | R10 `9809e02` | ✅ **CLOSED** | `machine.go:640-648`; `TestMachineAtomicWriteSyncsParentAfterRename` asserts order `[file, directory]`; MUT-3 proves the guard |
| **W4** stale generation accepted on replay | *not planned* | ⚠️ **OPEN (accepted)** | Re-proved live; see residual-risk statement below |
| **W5** → escalated to C2 in round 4 | R9 | ✅ **CLOSED** | as C2 |

**Nine of nine remediated findings are genuinely closed**, each verified by evidence I produced
myself rather than by re-reading the implementer's claims.

### W4 residual-risk statement (deliberately unplanned)

W4 was not scheduled for remediation. I am not silently accepting it; here is the explicit
position. A replayed `quota.exhaust` (same `operation_id` + same profile already in the ledger)
returns success even when the submitted generation is stale, because the ledger replay check at
`machine.go:451` runs before the generation check at `machine.go:459`.

**Why the omission is still defensible**: the replay branch returns *before* `saveMachineState`,
so newer state is never overwritten; the response echoes the **current** generation, so the
adapter cannot act on a stale value; and the ledger key already binds the logical operation, so
only a genuine duplicate of the *same* transition can reach it. Non-replayed stale transitions
are still correctly rejected with exit `75` `stale_generation`. It is idempotency, not
corruption.

**Residual risk**: (1) failover R7 S2 cannot be marked fully compliant while the specification
says "MUST reject stale generations" and the code accepts one class of them — the contradiction
will resurface in every future round; (2) the behaviour is undocumented in `design.md`, so a
future refactor may "fix" it and break idempotency, or extend the replay window and turn it into
a real staleness hole. **Recommendation**: document the idempotency exception in `design.md`
§Interfaces and narrow failover R7's wording in the same slice, per the ADR 0001 rule. Cost is
two sentences; leaving it costs a permanent PARTIAL.

## Isolation and Safety Compliance

Every execution used temporary `HOME`, `ACM_DIR`, `ACM_OPENCODE_CONFIG_HOME`,
`ACM_OPENCODE_PLUGIN_PATH`, `ACM_BIN_DIR`, `ACM_SHARE_DIR`, and `TMPDIR` under `mktemp -d`,
with `env -i` and offline fake `curl`/`acm`. No real credential, profile, alias file, or
OpenCode configuration was read, written, or deleted; `~/.config/opencode/` and `~/.acm/` were
never touched, and `$HOME/.local/share/acm/opencode` was confirmed absent after the E2E run.
All sandboxes were removed. All four mutations were restored and checksum-verified byte-identical,
with `git status --porcelain` empty. Nothing was fixed, refactored, committed, pushed, or opened
as a PR. This report is the only file written.

## Required Before This Change Can Merge

1. **C1(R5)** — restore a restorable backup for every config-mutating `enable`, **or** amend
   lifecycle R10/R11 and ship a supported undo path. Then add the missing test: a *successful*
   plain `enable --confirm` on a config without the upstream plugin, asserting the undo contract.
2. **W1** — make `enableOpenCode` honour `ACM_SHARE_DIR`, or remove it from `install.sh`; extend
   `install.test.js` to run `acm opencode enable` against the staged bundle (also closes S3).
3. **W2** — correct `README.md:63` to match ADR 0001 and the amended auth R3.
4. **W3** — restore doctor's profile listing and state line, or add a requirement authorising the
   reduced output.
5. **W5** — document the replay idempotency exception in `design.md` and align failover R7's
   wording in the same slice.

## Final Verdict

**FAIL** — 1 CRITICAL, 6 WARNING, 5 SUGGESTION. **Not safe to merge.**

Say it plainly: the R7–R11 chain closed all nine round-4 findings, and I proved each closure
independently rather than trusting the report. The engineering is real and the new real-binary
and real-installer harnesses are the strongest guards this change has ever had. But the chain
also shipped a fresh instance of its own signature defect — R9's fix removed the backup from the
one enable path no test exercises, so ACM now edits a user's OpenCode configuration and cannot
undo it. A green suite did not see it, and would not have.

*No file was fixed. All mutated files restored byte-identical and checksum-verified. Read-only
apart from this report.*

---

# Appendix: Round 4 (superseded, preserved for history)

> Preserved verbatim from the round-4 report at tip `06b9ab6`. Its YAML envelope is reproduced
> as a plain text block so that this document has exactly one machine-readable envelope.

```text
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

**Change**: `acm-opencode-claude-plugin` · **Round**: 4 · **Tip**: `06b9ab6` on
`feat/opencode-plugin-r6-compat-policy`, 15 commits from `main` · **Verdict**: FAIL — 3 CRITICAL.

## Round-4 Executive Summary

Every mandated command exited `0`. All 24 Node tests and the entire Go suite passed, with and
without `-race`. Round-3 C1 was closed: the production entry point loaded on the installed
runtime. The change nevertheless failed in the same shape as rounds 1–3: a documented state that
no test exercised, while the suite stayed fully green. The defect was introduced by the round-3
fix itself — `06b9ab6` deleted `integrations/opencode/compatibility.json` but left
`install.sh:66` fetching it, so under `set -eu` with `curl -fsSL` the installer aborted on a 404
and never bundled the plugin. A second CRITICAL was a spec scenario whose implementation and test
asserted the opposite of the specification; a third was a specification the accepted ADR silently
invalidated without amendment.

## Round-4 Findings

### CRITICAL

- **C1 — `install.sh:66` fetches the deleted `compatibility.json`; the installer aborts and never
  bundles the plugin.** Introduced by `06b9ab6`. Breaks lifecycle R9 S1. Offline simulation showed
  `curl: (22) ... 404`, `loop_exit=22`, share dir never created; the control with the file removed
  from the list bundled all seven assets. `acm opencode enable` then failed with *"el adaptador
  OpenCode de ACM no está instalado"*. `sh -n install.sh` validates syntax only and `install.sh`
  had no behavioural test at all.
- **C2 — Plugin-conflict detection does not stop; the test asserts the opposite of the spec.**
  `opencode_lifecycle.go:185-195` filtered both entries and appended the ACM URL unconditionally;
  `enableOpenCode` returned `nil` and the command exited `0`. `opencode_lifecycle_test.go:52-56`
  constructed the both-present conflict and asserted `code == 0`. Breaks lifecycle R10 S2.
  (Round-1 W5 escalated.)
- **C3 — The compatibility requirement was invalidated by ADR 0001 but the spec was never
  amended.** `specs/acm-opencode-claude-auth/spec.md:44,52-56` still mandated blocking outside an
  explicit OpenCode/Claude CLI matrix that `06b9ab6` deliberately removed. Breaks auth R3 S2. Fix
  by amending the spec, not the code.

### WARNING

- **W1** — `quota.exhaust` cooling with `replacement_available: false` never mapped end-to-end
  against the real binary.
- **W2** — `oauth.refresh.begin` returning `credential_quarantined` had zero coverage
  (`machine.go:344-347`).
- **W3** — the 503 shape was undocumented and mostly unexercised; four codes collapsed into it and
  `state_busy` / `invalid_lease` carried `retryable:true` at the machine layer yet surfaced with no
  `Retry-After`.
- **W4** — stale generation accepted on a replayed `quota.exhaust` (`machine.go:451` precedes
  `machine.go:459`); deliberate idempotency, non-corrupting, but contradicting failover R7 and
  undocumented.
- **W5** — escalated to C2.
- **W6** — ENOENT leaked a filesystem path out of `auth.fetch` (`index.js:25,54-57`), disclosing
  the ACM directory layout and profile name into OpenCode's error surface.
- **W7** — four machine error codes had zero coverage anywhere: `unknown_operation`,
  `state_busy`, `invalid_lease`, and the dispatcher `default:` `invalid_operation`.

### SUGGESTION

- **S1** — `atomicWriteMachineFile` synced the temp file but never `filepath.Dir(path)` after
  `os.Rename`, so a crash could lose the rename.
- **S2** — `compat.test.js:18-21` was self-referential (read `package.json`, asserted
  `package.json`).
- **S3** — `ACM diagnostics: record_failed` on stderr assessed as acceptable best-effort behavior,
  not a defect.
- **S4** — `output_too_large` (`machine.go:111-115`) was dead code; a full 256-event ring truncates
  to 64 events and 6,658 bytes, far under the 16 KiB cap.
- **S5** — `quota.test.js` stubs omitted `replacement_available`, so the `Retry-After` expectation
  rested on `undefined !== true`.
- **S6** — `validMachineRequest` was lax for read operations (`machine.go:159`).

## Round-4 Verdict

**FAIL** — 3 CRITICAL, 6 WARNING, 6 SUGGESTION. Round-3 C1 genuinely closed and the round-1/2
defects held under mutation, but the fix that closed C1 opened C1' at the installer boundary, and
two specification scenarios remained contradicted by the code and by their own tests.
