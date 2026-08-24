# Tasks: ACM OpenCode Claude Plugin

## Review Workload Forecast
| Field | Value |
|---|---|
| Estimated changed lines | 1,050–1,350 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 → PR 5 |
| Delivery / chain | auto-chain / feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Work Units
| Unit | Goal / dep / budget | Focused test | Runtime harness | Rollback |
|---|---|---|---|---|
| 1 | main → v1 state; ≤300 | `go test -run TestMachine .` | `go test -run TestMachineCLIProcess .` | machine/routing/tests |
| 2 | leases → refresh; after 1; ≤320 | `go test -run TestOAuthRefresh .` | `go test -run TestMachineRefreshLeaseProcess .` | lease/commit/tests |
| 3 | plugin → adapter; after 2; ≤360 | `node --test integrations/opencode/test/compat.test.js` | `node --test integrations/opencode/test/machine-process.test.js` | modules/fixtures |
| 4 | quota/diagnostics; after 3; ≤350 | `node --test integrations/opencode/test/quota.test.js` | `node --test integrations/opencode/test/quota-integration.test.js` | quota/diagnostics |
| 5 | bundle → migration/docs; after 4; ≤350 | `go test -run TestOpenCodeMigration .` | `ACM_OPENCODE_CONFIG_HOME=$(mktemp -d) go test -run TestOpenCodeMigration .`; temporary config home, never live | lifecycle/bundle/docs |

## Phase 1: Protocol/ledger (PR 1)
- [x] 1.1 **RED**: add `machine_test.go` v1/unknown-field, secretless deterministic select, canonical paths, stale hash-ledger (once/profile; 24h/1024), and CLI bounded 64KiB stdin/16KiB stdout cases.
- [x] 1.2 **GREEN**: create `machine.go`; replace `main.go:runMachine` with bounded v1 stdin/stdout selection/status, canonical paths, ledger, and exit taxonomy.
- [x] 1.3 **REFACTOR**: remove spike `runMachine` lock/response tests from `main_test.go`; retain command routing.

## Phase 2: OAuth lease/commit (PR 2)
- [x] 2.1 **RED**: add `machine_test.go` busy/expired lease, kill-before-commit, stale generation, write/fsync failure, 0600 atomicity, no-secret cases; quarantine only invalid_grant/revoked/unrecoverable.
- [x] 2.2 **GREEN**: implement `oauth.refresh.begin|commit|abort`: stdin-only secrets, leases, generation/fsync commit; stale/contention/expiry/I/O fail closed and preserve/reload state.
- [x] 2.3 **REFACTOR**: consolidate helpers; preserve `go test -race ./...` concurrency coverage.

## Phase 3: Plugin (PR 3)
- [x] 3.1 **RED**: add `integrations/opencode/test/machine-process.test.js` `execFile` cases: fixed operation; timeout; malformed/oversized stdout; nonempty stderr; unexpected exit/schema reject safely; add Linux/ACM, matrix, hash, transform, refresh, no-auth-write, OpenCode-owned retry/session and no replay/queue/stream/agent-continuation fixtures.
- [x] 3.2 **GREEN**: create `index,machine,oauth,compat.js`, `compatibility.json`, and Node script for OpenCode 1.18.19/SDK 1.17.12/Claude CLI 2.1.236.
- [x] 3.3 **REFACTOR**: delete `acm-anthropic-plugin.js`, `lib/failover.js`, and Bun replay tests; retain synthetic-only fixtures.

## Phase 4: Quota/diagnostics (PR 4)
- [x] 4.1 **RED**: add Node confirmed/stale transition, generic 401/429/529 passthrough, cooling-versus-quarantine metadata, and bounded redaction fixtures.
- [x] 4.2 **GREEN**: create `quota.js`/`diagnostics.js`; quarantine revoked credentials, return retry metadata only, and leave waiting/retry to OpenCode.
- [x] 4.3 **REFACTOR**: enforce one provider call/attempt; remove failover fixture assumptions.

## Phase 4a: Quota backend (PR 4a)
- [x] 4a.1 **RED**: add `machine_test.go` cases for confirmed exhaustion, explicit and fallback cooldowns, stale or unknown profiles, generation and ledger interaction, idempotent operation IDs, secret rejection, and the CLI process boundary.
- [x] 4a.2 **GREEN**: implement strict, generation-aware, atomically persisted `quota.exhaust` handling through the v1 machine dispatcher.
- [x] 4a.3 **REFACTOR**: consolidate shared helpers without weakening `go test -race ./...` coverage.

