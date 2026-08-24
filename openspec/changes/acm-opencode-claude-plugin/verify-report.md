```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:c3f5810c315c5a6dd9f69bb5f8fce41ceba7e83dfe2d3c04ad9f1b73f3824a46
verdict: pass
blockers: 0
critical_findings: 0
requirements: 11/11
scenarios: 23/23
test_command: node --test "integrations/opencode/test/*.test.js" && go test ./...
test_exit_code: 0
test_output_hash: sha256:321843c6481f995c7645a2bb9252d7d4d1b4b3b6f465a46ecaa39a78b0e962f4
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verification Report

- **Change**: `acm-opencode-claude-plugin`
- **Round**: 6
- **Previous report**: round 5 at commit `e00968b`
- **Evidence tip**: `5b27806` on `feat/opencode-plugin-r16-replay-contract`
- **Mode**: Strict TDD, full spec-driven verification
- **Artifact store**: hybrid (OpenSpec + Engram, project `acm`)
- **Verdict**: **PASS**
- **Archive readiness**: Ready from the verification scope

## Executive Summary

All 11 current requirements and all 23 current scenarios are compliant. The authoritative scenario count is 23: 7 authentication, 8 failover, and 8 lifecycle scenarios. This corrects the stale 21-scenario total in the round-5 report.

The exact full Node command passed 29/29 tests, `go test ./...` passed, and `go build ./...` passed. The round-5 CRITICAL is independently closed: plain `acm opencode enable --confirm` creates a checksummed backup and manifest, and real-binary rollback restores 54 bytes byte-for-byte with SHA-256 `1699ac0e251ff28ea53a07ad47734c28369f8512cc2d31a49571f53f1d08abcc`. The offline installer harness separately installed under a custom share, enabled the staged entry point, loaded both hooks, rolled back 29 exact bytes, rejected a missing custom share without fallback, and left host canaries unchanged.

Three high-risk regression guards were mutation-tested. Each focused suite turned red for the intended reason, each mutation was reverted with `git checkout -- <file>`, and restored source hashes match their baselines. Final worktree state contains only the accepted untracked `.pi/` directory.

## Evidence Scope

Verified chain:

`4a1101a` (R7) → `067a5c6` (R8) → `dbd7115` (R9) → `9809e02` (R10) → `533555b` (R11) → `c47970a` (R12) → `8f9c0ed` (R13) → `94cfd87` (R14) → `2ce929e` (R15) → `2ca9181` (R16) → `5b27806` (test-only flake fix, 5 changed lines).

`e00968b` is the intervening round-5 report and remediation-plan commit. The source evidence revision is the SHA-256 of the newline-terminated evidence-tip commit ID.

## Completeness

| Dimension | Result | Evidence |
|---|---:|---|
| Tasks | 69/69 complete | `tasks.md`; 69 checked, 0 unchecked |
| Proposal | Present | `proposal.md` |
| Specifications | 3 present | auth, failover, lifecycle |
| Design | Present | `design.md` |
| Requirements | 11/11 | 4 auth + 4 failover + 3 lifecycle |
| Scenarios | 23/23 | 7 auth + 8 failover + 8 lifecycle |

## Build and Test Execution

Environment: Node `v26.7.0`; Go `go1.26.4 linux/amd64`.

| Command | Exit | Result | Output hash |
|---|---:|---|---|
| `node --test "integrations/opencode/test/*.test.js" && go test ./...` | 0 | Node: 29 passed, 0 failed/skipped; Go: `ok`, 38.828s | `sha256:321843c6481f995c7645a2bb9252d7d4d1b4b3b6f465a46ecaa39a78b0e962f4` |
| `go build ./...` | 0 | Passed; no output | `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `go test -count=1 -v -run '^TestOpenCodeMigrationPlainEnableCreatesRestorableBackup$' .` | 0 | In-process and real compiled-binary enable→rollback passed; 54 exact bytes restored | `sha256:036a7d5e03c06bbb9df118a6212245a344eee6025a10bc7f4037b13a43de8c27` |
| `node --test integrations/opencode/test/install.test.js` | 0 | 1/1 offline installer E2E passed; install/enable/load/rollback/custom-share isolation proved | `sha256:77bbbdfcfb2c460ecbbf9d14fd6f2ec137c9fc14aa25dcb75b51973a10fd639f` |
| `go test -cover ./...` | 0 | 48.4% package statement coverage; no threshold configured | `sha256:f43cfd6b66e275359abce89045ec13480aa4df958d23e2a44f7e2b5d9b0c1a41` |
| `gofmt -l . && go vet ./... && git diff --check` | 0 | Clean; no output | `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |

`sh -n install.sh` was neither run nor cited as installer evidence. Installer compliance rests exclusively on the offline behavioral harness in `integrations/opencode/test/install.test.js`.

## Round-5 Finding Closure

| Round-5 item | Closure | Independent round-6 evidence | Result |
|---|---|---|---|
| CRITICAL: plain enable had no rollback backup | R12 `c47970a` | Focused real-binary `TestOpenCodeMigrationPlainEnableCreatesRestorableBackup`; offline install enable→rollback; backup mutation turned the focused test red | ✅ CLOSED |
| W1: custom share could install but not enable | R13 `8f9c0ed` | `install.test.js` uses custom `ACM_SHARE_DIR`, enables its exact staged URL, imports it, and rejects missing custom entry without default fallback | ✅ CLOSED |
| W2: compatibility guidance contradicted runtime policy | R14 `94cfd87` | `contract-coherence.test.js`; package-range mutation failed in `assertCompatibilityPolicy` | ✅ CLOSED |
| W3: doctor lost state/profile visibility | R15 `2ce929e` | `TestDoctorRestoresStateAndRedactedProfileVisibility` passes for available and unavailable diagnostics | ✅ CLOSED |
| W5: replay exception contradicted spec/design | R16 `2ca9181` | Coherence test plus real-binary replay/non-replay trace; current-generation mutation turned the contract red | ✅ CLOSED |
| W6/W7: local-auth containment and uncovered machine outcomes | R10 `9809e02`, R11 `533555b` | `machine-contract.test.js`, `quota-integration.test.js`, and full suites pass | ✅ CLOSED |

The maintainer's final decision to retain doctor profile-name redaction is respected. `unknown disponible` rows preserve operational visibility without exposing raw profile names and are not a finding.

## Spec Compliance Matrix

| Requirement | Scenario | Runtime evidence | Result |
|---|---|---|---|
| Auth R1 — Ephemeral authentication | Selected profile supplies authentication | `machine-contract.test.js` real `credential.select`; `quota-integration.test.js` valid credential/provider control | ✅ COMPLIANT |
| Auth R1 — Ephemeral authentication | Non-ACM or unsupported host is requested | `compat.test.js` — `keeps Linux and ACM profile state as hard preconditions` | ✅ COMPLIANT |
| Auth R2 — Controlled refresh | Normal expiry refresh succeeds | `compat.test.js` begin/commit test; `quota-integration.test.js` real-binary refresh generation 2→3 | ✅ COMPLIANT |
| Auth R2 — Controlled refresh | Stale or failed refresh commit | `TestOAuthRefreshCommitFailsClosedOnStaleWriteAndFsync` | ✅ COMPLIANT |
| Auth R3 — Quarantine/compatibility | Refresh credentials are revoked | `quota.test.js` unrecoverable refresh; `TestOAuthRefreshAbortQuarantinesOnlyTerminalReasons` | ✅ COMPLIANT |
| Auth R3 — Quarantine/compatibility | Claude CLI evidence is diagnostic only | `compat.test.js` non-exact/missing CLI cases; `contract-coherence.test.js` | ✅ COMPLIANT |
| Auth R4 — Redacted diagnostics | Doctor collects a failed refresh event | `TestDoctorRestoresStateAndRedactedProfileVisibility`; `TestMachineDiagnosticsAreBoundedRedactedAnd0600` | ✅ COMPLIANT |
| Failover R1 — Evidence-based transition | Confirmed quota exhaustion selects another profile | `quota-integration.test.js` real binary: one provider call, alpha exhausted, beta selected | ✅ COMPLIANT |
| Failover R1 — Evidence-based transition | Generic rate-limit-like response is received | `quota.test.js` and `quota-integration.test.js`: 401/429/529 object passthrough | ✅ COMPLIANT |
| Failover R2 — OpenCode-owned retry | OpenCode retries after transition | `compat.test.js` one-attempt/no-replay hooks; `quota-integration.test.js` exactly one provider call | ✅ COMPLIANT |
| Failover R3 — Bounded attempts | Multiple OpenCode retries consume candidates | `TestMachineLedgerIsOncePerProfileStaleAndBounded`; real-binary alpha→beta selection | ✅ COMPLIANT |
| Failover R3 — Bounded attempts | Concurrent stale transition arrives | `machine-contract.test.js`: matching replay returns current generation/no write; stale non-replay exits 75/no write | ✅ COMPLIANT |
| Failover R4 — No eligible profile | Cooling profile supplies retry metadata | `machine-contract.test.js` and `quota-integration.test.js`: earliest reset → `Retry-After` | ✅ COMPLIANT |
| Failover R4 — No eligible profile | Only quarantined profiles remain | Same real-binary tests: 401, `acm login`, no retry header | ✅ COMPLIANT |
| Failover R4 — No eligible profile | Cooling and quarantined profiles are mixed | Same real-binary tests: reset derives only from cooling profile | ✅ COMPLIANT |
| Lifecycle R1 — Bundled opt-in | Fresh ACM installation | `install.test.js` runs real installer before explicit enable, stages seven assets disabled, and preserves host state | ✅ COMPLIANT |
| Lifecycle R1 — Bundled opt-in | User explicitly enables the experiment | Focused real-binary plain-enable test and offline installer E2E | ✅ COMPLIANT |
| Lifecycle R1 — Bundled opt-in | Custom share installation remains enable-able | `install.test.js`: custom-share URL configured and staged ESM hooks loaded | ✅ COMPLIANT |
| Lifecycle R1 — Bundled opt-in | Configured custom share entry point is missing | `install.test.js`: exit 2, no default fallback, config/backup state unchanged | ✅ COMPLIANT |
| Lifecycle R2 — Guided migration | Confirmed migration from upstream plugin | `TestOpenCodeMigrationRealBinaryRequiresExplicitReplacement` | ✅ COMPLIANT |
| Lifecycle R2 — Guided migration | Plugin conflict is detected | `TestOpenCodeMigrationEnforcesExclusivityAndChecksum` and real-binary replacement test | ✅ COMPLIANT |
| Lifecycle R3 — Rollback | Rollback after experimental use | Plain-enable real binary restores 54 bytes; installer E2E restores 29 bytes | ✅ COMPLIANT |
| Lifecycle R3 — Rollback | Backup is missing or invalid | `TestOpenCodeMigrationRollbackAndMissingBackup`; checksum failure case | ✅ COMPLIANT |

**Compliance summary**: 23/23 scenarios compliant.

## Correctness (Static Evidence)

| Requirement | Status | Evidence |
|---|---|---|
| ACM-managed ephemeral auth | ✅ Implemented | `index.js` selects through the machine boundary, removes the internal header, and sends one provider request |
| Controlled refresh commit | ✅ Implemented | `oauth.js` uses begin/commit/abort; Go validates lease/profile/generation and persists atomically |
| Quarantine and compatibility | ✅ Implemented | terminal refresh reasons quarantine; package range governs API compatibility; CLI is diagnostic-only |
| Redacted diagnostics | ✅ Implemented | bounded allowlists, 0600 state, aggregate doctor output, redacted `unknown` profile rows |
| Evidence-based quota transition | ✅ Implemented | 429 + typed error + rejected unified-status gate; generic statuses pass through |
| OpenCode-owned retry | ✅ Implemented | only `auth` and `chat.headers` hooks; no wait/replay/continuation implementation |
| Bounded operation attempts | ✅ Implemented | 24h/1024 ledger, once-per-profile selection, serialized transition, narrow replay exception |
| No eligible profile outcome | ✅ Implemented | replacement, cooling, quarantine, and mixed outcomes remain distinct |
| Bundled experimental opt-in | ✅ Implemented | installer stages seven assets disabled; lifecycle requires explicit confirmation and honors custom share |
| Guided exclusive migration | ✅ Implemented | upstream conflict stops unless explicit replacement; backup precedes config mutation |
| Configuration-only rollback | ✅ Implemented | checksum validation restores config bytes and removes backup artifacts without ACM account-state migration |

## Design Coherence

| Decision | Result | Evidence |
|---|---|---|
| Versioned secretless stdin/stdout boundary | ✅ Followed | strict operation/schema/size validation; process tests cover malformed, oversized, stderr, timeout, and exits |
| Lease-spanning refresh with generation-checked commit | ✅ Followed | Go and Node refresh tests cover success, stale, expiry, persistence, and fsync failure |
| Stable operation hash and OpenCode recovery ownership | ✅ Followed | operation hash fixture and one-provider-call/no-replay tests |
| Replay exception returns current generation before persistence | ✅ Followed | real-binary inode/byte no-write trace and mutation check |
| Compatibility uses declared package range; CLI diagnostic-only | ✅ Followed | package/spec/ADR/README coherence guard plus missing/non-exact runtime cases |
| Lifecycle backup, custom share, and rollback transaction | ✅ Followed | real binary and offline installer harnesses |
| Redacted diagnostics and doctor aggregates | ✅ Followed | bounded diagnostic tests and final maintainer-approved profile redaction |

No design deviation breaks or weakens a specification.

## Guard Audit by Mutation

| Guard | Mutation | Expected/observed | Output hash | Result |
|---|---|---|---|---|
| Plain-enable backup is unconditional | Guard backup writes with `replaceUpstream` in `opencode_lifecycle.go` | Focused test exit 1: `.acm-backup` missing | `sha256:c8ab5b5d7df521b3c1c27a6e82b2159d86660459960ca35c642bcb8ac8630e92` | ✅ LIVE |
| Compatibility policy is coherent | Change package range `^1.18.18` → `^1.18.17` | Coherence test exit 1 at `assertCompatibilityPolicy` | `sha256:c4921ef29b9d54a73c3f6859f4b88fd4b31de8a81612232b080c5086a0326473` | ✅ LIVE |
| Ledger replay returns current generation | Force replay response to return submitted stale generation | Machine contract exit 1: `2 !== 3` | `sha256:1e5c1cf65ff3523c99834d0eecf29e3639143b3862a25347d91bf30a374fb60d` | ✅ LIVE |

Restored source hashes:

- `opencode_lifecycle.go`: `feecc7c7a0858e26a9d27a603f259a328d15da0e84c4787a75c16e25fd5c26ea`
- `integrations/opencode/package.json`: `0b3ae8a9b9f9ccad46bd924df4b02577bbf5ec49beb125a0916112c6cc2bdf2e`
- `machine.go`: `5171b7734e71666d917d008f5dc1b9083d65e9565bf6d7fded466e84e8965a66`

## Strict TDD Verification

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | `apply-progress.md` contains RED/GREEN/triangulation/safety-net evidence through R16 |
| All tasks complete and test-mapped | ✅ | 69/69 task checkboxes complete; each behavior work unit names its test boundary |
| RED evidence credible | ✅ | Test files exist; no artificial behavioral RED was invented for R16's documentation-only correction |
| GREEN confirmed now | ✅ | Full Node and Go suites pass; focused R12 and R13 harnesses pass |
| Triangulation adequate | ✅ | Success/failure and real-binary paths cover auth, quota, lifecycle, and replay boundaries |
| Safety net and live guards | ✅ | Existing suites recorded before changes; three highest-risk guards independently turned red under mutation |

**TDD compliance**: 6/6 checks passed.

### Test Layer Distribution

| Layer | Executed cases | Files | Tools |
|---|---:|---:|---|
| Unit/state/filesystem | 22 Go top-level + 17 Node | 6 | `go test`, `node:test` |
| Compiled-process integration | 5 Go top-level + 11 Node | 5 | real `acm`, fixed child-process harness |
| Offline installer E2E | 1 Node | 1 | real `sh install.sh`, fake network boundary, real built `acm` |
| **Total** | **27 Go top-level (+ subtests) + 29 Node = 56+** | **10 unique test files** | |

### Changed File Coverage

Go coverage profile: 48.4% package-wide. No Node coverage tool or branch coverage is configured.

| Changed production file | Statement coverage | Relevant changed functions | Rating |
|---|---:|---|---|
| `machine.go` | 85.8% (326/380) | `exhaustMachineQuota` 89.2%; replay, stale, persistence, and outcome paths have real-binary coverage | ⚠️ Acceptable |
| `opencode_lifecycle.go` | 81.7% (134/164) | `enableOpenCode` 72.5%; `rollbackOpenCode` 88.2%; high-risk paths also have E2E coverage | ⚠️ Acceptable |
| `main.go` | 17.7% (114/643) | R15's changed `cmdDoctor` is 100%; `cmdLs` is 77.4%; low file aggregate is dominated by unchanged legacy commands | ⚠️ Low, accepted S5 scope exclusion |

The coverage profile and command outputs are retained under `/tmp/opencode` for this verification run. Coverage is informational because the configured threshold is 0 and accepted exclusion S5 explicitly leaves broader coverage work outside this remediation chain.

### Assertion Quality

All 10 related test files were inspected. No tautology, assertion-free production path, ghost loop, smoke-only assertion, or mock-heavy test was found. Fixed parameter loops generate named cases from non-empty literals, and empty-result assertions have companion positive controls.

**Assertion quality**: ✅ All assertions verify behavior.

### Quality Metrics

- **Formatter**: ✅ `gofmt -l .` produced no output.
- **Static analysis**: ✅ `go vet ./...` produced no output.
- **Type/build checks**: ✅ `go build ./...` and `go test ./...` passed.
- **Diff hygiene**: ✅ `git diff --check` produced no output.
- **JavaScript linter/type checker**: ➖ Not configured.

## Issues Found

### CRITICAL

None.

### WARNING

None in the approved verification scope.

### SUGGESTION

None newly raised. Accepted exclusions are recorded separately and are not reopened as findings.

## Known Accepted Scope Exclusions

- **W4 — Non-corrupting idempotency behavior**: same-operation, same-profile ledger replay remains intentionally accepted. It returns current generation before persistence; R16 aligned spec/design with that behavior without changing runtime semantics.
- **S2 — Installer asset-list derivation**: `runtimeAssets` remains explicit rather than dynamically derived.
- **S4 — Legacy login cooldown clearing**: the previously accepted cooldown-file behavior remains outside R12–R16.
- **S5 — Broader coverage investment**: package-wide and Node coverage improvements remain outside scope; current assurance relies on real-binary and offline E2E contracts.

These exclusions remain residual maintenance risks, but none contradicts the current 11 requirements or 23 scenarios.

## Isolation and Final State

All runtime tests used temporary HOME/ACM/state/config/share/cache roots. The installer harness used fake offline `curl` and a real locally built binary; it verified host canaries and removed its sandbox. No real credential or OpenCode configuration was used.

All three production/config mutations were reverted with `git checkout`. `git status --short` at the end of verification showed only `?? .pi/`, which the task explicitly permits.

## Final Verdict

**PASS** — 11/11 requirements and 23/23 current scenarios are compliant; 0 CRITICAL, 0 WARNING, and 0 new SUGGESTION findings.

The round-5 CRITICAL and planned warnings are closed with passing real-process evidence and live mutation guards. The change is ready for the next SDD settlement/archive step, subject to the separately accepted scope exclusions above.