## Phase 5: Lifecycle (PR 5)
- [x] 5.1 **RED**: add `opencode_lifecycle_test.go` `TestOpenCodeMigrationRollbackOnJSONCConflict`: JSON/JSONC conflict/ambiguity restores all; cover opt-in/exclusivity/checksums/invalid-missing backup/rollback.
- [x] 5.2 **GREEN**: implement `opencode_lifecycle.go`, bundle in `install.sh`, and atomically migrate/rollback temporary configs with confirmation, validation, restart guidance.
- [x] 5.3 **REFACTOR**: update `README.md` disabled-by-default enable/rollback; run Go/race/Node tests with synthetic state only.

## Remediation Review Workload Forecast
| Field | Value |
|---|---|
| Estimated changed lines | 1,100–1,260 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | R1 → R2 → R3 → R4 |
| Delivery / chain | auto-chain / feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Remediation Work Units
| Unit | Goal / dependency / budget | Focused test | Runtime harness | Rollback |
|---|---|---|---|---|
| R1 | Binary contract + machine outcomes; starts at PR5; ≤350 | `node --test integrations/opencode/test/machine-contract.test.js` | Same command builds `acm` with temp `HOME`/`ACM_DIR` and invokes each machine operation | `machine.go`, `machine_test.go`, contract test |
| R2 | Adapter maps real machine outcomes; after R1; ≤310 | `node --test integrations/opencode/test/{machine-contract,quota-integration,quota}.test.js` | Compiled binary fixture: cooling, quarantined, refresh→quota | `index.js`, `oauth.js`, `quota.js`, tests |
| R3 | Live compatibility evidence; after R2; ≤220 | `node --test integrations/opencode/test/compat.test.js` | `node integrations/opencode/scripts/check-compat.js` with controlled runtime-version fixture | `compat.js`, `index.js`, checker, tests |
| R4 | Persistent diagnostics + login recovery; after R3; ≤380 | `go test -run 'TestMachineDiagnostics|TestLogin' . && node --test integrations/opencode/test/{machine-contract,quota}.test.js` | Temp-state compiled `acm doctor` after refresh failure/login | `machine.go`, `main.go`, diagnostics modules, tests |
| R5 | Replacement-aware quota transition; after R4; ≤220 | `node --test integrations/opencode/test/quota-integration.test.js` | Compiled binary: exhaust a profile while a healthy replacement remains selectable, assert no `Retry-After` | `machine.go` quota response, `quota.js` header mapping, tests |
| R6 | Retroactive ecosystem compatibility policy; after R5; 193/200 | `node --test integrations/opencode/test/compat.test.js` | Missing: no installer harness proved bundle reachability; R8 must create it | R6 compatibility files/tests and ADR 0001 |
| R7 | ADR/spec coherence; after R6; ≤90 | `node --test integrations/opencode/test/contract-coherence.test.js` | Isolated Node factory loads with no CLI and `9.9.9` CLI | R3 amendment/test |
| R8 | Distribution reaches user; after R7; ≤260 | `node --test integrations/opencode/test/install.test.js` | **Creates missing harness:** actual `sh install.sh` in temp HOME/BIN/SHARE with fake curl/acm; no host writes | installer/assets/test |
| R9 | Guided conflict resolution; after R8; ≤220 | `go test -run TestOpenCodeMigration .` | Compiled `acm` against temp config/plugin: conflict stops, `--replace-upstream` succeeds | lifecycle/docs/tests |
| R10 | Machine outcomes and durable retry; after R9; ≤390 | `go test -run TestMachine . && node --test integrations/opencode/test/{machine-contract,quota-integration}.test.js` | Temp-state compiled binary induces each outcome and adapter response | machine/adapter/design/tests |
| R11 | Secretless local auth errors; after R10; ≤130 | `node --test integrations/opencode/test/quota-integration.test.js` | Factory fetch with missing temp credential path returns fixed safe error | index/test |

## Phase 6: Contract Guard and Machine Outcomes (R1; CRITICAL 1–3)
- [x] 6.1 **RED**: add `integrations/opencode/test/machine-contract.test.js`; spawn a real compiled `acm` for `credential.select`, refresh begin/commit/abort, and `quota.exhaust`, asserting exact response keys, secretlessness, cooling reset, and unavailable variants.
- [x] 6.2 **GREEN**: update `machine.go` and `machine_test.go` so cooling returns the earliest `reset_at`, all-quarantined returns actionable non-retryable `acm login`, and mixed state derives reset only from cooling profiles.
- [x] 6.3 **REFACTOR**: centralize `machine.go` unavailable/outcome response construction; retain deterministic JSON and the strict protocol boundary.

## Phase 7: Adapter Transition Semantics (R2; CRITICAL 1–4)
- [x] 7.1 **RED**: extend `machine-contract.test.js` and `quota-integration.test.js` with real-binary cooling `Retry-After`, quarantined 401, mixed state, and refresh-then-quota rotation scenarios; remove fictional response stubs.
- [x] 7.2 **GREEN**: update `integrations/opencode/{machine,index,oauth,quota}.js` to preserve safe machine failure metadata, translate `reset_at` to `Retry-After`, map unavailable outcomes, and use the refresh-commit generation for `quota.exhaust`.
- [x] 7.3 **REFACTOR**: consolidate response mapping in `quota.js`; verify no plugin wait, replay, stream continuation, or raw machine error escapes.

## Phase 8: Compatibility Evidence (R3; CRITICAL 5)
- [x] 8.1 **RED**: add `integrations/opencode/test/compat.test.js` cases proving production rejects missing or mismatched observed OpenCode, SDK, or Claude CLI versions without an injected matrix echo.
- [x] 8.2 **GREEN**: update `integrations/opencode/{compat,index}.js` and `scripts/check-compat.js` to resolve installed runtime versions independently of `compatibility.json` and fail closed outside the matrix.
- [x] 8.3 **REFACTOR**: share one version-resolution path between plugin and checker; retain only OS/process boundaries as controlled test doubles.

## Phase 9: Diagnostics and Reauthentication Recovery (R4; CRITICAL 6 + escalated risk)
- [x] 9.1a **RED (R4a)**: add `machine_test.go`, `main_test.go`, and Node cases for a 0600 bounded redacted diagnostic ring, `acm doctor` aggregates/lease health, production composition, and real-binary diagnostic recording.
- [x] 9.1b **RED (R4b)**: add cases for successful `acm login` clearing only its profile quarantine.
- [x] 9.2a **GREEN (R4a)**: add production diagnostic recording/status in `machine.go`, surface aggregates from `cmdDoctor`, and provide a best-effort observable sink from `integrations/opencode/index.js`.
- [x] 9.2b **GREEN (R4b)**: implement generation-safe unquarantine after successful Claude login.
- [x] 9.3a **REFACTOR (R4a)**: centralize diagnostic redaction/state mutation and prove tokens, bodies, paths, environment values, and identifiers are absent from persisted diagnostics and doctor output.
- [x] 9.3b **REFACTOR (R4b)**: centralize login quarantine recovery without changing diagnostics or failover response contracts.

## Phase 10: Replacement-Aware Quota Transition (R5; round-2 CRITICAL)

Round-2 verification found that `machineQuotaResponse` always reports `outcome: "cooling"` with a `reset_at`, and `quota.js` always maps that to a `Retry-After`, without either side asking whether another profile is still selectable. Exhausting `alpha` while a healthy `beta` is available therefore returns `429 Retry-After: 3600`, so OpenCode idles for an hour with a usable account standing by. `design.md` already specifies the missing case: "Replacement available: plugin returns bounded 429 without provider `Retry-After`, prompting SessionRetry." The suite stayed green because no test exercised exhaustion *with* a replacement available.

- [x] 10.1 **RED**: add a `quota-integration.test.js` case that exhausts a profile against the real binary while a healthy replacement remains selectable, and asserts the response carries NO `Retry-After`; extend `machine-contract.test.js` for the new replacement-aware `quota.exhaust` response, keeping every existing assertion and the `retry_after` negative control intact.
- [x] 10.2 **GREEN**: make `quota.exhaust` report whether a replacement is still selectable after the transition, and make `quota.js` suppress `Retry-After` in that case while preserving the all-cooling earliest-reset behavior and the all-quarantined non-retryable `401`.
- [x] 10.3 **REFACTOR**: express all four `design.md` outcomes through one mapping path, and strengthen the contract guard's negative control so a bogus field name inside `assert.throws` can no longer pass undetected.

## Phase 11: Ecosystem Compatibility Convention (R6; round-3 CRITICAL)
- [x] 11.1 **RED**: replace exact-matrix tests with plugin API range, non-exact runtime load, retained precondition, and best-effort CLI diagnostic expectations.
- [x] 11.2 **GREEN**: declare `@opencode-ai/plugin: ^1.18.18`, remove the matrix/resolver/checker, and record CLI detection without gating load.
- [x] 11.3 **REFACTOR**: remove obsolete fixtures/scripts, preserve credential validation and frozen boundaries, and record ADR 0001.

## Round-4 Review Workload Forecast
| Field | Value |
|---|---|
| Estimated changed lines | 780–960 across five PRs |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery / chain | auto-chain / feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

Scope rule: slice by end-user capability and rollback boundary, including every producer and consumer; never exclude a consumer merely because it is another file.

## Phase 12: Contract Coherence (R7; C3)
- [x] 12.1 **RED**: add `contract-coherence.test.js` that fails while auth R3 requires a removed version matrix, but proves package-range load with missing/non-exact CLI evidence.
- [x] 12.2 **GREEN**: amend `specs/acm-opencode-claude-auth/spec.md` R3 to match ADR 0001: retain quarantine and hard platform/profile/credential gates; make CLI detection diagnostic-only.
- [x] 12.3 **REFACTOR**: keep the ADR and amended R3 in this decision-coherence slice; every future superseding compatibility decision MUST amend its spec in the same slice.

## Phase 13: Distribution Integrity (R8; C1)
- [x] 13.1 **RED**: add `install.test.js` offline fixture; it must fail on the stale `compatibility.json` request and assert the complete staged runtime asset set.
- [x] 13.2 **GREEN**: update `install.sh` to fetch only shipped adapter assets and stage them atomically in the fixture’s `ACM_SHARE_DIR`.
- [x] 13.3 **REFACTOR**: make fixture cleanup assert no host paths, aliases, credentials, or real installer targets were touched.

## Phase 14: Guided Migration (R9; C2)
- [x] 14.1 **RED**: correct `opencode_lifecycle_test.go`: both plugins plus `--confirm` exits nonzero, preserves bytes, and creates no backup; `--replace-upstream` is required.
- [x] 14.2 **GREEN**: update `opencode_lifecycle.go` and `README.md`: only `enable --confirm --replace-upstream` may migrate and back up JSON/JSONC.
- [x] 14.3 **REFACTOR**: share plugin-conflict detection without weakening checksum, ambiguity, or rollback guards.

## Phase 15: Outcome and Durability Boundary (R10; W1/W2/W3/W7/S1)
- [x] 15.1 **RED**: extend real-binary guards for cooling/no replacement, begin-quarantine, busy, invalid lease, unknown operation, dispatcher invalid operation, and directory-sync-after-rename.
- [x] 15.2 **GREEN**: fsync the parent directory; map retryable busy/invalid-lease 503s with bounded retry signal, preserve non-retryable codes, and document all 503 outcomes in `design.md`.
- [x] 15.3 **REFACTOR**: centralize outcome mapping; model `replacement_available:false` explicitly in fixtures and retain mutation guards.

## Phase 16: Local Auth Error Containment (R11; W6)
- [x] 16.1 **RED**: add missing-credential and valid-control cases in `quota-integration.test.js`; assert `auth.fetch` exposes neither temp path nor credential identifier.
- [x] 16.2 **GREEN**: normalize local credential read/parse failures in `integrations/opencode/index.js` to a fixed safe error while retaining machine-response mapping.
- [x] 16.3 **REFACTOR**: isolate the safe-error helper and rerun the focused factory harness.

## Round-5 Remediation Review Workload Forecast
| Field | Value |
|---|---|
| Estimated changed lines | 600–810 across five slices |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | R12 → R13 → R14 → R15 → R16 |
| Delivery / chain | auto-chain / feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Round-5 Remediation Work Units
| Slice | Capability | Dependency | Line budget | Verification command | Real-binary / offline proof | Files touched |
|---|---|---|---:|---|---|---|
| R12 | Plain opt-in is always restorable | R11 | 150–190 | `go test -count=1 -run '^TestOpenCodeMigration(PlainEnableCreatesRestorableBackup|RollbackAndMissingBackup)$' .` | Real compiled `acm`: isolated plain enable then rollback restores bytes | `opencode_lifecycle.go`, `opencode_lifecycle_test.go`, `README.md` |
| R13 | Custom installer share remains enable-able | R12 | 230–300 | `node --test integrations/opencode/test/install.test.js` | Offline `sh install.sh` + fake curl + built `acm`, custom `ACM_SHARE_DIR`, enable, staged ESM load | `install.test.js`, `opencode_lifecycle.go`, `specs/acm-opencode-plugin-lifecycle/spec.md`, `README.md` |
| R14 | Compatibility guidance matches runtime policy | R13 | 45–70 | `node --test integrations/opencode/test/contract-coherence.test.js` | Offline factory loads with absent/non-exact CLI evidence | `README.md`, `contract-coherence.test.js` |
| R15 | Doctor preserves legacy operational visibility | R14 | 110–150 | `go test -count=1 -run '^TestDoctor' .` | Built `acm doctor` with isolated HOME/ACM_DIR and fake tool binaries shows state plus profiles | `main.go`, `main_test.go` |
| R16 | Replay idempotency contract is explicit | R15 | 65–100 | `node --test integrations/opencode/test/{contract-coherence,machine-contract}.test.js` | Real compiled binary accepts only matching ledger replay, returns current generation, and writes no state | `specs/acm-opencode-claude-failover/spec.md`, `design.md`, `test/{contract-coherence,machine-contract}.test.js` |

Feature-branch chain: R12 base = tracker branch; R13 base = R12 branch; R14 base = R13 branch; R15 base = R14 branch; R16 base = R15 branch. Retarget/rebase any polluted child diff.

## Phase 17: Restorable Plain Opt-In (R12; C1(R5))
- [x] 17.1 **RED**: in `opencode_lifecycle_test.go`, add no-upstream plain-enable tests that fail without manifest/backup, then prove real compiled `acm` enable→rollback restores original bytes under temp HOME, ACM_DIR, config, bin, and plugin paths.
- [x] 17.2 **GREEN**: make `opencode_lifecycle.go` create the checksummed backup and manifest before every config-mutating enable; update `README.md` so plain opt-in documents rollback availability.
- [x] 17.3 **REFACTOR**: retain one backup transaction for plain and replacement enable; rerun focused, real-binary, Go, Node, formatting, vet, and diff checks.

## Phase 18: Custom-Share Installation Capability (R13; W1)
- [x] 18.1 **RED**: extend `install.test.js` with an offline custom-`ACM_SHARE_DIR` install that builds the real binary, runs enable against a temp config, and fails while lifecycle discovery uses the default share.
- [x] 18.2 **GREEN**: amend `specs/acm-opencode-plugin-lifecycle/spec.md` for the shared override; make `opencode_lifecycle.go` resolve `ACM_SHARE_DIR` before its default plugin path and document it in `README.md`.
- [x] 18.3 **REFACTOR**: keep installer, staged ESM load, enable, rollback, and host-canary assertions in one isolated harness; remove all sandbox trees.

## Phase 19: Compatibility Documentation Coherence (R14; W2)
- [x] 19.1 **RED**: extend `contract-coherence.test.js` to fail while `README.md` names the removed matrix and to retain missing/non-exact CLI factory-load evidence.
- [x] 19.2 **GREEN**: replace the matrix claim in `README.md` with ADR 0001's package-range and diagnostic-only CLI policy.
- [x] 19.3 **REFACTOR**: centralize the README/ADR/spec assertions; rerun the offline factory harness and full Node suite.

## Phase 20: Doctor Legacy Visibility (R15; W3)
- [x] 20.1 **RED**: add `main_test.go` cases that fail until `cmdDoctor` prints `estado : <ACM_DIR>` and delegates to `cmdLs`, including unavailable diagnostics and redaction controls.
- [x] 20.2 **GREEN**: restore the state line and profile listing in `main.go` without removing diagnostic aggregates or leaking lease/profile identifiers.
- [x] 20.3 **REFACTOR**: prove the compiled `acm doctor` output with temporary HOME, ACM_DIR, bin dirs, and fake tool binaries; rerun focused and full Go checks.

## Phase 21: Replay-Exception Contract (R16; W5)
- [ ] 21.1 **RED**: make `contract-coherence.test.js` fail until R7 S2 and `design.md` state the same replay exception; add a baseline real-binary replay trace without inventing a failing behavior test.
- [ ] 21.2 **GREEN**: narrow failover R7 S2 to reject stale non-replay transitions while allowing only same-operation/profile ledger replays that return current generation before persistence; document this in `design.md`.
- [ ] 21.3 **REFACTOR**: align terms across spec/design/tests and rerun the offline contract plus real-binary no-write replay proof.
