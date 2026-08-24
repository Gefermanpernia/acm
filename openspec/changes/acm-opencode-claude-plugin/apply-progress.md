# Apply Progress: ACM OpenCode Claude Plugin

## Work Unit

- Change: `acm-opencode-claude-plugin`
- Mode: Strict TDD
- Delivery: `auto-chain` / `feature-branch-chain`
- Current slice: R11 `Local Auth Error Containment` (`feat/opencode-plugin-r11-auth-error-containment`)
- Parent: `feat/opencode-plugin-r10-outcome-durability` under `feature-branch-chain`
- PR 1 evidence revision: `sha256:c1002a733f7afab6595105e71146d3ce1d48c2b302dd666bf266f5c76c2584f2`
- PR 2 source revisions: `machine.go sha256:0c446c12ebaf91501ea6f906e90df6f7cdd03e11bee8a9ce2418ce763a591ca8`; `machine_test.go sha256:0ee0eb4872587f1a480c731a75b9883c60167c5893570927517ff55e45333bd5`
- PR 3 adapter revisions: `index.js sha256:f3f1cf365a8904998b88cc211bf0b80a93656823f0efa1cafd1712c24a6d2651`; `machine.js sha256:71d08bdd500d341cc2d4278f37c2a27928e651647f11c97eb799819661dcac9f`; `oauth.js sha256:34eafad5f37b71bb0f7f5e4145f57e68b22c1f1762881a46471b1191765dcc37`; `compat.js sha256:e8424aa08c009a14d1ada9a9a7498e1b41398eae347d4341d56564dd7715f90d`
- PR 4 revisions: `quota.js sha256:d2e1df9a0e4596cef6b02db2f74b4835e70f10a1a10e2d2db56c33dd2a180602`; `diagnostics.js sha256:5feb882ee4b735e62201dd5c74006caaec2db7fd3bc9140a0ad920a0e52d7687`; `quota.test.js sha256:d66261883c2c5cf04318d2198439b8393b83db1ef92f903637925b26d42e4111`; `quota-integration.test.js sha256:c6a5342c58ff18a0288103b6d995a1ecace5a26f2ccbc3ea4d81a32bf1b847bd`
- PR 4a revisions: `machine.go sha256:33bc86e798cd8d1b4f797f80fdd2f7100498208e60149e0264aadd94e11dbdd6`; `machine_test.go sha256:68e149858a0e85fb168987b844a840cc9b198cb49519565aadde70127394e474`
- PR 5 revisions: `opencode_lifecycle.go sha256:26c4f2e6632311674dafa1280b2d186f4b929f9d031f3b9f3663b57c885e89cd`; `opencode_lifecycle_test.go sha256:7901c0f6258faeff5574d2791c2baec286a116e525728b20738a55794646bbbd`

## Completed Tasks

- [x] 1.1 Added RED protocol, selection, path-containment, ledger, and process-boundary tests.
- [x] 1.2 Implemented the bounded v1 machine protocol, canonical Claude profile selection/status, generation state, logical-operation ledger, and exit taxonomy.
- [x] 1.3 Removed the spike machine tests from `main_test.go`; the real CLI process test retains command-routing coverage.
- [x] 2.1 Added RED lease contention/expiry, process death, stale generation, write/fsync failure, atomic mode, secret-absence, and terminal-quarantine tests.
- [x] 2.2 Implemented `oauth.refresh.begin|commit|abort` with stdin-only credentials, expiring leases, generation checks, atomic `0600` credential replacement, file fsync, and safe failure responses.
- [x] 2.3 Consolidated locked state updates and atomic persistence while retaining focused, process, full-suite, and race coverage.
- [x] 3.1 Added RED Node conformance and process-boundary tests with synthetic compatibility, transform, refresh, ownership, and failure fixtures.
- [x] 3.2 Implemented the Linux-only modular OpenCode adapter, fixed machine subprocess boundary, controlled refresh flow, exact compatibility matrix, and compatibility script.
- [x] 3.3 Removed the superseded replay/direct-write prototype and Bun test while retaining only synthetic Node fixtures.
- [x] 4.1 Added RED synthetic fixtures for confirmed and stale transitions, generic status passthrough, reset fallback, distinct unavailable outcomes, and bounded redaction.
- [x] 4.2 Implemented quota classification, ACM transition requests, retry-only responses, diagnostics redaction, and unrecoverable refresh quarantine.
- [x] 4.3 Kept one provider call per attempt and retained OpenCode ownership of waiting, retry, replay, and continuation.
- [x] 4a.1 Added RED Go cases for explicit and fallback cooling, stale/unknown/secret-bearing failures, ledger/generation behavior, idempotence, persistence failure, and the real CLI boundary.
- [x] 4a.2 Implemented strict `quota.exhaust` dispatch with generation-aware ledger receipts and atomic cooling persistence.
- [x] 4a.3 Reused machine state/failure helpers, extracted the quota response helper, and retained race coverage.
- [x] 5.1 Added RED lifecycle tests for opt-in, JSON/JSONC ambiguity, post-write restoration, plugin exclusivity, checksums, missing backups, and rollback.
- [x] 5.2 Implemented confirmed Linux-only migration and rollback with fixed config origins, token-preserving plugin-array edits, checksummed backups, atomic replacement, validation, restart guidance, and disabled adapter bundling.
- [x] 5.3 Replaced the stale OpenCode README prototype section and completed synthetic Go, race, Node, formatting, vet, and frozen-boundary checks.
- [x] 6.1 Added a real compiled-binary contract guard for all six adapter operations, exact response shapes, secretlessness, and exit codes 0/2/69/74/75.
- [x] 6.2 Added cooling, quarantined, and mixed-state tests; implemented earliest cooling reset metadata and actionable quarantine exhaustion.
- [x] 6.3 Centralized unavailable selection responses while preserving deterministic bounded JSON and strict request handling.
- [x] 7.1 Replaced fictional quota response stubs with real-binary cooling, quarantined, mixed-state, refresh-then-quota, and safe-failure scenarios.
- [x] 7.2 Preserved bounded machine failure metadata, mapped machine outcomes to HTTP signals, translated `reset_at`, and propagated refresh-commit generation.
- [x] 7.3 Consolidated adapter response mapping and retained one provider call with no adapter-owned wait, replay, stream, or continuation behavior.
- [x] 8.1 Added production-path RED cases for missing and mismatched observed OpenCode, SDK, and Claude CLI versions without injecting resolved versions.
- [x] 8.2 Implemented bounded offline runtime-version resolution from OpenCode/Claude CLI output and SDK package metadata, with identified OpenCode package metadata as a secondary source and fail-closed compatibility checks.
- [x] 8.3 Shared the resolver between the plugin and checker, retaining only synthetic filesystem state and the process boundary as test controls; standalone installs resolve core from `opencode --version`.
- [x] 9.1a Added RED Go and Node coverage for the bounded redacted ring, doctor aggregates/lease health, missing or failed sinks, production composition, and the real machine boundary.
- [x] 9.2a Added `diagnostics.record`, atomic `0600` persistence, bounded `diagnostics.status`, doctor aggregation, and the production adapter sink.
- [x] 9.3a Centralized finite-value redaction and locked state mutation; proved tokens, request bodies, paths, environment values, operation IDs, and private identifiers are absent from persisted diagnostics and doctor output.
- [x] 9.1b Added RED Go and real-binary coverage for successful, failed, partial, aborted, concurrent, and persistence-failed login recovery.
- [x] 9.2b Added generation-safe Claude quarantine recovery after successful login, preserving other quarantines and all cooling state.
- [x] 9.3b Centralized quarantine/cooling availability mutation and diagnostic append logic without changing machine response contracts.
- [x] 10.1 Added real-binary RED coverage for replacement-aware quota exhaustion and exact true/false `replacement_available` contracts while preserving every existing guard assertion.
- [x] 10.2 Added replacement availability to `quota.exhaust` and suppressed `Retry-After` only when another profile is immediately selectable.
- [x] 10.3 Routed confirmed machine outcomes through one adapter mapping path, added real-binary 429/529 passthrough evidence, and pinned the negative control to `retry_after`.

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1 | `machine_test.go` | Unit + process | `go test -v .`: 5/5 spike tests passed | `go test -run TestMachine .` failed before production code with the expected missing v1 signature/constants/state symbols | `go test -run TestMachine .`: exit 0, 5/5 tests passed | Version/unknown field; principal/escape; two profiles/unavailable/stale/1024 cap; normal/oversized process cases | Helpers consolidated and formatted; focused tests remained green |
| 1.2 | `machine_test.go` | Unit + process | 5/5 existing spike tests passed before modifying `main.go` | Task 1.1 tests referenced behavior absent from the spike | `go test -run TestMachine .`: exit 0, `ok github.com/Gefermanpernia/acm 1.479s` | Strict request validation, deterministic secretless JSON, canonical roots, status generation, ledger rollover, and stable exits all use distinct paths | Map encoding preserves deterministic ordering; state/path helpers were reduced without weakening tests |
| 1.3 | `machine_test.go` | Process/approval | Initial `go test -v .`: 5/5 spike tests passed | Migrated v1 tests failed against the spike before replacement | `go test -run TestMachineCLIProcess .`: exit 0, `ok github.com/Gefermanpernia/acm 1.534s` | Real binary success and oversized-input paths both pass through `main` command routing | Removed unreviewed spike `main_test.go`; final focused and race suites remained green |
| 2.1 | `machine_test.go` | Unit + process | `go test -v -run TestMachine .`: exit 0, 5/5 PR 1 tests passed | Focused and process commands both failed to compile with the expected absent `machineState.Leases`, `machineWrite`, and `machineSync` symbols | Tests were implemented by task 2.2 and then passed | Busy/expired leases; stale/expired commits; write/fsync failure; success; four abort classifications; process death before commit | Test helpers share deterministic synthetic state; no production secret appears in assertions or diagnostics |
| 2.2 | `machine_test.go` | Unit + process | PR 1 safety net remained 5/5 green | Task 2.1 RED failed against the prior implementation | `go test -run TestOAuthRefresh .`: exit 0, `ok github.com/Gefermanpernia/acm 0.144s`; harness exit 0, `ok ... 0.647s` | Four focused top-level tests with eight subtests exercise distinct success, contention, expiry, stale, I/O, and quarantine paths | Locked state mutation and atomic file persistence were separated from operation handlers |
| 2.3 | `machine_test.go` | Approval/refactor | Focused and harness tests passed before refactoring (`0.211s` and `0.831s`) | N/A: behavior-preserving refactor used the passing task 2.1 approval suite | Focused and harness tests passed after refactoring (`0.144s` and `0.647s`) | Existing branch-complete cases remained unchanged | `go test -race ./...`: exit 0, `ok github.com/Gefermanpernia/acm 2.414s` |
| 3.1 | `test/compat.test.js`, `test/machine-process.test.js` | Unit + process | N/A: replacement Node tests and modules were new; Bun was unavailable | Both commands exited 1 with `ERR_MODULE_NOT_FOUND` for the intentionally absent `compat.js` and `machine.js` | Implemented by task 3.2; focused 5/5 and harness 7/7 passed | Exact/inexact matrix cases; stable hash; transformed request; refresh begin/commit; one-attempt ownership; six subprocess rejection paths | Synthetic fixture helpers kept all network, credentials, paths, and process behavior local |
| 3.2 | `test/compat.test.js`, `test/machine-process.test.js` | Unit + process | Task 3.1 RED captured before production modules existed | Task 3.1 failed for the expected missing module boundary | `node --test integrations/opencode/test/*.test.js`: exit 0, 12/12 passed | Oversized output was separated from timeout, and a 200 ms harness deadline removed Node startup flakiness without weakening the timeout case | Machine/OAuth/compatibility boundaries were split into focused modules; tests remained green |
| 3.3 | `test/compat.test.js`, `test/machine-process.test.js` | Removal/refactor | 12/12 replacement Node tests passed before deletion; legacy Bun suite could not run because Bun is absent | N/A: removal of superseded untracked replay behavior used the approved replacement suite | Focused 5/5, harness 7/7, full Node 12/12, Go, vet, and race checks passed after deletion | Hook-surface assertion proves no replay, queue, stream, tool-loop, or agent-continuation hook; provider send count remains one | Deleted `acm-anthropic-plugin.js`, `lib/failover.js`, and `failover.test.js`; package now uses Node test |
| 4.1 | `test/quota.test.js`, `test/quota-integration.test.js` | Unit + adapter integration | Existing Node suite: exit 0, 12/12 passed | Focused exited 1 with `ERR_MODULE_NOT_FOUND` for absent `diagnostics.js`; harness exited 1 because `quota.exhaust` was not called | Implemented by task 4.2; focused 7/7 and harness 1/1 passed | Confirmed/generic/stale/reset, terminal refresh, cooling/quarantine, and redaction paths use distinct inputs | Synthetic fixtures contain no real endpoint or credential data |
| 4.2 | `test/quota.test.js`, `test/quota-integration.test.js` | Unit + adapter integration | Task 4.1 RED captured before production modules existed | Missing modules and missing transition wiring failed for the expected reasons | Focused exit 0, 7/7; harness exit 0, 1/1 | Invalid reset omits `reset_at`; unrecoverable refresh first failed with `transient`, then passed with terminal quarantine | Classifier, bounded evidence reader, response synthesis, and diagnostic redaction remain separate functions |
| 4.3 | `test/quota.test.js`, `test/quota-integration.test.js` | Approval/refactor | Focused 7/7 and harness 1/1 passed before the final diagnostics refactor | N/A: behavior-preserving refactor used passing approval tests | Focused 7/7 and harness 1/1 remained green; full Node 20/20 passed | Harness proves exactly one model-provider send and no in-plugin recovery path | Exported one-event redaction directly instead of allocating a bounded event list |
| 4a.1 | `machine_test.go` | Unit + process | `go test -count=1 ./...`: exit 0; package passed in `2.713s` | Focused and process commands exited 1 because `machineState.Cooling` and `machineOperation.Exhausted` were absent | Implemented by task 4a.2; focused cases and CLI harness passed | Explicit/missing/expired reset, stale/unknown/invalid-ID/secret inputs, persistence failure, idempotence, and replacement selection use distinct paths | Synthetic temporary state only; assertions verify concrete state, output, and exits |
| 4a.2 | `machine_test.go` | Unit + process | Task 4a.1 RED captured before production changes | Compile failed for the expected missing cooling and exhaustion state | Focused exit 0, 4/4 top-level tests with 7 subtests; process exit 0, 1/1 | Repeated operation uses its receipt before stale-generation rejection; missing and expired resets force ACM fallback policy | Handler reuses lock, atomic state writer, failure taxonomy, and bounded encoder |
| 4a.3 | `machine_test.go` | Approval/refactor | Focused and process tests passed before final test/security refinement | N/A: behavior-preserving helper consolidation used the passing quota suite | Final focused, process, full, vet, formatting, and race commands all exited 0 | Invalid operation ID and persistence-failure cases supplement the success/idempotence paths | `go test -count=1 -race ./...`: exit 0; package passed in `4.785s` |
| 5.1 | `opencode_lifecycle_test.go` | Unit + filesystem transaction | `ACM_OPENCODE_CONFIG_HOME=<temp> go test -count=1 ./...`: exit 0; package `2.486s` | Focused build exited 1 because `runOpenCodeLifecycle` was undefined | Implemented by 5.2; focused lifecycle suite passed | Explicit opt-in; dual JSON/JSONC origins; forced post-write validation conflict; upstream/ACM exclusivity; corrupt and missing backups; successful rollback | Synthetic temporary config and adapter paths only |
| 5.2 | `opencode_lifecycle_test.go` | Unit + filesystem transaction | Task 5.1 RED captured before lifecycle production code | Missing lifecycle boundary failed for the expected compile error | `ACM_OPENCODE_CONFIG_HOME=$(mktemp -d) go test -count=1 -run TestOpenCodeMigration .`: exit 0; package `0.061s` | Comments and unrelated config survive plugin-array edits; rollback checksum is verified before mutation | Reused the existing atomic fsynced writer without changing `machine.go` |
| 5.3 | Lifecycle suite + Node suite | Approval/refactor | Focused lifecycle suite passed before docs/package refinement | N/A: documentation and bundle refactor used passing lifecycle tests | Final Go, race, Node, vet, formatting, shell syntax, and diff checks exited 0 | Node suite retained 20 compatibility/quota/process cases; lifecycle retained all failure paths | README now documents only the current v1 design and lifecycle commands |
| 6.1 | `test/machine-contract.test.js` | Real-binary contract | N/A: new test; existing Node suite passed 20/20 | Deliberately expected `retry_after`; exit 1 showed real `reset_at` | Focused exit 0, 1/1; full Node exit 0, 21/21 | Six operations, all five exit classes, exact keys/types, required adapter reads, known drift negative control, and secret absence | Build uses the existing Go cache; every runtime invocation remains isolated under temporary `HOME`/`ACM_DIR` |
| 6.2 | `machine_test.go`, `test/machine-contract.test.js` | Unit + real-binary process | Machine/OAuth safety net and contract guard passed before edits | Go test exited 1 with all 3 new scenarios failing; real-binary guard exited 1 on cooling exit 69 instead of 75 | Focused Go 3/3 and real-binary guard 1/1 passed after implementation | Two cooling epochs, all-quarantined, and mixed quarantine/cooling paths prove distinct outputs and cooling-only reset selection | Minimal GREEN first scanned profile availability at the selection boundary |
| 6.3 | Same files | Approval/refactor | Focused Go 3/3 and real-binary guard 1/1 passed before extraction | N/A: behavior-preserving extraction used the exact-response approval tests | Focused Go 3/3 and real-binary guard 1/1 remained green after extraction | Exact JSON keys/types and canonical Go JSON ordering remained unchanged | `machineUnavailableResponse` now owns all cooling, quarantine, and generic unavailable response construction |
| 7.1 | `test/machine-contract.test.js`, `test/quota-integration.test.js`, `test/quota.test.js`, `test/compat.test.js` | Real-binary + unit | Relevant Node safety net passed 21/21 | Exit 1: 4 failures proved missing mapper export, lost commit generation, raw unavailable error, and absent `Retry-After` | Implemented by 7.2; focused suite passed 13/13 | Cooling, all-quarantined, mixed, refresh→quota, generic failure, and invalid reset use distinct paths | Fictional `replacement`, `quarantined`, and `retry_after` machine response stubs were removed |
| 7.2 | `machine.js`, `index.js`, `oauth.js`, `quota.js` | Adapter integration | Task 7.1 RED captured before production edits | New real-binary scenarios failed against the R1b adapter | Focused suite passed 13/13; process boundary passed 7/7 | Commit generation 2 reaches quota, generation 3 is returned, and beta is selected; unavailable exits map to 429/401/503 | Safe metadata is allowlisted and credentials returned to OpenCode exclude internal generation |
| 7.3 | Same files | Approval/refactor | Task 7.2 focused GREEN passed | N/A: response consolidation used the passing Phase 7 suite | Full Node suite passed 20/20; real-binary harness passed 1/1 | Exactly one provider call, two public hooks, generic machine failure as bounded 503, and no timer/replay surface | `mapMachineResponse` owns cooling, quarantine, and generic unavailable HTTP synthesis |
| 8.1 | `test/compat.test.js` | Unit + process boundary | Compatibility and quota integration safety net passed 6/6 | Focused exit 1: 5/11 passed and all six missing/mismatch cases failed because production ignored observed evidence | Implemented by 8.2; focused suite passed 11/11 | Missing and mismatched OpenCode, SDK, and Claude versions use six distinct cases; invalid SDK text covers unparseable evidence | Synthetic package roots and process output only; no resolved-version stub |
| 8.2 | `compat.js`, `index.js`, `test/compat.test.js` | Unit + plugin integration | Correction baseline passed 12/12 | Standalone package plus working command failed with `opencode: null`; secondary-source RED then failed 1/17 | Focused suite passed 17/17 after command-first resolution and explicit identified-package fallback | Standalone, fallback, missing, mismatch, unparseable, ambiguous, multiline, and cross-source disagreement paths are covered | Fixed errors contain no paths, environment values, or raw child output |
| 8.3 | Plugin/checker compatibility tests | Approval + runtime process | Focused compatibility suite passed before checker harness correction | Checker mutation remained accepted until core resolution used the command path | Focused suite passed 17/17; full Node suite passed 32/32 | Checker accepted pinned evidence, rejected observed OpenCode 9.9.9 despite pinned package metadata, then accepted after restore | Plugin and checker both call `resolveVersions`; only process execution is injected and package files remain real synthetic I/O |
| 9.1a | `machine_test.go`, `main_test.go`, `test/{machine-contract,quota,quota-integration}.test.js` | Unit + real-binary process | Go package, quota 6/6, and machine contract 1/1 passed before edits | Go failed on absent diagnostics symbols; quota exited 1 (6/7); machine contract and integration each exited 1 (0/1) | Focused Go and all three Node commands exited 0 | 24h expiry, 256-record cap, safe/unsafe values, missing/failing sink, real composition, doctor and process paths | Existing `retry_after` negative control and prior response contracts remain |
| 9.2a | Production Go/JS diagnostics files | State + composition | Task 9.1a RED captured before production changes | Real binary rejected `diagnostics.record`; production composition emitted no diagnostic call | Focused Go passed in `1.104s`; machine contract 1/1; quota 7/7; integration 1/1 | Direct recording and quota-driven production recording cross the real binary boundary | One locked mutation path owns persistence; failures remain best-effort and observable |
| 9.3a | Same files | Security refactor | Focused GREEN passed before final cleanup | N/A: behavior-preserving security refactor used passing 9.1a tests | Full Node 33/33, Go, race, vet, formatting, and live probe passed | Unsafe token/path/identifier values become `unknown`; request-body fields reject without mutation | Finite allowlists, bounded snapshots, aggregate doctor output, and fixed failure codes expose no private values |
| 9.1b | `machine_test.go`, `main_test.go`, `test/machine-contract.test.js` | Unit + real-binary process | Go package and machine contract passed before edits | Focused Go failed to compile on absent recovery symbols; real-binary contract retained both quarantines; a second RED proved exit 0 without credential replacement was only a partial login | Focused Go and real-binary contract passed | Success with credential replacement, failure, partial exit 0, abort, generation conflict, persistence failure, second quarantine, cooling preservation, and reselection use distinct paths | Synthetic temporary state and a local credential-replacing login executable; no provider or real credential access |
| 9.2b | `machine.go`, `main.go` | Locked state transition | Task 9.1b RED captured before production edits | Missing login recovery kept `credential.select` permanently quarantined; process exit 0 alone incorrectly recovered partial login | Focused Go passed in `0.070s`; machine contract passed 1/1 in `969.979992ms` | Only exit 0 plus changed credential bytes recovers; other login outcomes preserve state; stale generation returns 75; persistence failure returns 74 | Recovery uses `withMachineState`, expected generation, ledger pruning, atomic `0600` save, and a bounded redacted diagnostic |
| 9.3b | Same files | Approval/refactor | Focused GREEN passed before consolidation | N/A: behavior-preserving refactor used the 9.1b suite | Node 33/33, Go, race, vet, formatting, frozen boundary, and live probe passed | Existing `credential.select`, `quota.exhaust`, and OAuth response shapes remain guarded | One helper now mutates quarantine/cooling availability; one helper appends bounded diagnostics |
| 10.1 | `test/machine-contract.test.js`, `test/quota-integration.test.js` | Real-binary integration | Relevant Node safety net passed 9/9; focused Go quota tests passed | Focused real-binary command exited 1 with 0/2 passing: `replacement_available` was absent and replacement response carried `Retry-After` | Implemented by 10.2; focused command passed 2/2 | Contract proves both `replacement_available:true` and `false`; integration proves replacement, cooling, quarantine, and unconfirmed 429/529 paths | Tests reuse one compiled binary per file and isolated synthetic state |
| 10.2 | `machine.go`, `quota.js` | State transition + adapter mapping | Task 10.1 RED captured before production edits | Missing field and unwanted header failed for the expected reasons | Focused Node passed 9/9; focused Go quota suite exited 0 | Healthy replacement suppresses the header; no replacement retains earliest-reset metadata; quarantine remains non-retryable | `machineQuotaResponse` reuses `nextMachineProfile`, and failed replacement validation occurs before persistence |
| 10.3 | Same files | Approval/refactor + mutation control | Focused 9/9 passed before mapping consolidation | N/A: behavior-preserving refactor used the passing four-outcome suite | Full Node 33/33, Go, race, vet, formatting, and frozen boundary all passed | Replacing `retry_after` with `totally_bogus_field` made the guard fail 0/1; restoring it passed 1/1 | One `mapMachineResponse` return path handles replacement, cooling, quarantine, and generic machine failure; unconfirmed provider responses pass through unchanged |

## Test Summary

- Tests written and passing: 5 top-level tests.
- Layer: unit tests plus one real compiled-binary process harness.
- Approval baseline: 5 pre-existing spike tests passed before migration.
- Focused verbose result: 5/5 PASS in `1.542s` package time.
- Pure/state helpers created: strict decode, canonical profile resolution, bounded ledger pruning, atomic state replace, and advisory lock handling.
- PR 2 tests written and passing: 5 top-level tests (4 focused OAuth tests plus 1 process harness), with 8 focused subtests.
- PR 2 layer: unit/state tests plus one real compiled-binary process harness.
- PR 2 pure/state helpers: operation-specific request validation, locked state updates, lease lookup/pruning, and atomic fsynced file replacement.
- PR 3 tests written and passing: 12 Node tests (5 compatibility/adapter tests and 7 real `execFile` process cases).
- PR 3 layers: pure unit tests plus a synthetic child-process harness; no E2E or real provider traffic.
- PR 3 pure helpers: exact compatibility validation, stable operation hashing, and deterministic request transformation.
- PR 4 tests written and passing: 8 Node tests (7 focused unit/security cases plus 1 adapter integration harness); full Node suite is 20/20.
- PR 4 pure helpers: bounded quota evidence parsing, reset normalization, retry response synthesis, and allowlisted diagnostic redaction.
- PR 4a tests written and passing: 5 top-level Go tests with 7 focused subtests and one compiled-binary process harness.
- PR 4a layers: unit/state transition tests plus a real CLI process test; all state and credentials are synthetic temporary fixtures.
- PR 4a state helpers: persisted cooling map, per-operation exhaustion receipt, and shared secretless quota response construction.
- PR 5 tests written and passing: 4 top-level Go lifecycle tests covering confirmed migration, transactional restoration, invalid backups, and rollback.
- PR 5 layer: isolated filesystem transaction tests with `t.TempDir()` plus the existing 20-test synthetic Node adapter suite.
- R1a tests written and passing: 1 real-binary Node contract test; full Node suite is 21/21.
- R1a layer: process contract with a temporary compiled binary and synthetic isolated machine state; no mocks or stubs.
- R1b tests written and passing: 3 Go subtests plus 3 real-binary contract scenarios; full Node suite remains 21/21.
- R1b layers: table-driven unit/state tests and a compiled-binary process guard using only synthetic profiles and state.
- R2 tests written and passing: 13 focused Node tests, including two compiled-binary process tests; full Node suite is 20/20.
- R2 layers: pure response mapping, subprocess-boundary validation, and a real adapter-to-binary harness with synthetic state and provider responses.
- R3 tests written and passing: standalone/fallback/parser cases, six production-path rejection cases, and one checker anti-tautology harness; full Node suite is 32/32.
- R3 layer: plugin integration with real synthetic package metadata, controlled process boundaries, and a real checker subprocess; no network or credential I/O.
- R4a tests written and passing: one bounded Go ring test, one doctor test, quota sink assertions, real-binary contract recording, and production-composition integration; full Node suite is 33/33.
- R4a layers: locked state mutation, direct command behavior, adapter composition, and a compiled real-binary process boundary using only synthetic temporary state.
- R4b tests written and passing: 6 focused Go cases across successful/failed/partial/aborted/conflict/I/O paths plus the real-binary contract round trip; full Node suite remains 33/33.
- R4b layers: locked state transition, direct login command seam, and a compiled-binary process harness with synthetic temporary state and no network access.
- R5 tests added: no new top-level count; the existing real-binary contract and integration tests now cover both replacement-availability values and all four design outcomes. Full Node remains 33/33.
- R5 layers: persisted Go state transition, compiled-binary process contract, adapter integration, and a mutation-tested negative control.

## PR 4a Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `go test -count=1 -v -run '^TestMachineQuotaExhaust(RecordsCooldownAndIsIdempotent\|UsesFallbackForMissingOrInvalidReset\|RejectsStaleUnknownAndSecretsWithoutMutation\|PersistenceFailurePreservesState)$' .` → exit 0; 4/4 top-level tests and 7/7 subtests passed; package `0.208s` |
| Runtime harness | `go test -count=1 -v -run '^TestMachineQuotaExhaustCLIProcess$' .` → exit 0; compiled the real binary and passed 1/1 in package `1.054s` |
| Live binary probe | Temporary `/tmp/opencode` binary with synthetic state: `diagnostics.status` exit 0 returned `{"generation":0,"ok":true,"operation":"diagnostics.status","operation_id":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","schema_version":1}`; after synthetic selection, `quota.exhaust` exit 0 returned `{"generation":2,"ok":true,"operation":"quota.exhaust","operation_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","outcome":"cooling","reset_at":1787330287,"schema_version":1}`; binary deletion confirmed |
| Full suite | `go test -count=1 ./...` → exit 0; `ok github.com/Gefermanpernia/acm 3.665s` |
| Race suite | `go test -count=1 -race ./...` → exit 0; `ok github.com/Gefermanpernia/acm 4.785s` |
| Static analysis | `go vet ./...` → exit 0; no output |
| Formatting | `gofmt -w machine.go machine_test.go` → exit 0; final `gofmt -l .` → exit 0 with no output |
| Diff hygiene | `git diff --check` and `git diff --exit-code feat/opencode-plugin-pr3 -- integrations/` → exit 0; no output |
| Rollback boundary | Revert only PR 4a additions in `machine.go` and `machine_test.go`: quota request/state fields, dispatcher/handler/response helper, cooling-aware selection, and quota tests. PR 1–3 behavior and every file under `integrations/` remain unchanged. |

## PR 5 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `ACM_OPENCODE_CONFIG_HOME=<temporary-directory> go test -count=1 -run TestOpenCodeMigration .` → exit 0; 4/4 top-level tests passed; package `0.076s` |
| Runtime harness | `ACM_OPENCODE_CONFIG_HOME=$(mktemp -d) go test -count=1 -run TestOpenCodeMigration .` → exit 0; 4/4 passed; package `0.061s`; every test additionally replaced the environment value with its own `t.TempDir()` |
| Full Go suite | `ACM_OPENCODE_CONFIG_HOME=<temporary-directory> go test -count=1 ./...` → exit 0; package `3.637s` |
| Race suite | `ACM_OPENCODE_CONFIG_HOME=<temporary-directory> go test -count=1 -race ./...` → exit 0; package `4.311s` |
| Full Node suite | `ACM_OPENCODE_CONFIG_HOME=<temporary-directory> node --test integrations/opencode/test/*.test.js` → exit 0; 20/20 passed; `884.430964ms` |
| Static and formatting | `gofmt -l .`, `go vet ./...`, `git diff --check`, and `sh -n install.sh` → exit 0 with no output |
| Frozen boundary | `git diff --exit-code feat/opencode-plugin-pr4b-quota-adapter -- machine.go machine_test.go integrations/` → exit 0 with no output |
| Live-config safety | Every Go/Node test command set `ACM_OPENCODE_CONFIG_HOME` to a newly created temporary directory; lifecycle tests then used `t.TempDir()`. The installer was syntax-checked only and never executed. No command read or wrote `~/.config/opencode/`. |
| Rollback boundary | Revert `opencode_lifecycle.go`, `opencode_lifecycle_test.go`, the two `main.go` routing/help additions, the disabled adapter download block in `install.sh`, and the OpenCode README section. No machine or adapter source belongs to this slice. |

## R1a Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/machine-contract.test.js` → exit 0; 1/1 passed; `1683.746023ms` |
| Runtime harness | Same focused command built the real `acm` outside the repository, invoked all six operations with temporary `HOME`, `ACM_DIR`, and `ACM_OPENCODE_CONFIG_HOME`, and removed the temporary tree; exit 0 |
| RED negative control | Deliberate `retry_after` expectation → exit 1; actual keys contained `reset_at`, expected keys contained `retry_after`; 0/1 passed |
| GREEN contract | Correct `reset_at` expectation plus an expected-throw negative control → exit 0; exact success/error keys and JSON types passed |
| Full Node suite | `node --test integrations/opencode/test/*.test.js` → exit 0; 21/21 passed; `1785.576226ms` |
| Go quality | `gofmt -l .` and `go vet ./...` → exit 0 with no output; `go test -count=1 ./...` → exit 0, package `3.556s`; `go test -count=1 -race ./...` → exit 0, package `4.668s` |
| Frozen boundary | Required `git diff --exit-code feat/opencode-plugin-pr5-lifecycle -- ...` command → exit 0 with no output |
| Rollback boundary | Remove only `integrations/opencode/test/machine-contract.test.js` and revert task/progress metadata; machine and adapter behavior remains unchanged. |

## Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `go test -run TestMachine .` → exit 0; `ok github.com/Gefermanpernia/acm 1.479s` |
| Runtime harness | `go test -run TestMachineCLIProcess .` → exit 0; compiled `acm`, exercised `machine v1 credential.select`, verified empty stderr, normal output ≤16 KiB, and >64 KiB stdin rejection; `ok github.com/Gefermanpernia/acm 1.534s` |
| Full suite | `go test ./...` → exit 0; `ok github.com/Gefermanpernia/acm 1.427s` |
| Race suite | `go test -race ./...` → exit 0; `ok github.com/Gefermanpernia/acm 2.222s` |
| Formatting | `gofmt -w machine.go machine_test.go` → exit 0, no output |
| Diff hygiene | `git diff --check` → exit 0, no output |
| Rollback boundary | Remove `machine.go` and `machine_test.go`, restore the three `main.go` machine routing/help additions, and restore the unreviewed spike `main_test.go` only if the prototype itself is intentionally restored. No README, integration, installer, or OpenCode configuration files belong to this slice. |

## R1b Work Unit Evidence

| Evidence | Exact value |
|---|---|
| RED | `go test -count=1 -v -run '^TestMachineSelectDistinguishesCoolingAndQuarantinedProfiles$' .` → exit 1; all 3 subtests failed against exit 69 / generic unavailable behavior. `node --test integrations/opencode/test/machine-contract.test.js` → exit 1; cooling returned 69 instead of expected 75. |
| Focused GREEN | Same focused Go command → exit 0; 3/3 passed; package `0.044s`. |
| Runtime harness | `node --test integrations/opencode/test/machine-contract.test.js` → exit 0; 1/1 passed; `1414.98246ms`; the test built the real binary outside the repository and used isolated synthetic state. |
| Live binary probe | Temporary binary returned all-cooling exit 75 with `reset_at:2000000120`; all-quarantined exit 69 with `credential_quarantined`, `credential requires acm login`, and no reset; mixed exit 75 with cooling-only `reset_at:2000000180`; stderr was empty and binary deletion was confirmed. |
| Full Go suites | `go test -count=1 ./...` → exit 0, package `3.647s`; `go test -count=1 -race ./...` → exit 0, package `4.737s`. |
| Full Node suite | `node --test integrations/opencode/test/*.test.js` → exit 0; 21/21 passed; `1451.754433ms`. Two earlier resource-contended runs exposed the pre-existing 200ms process-test timeout and were rerun without concurrent load. |
| Static and formatting | `gofmt -w machine.go machine_test.go`, `gofmt -l .`, `go vet ./...`, and `git diff --check` → exit 0 with no output. |
| Frozen boundary | Required diff against `feat/opencode-plugin-r1-contract-guard` for all listed out-of-scope files → exit 0 with no output. |
| Rollback boundary | Revert only `machine.go`, `machine_test.go`, and the R1b additions to `integrations/opencode/test/machine-contract.test.js`; R1a and all adapter production files remain intact. |

## R2 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Safety net | Relevant Node command → exit 0; 21/21 passed; `1513.875506ms` before Phase 7 test edits. |
| RED | Focused command → exit 1; 9/13 passed and 4 failed for the expected missing mapper export, missing commit generation, raw `no_available_profile` exception, and absent `Retry-After`; `1100.712795ms`. |
| Focused GREEN | Same focused command → exit 0; 13/13 passed; `1229.837394ms`. |
| Runtime harness | `node --test integrations/opencode/test/quota-integration.test.js` → exit 0; 1/1 passed; real binary reported cooling as `429 Retry-After: 90`, used quota request generation 2 after refresh commit, returned generation 3, and selected synthetic replacement `beta`; `1314.988908ms`. |
| Machine contract | `node --test integrations/opencode/test/machine-contract.test.js` → exit 0; 1/1 passed; `1371.163002ms`; the required negative control still rejects fictional `retry_after`. |
| Full Node suite | `node --test integrations/opencode/test/*.test.js` with temporary `ACM_OPENCODE_CONFIG_HOME` → exit 0; 20/20 passed; `1165.000506ms`. |
| Go quality | `gofmt -l .` and `go vet ./...` → exit 0, no output; `go test -count=1 ./...` → exit 0, `3.198s`; `go test -count=1 -race ./...` → exit 0, `4.766s`. |
| Frozen boundary | Required diff against `feat/opencode-plugin-r1b-machine-outcomes` for `machine.go`, `machine_test.go`, `main.go`, `install.sh`, `README.md`, and `opencode_lifecycle.go` → exit 0, no output. |
| Rollback boundary | Revert only `integrations/opencode/{index,machine,oauth,quota}.js` and the four Phase 7 Node test edits; R1 machine outcomes and every Phase 8–9 file remain unchanged. |

## R3 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Safety net | `node --test integrations/opencode/test/compat.test.js` → exit 0; 12/12 passed; `519.637865ms`. |
| RED | Standalone layout RED: focused command → exit 1; 11/13 passed, `opencode` resolved `null`, and checker mutation remained accepted; `351.765891ms`. Secondary-source RED: focused command → exit 1; 16/17 passed; `492.708645ms`. |
| Focused GREEN | `node --test integrations/opencode/test/compat.test.js` → exit 0; 17/17 passed; `594.078195ms`. |
| Runtime / anti-tautology harness | The focused checker test spawned the real checker against synthetic installed metadata: matching evidence returned `OpenCode 1.18.19, SDK 1.17.12, Claude CLI 2.1.236`; observed command `9.9.9` disagreed with pinned package metadata and returned exit 1 with `OpenCode compatibility check failed`; restoring `1.18.19` returned matching success output again. |
| Real-machine behavior | `resolveVersions()` returned `{"opencode":"1.18.19","sdk":"1.17.9","claude":"2.1.236"}`. The checker returned exit 1, empty stdout, and fixed stderr `OpenCode compatibility check failed`; rejection is correct solely because observed SDK 1.17.9 differs from pinned 1.17.12. |
| Full Node suite | `node --test integrations/opencode/test/*.test.js` → exit 0; 32/32 passed; `1371.897996ms`. |
| Go quality | `gofmt -l .` and `go vet ./...` → exit 0, no output; `go test -count=1 ./...` → exit 0, package `2.436s`; `go test -count=1 -race ./...` → exit 0, package `3.551s`. |
| Frozen boundary | Required diff against `feat/opencode-plugin-r2-adapter-semantics` for all listed out-of-scope Go, lifecycle, quota, and diagnostics files → exit 0, no output. |
| Rollback boundary | Revert only `integrations/opencode/{compat,index}.js`, `scripts/check-compat.js`, and compatibility-boundary changes in `test/{compat,quota-integration}.test.js`; all R1–R2 machine and adapter semantics remain intact. |

## R4a Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Safety net | `go test -count=1 ./...` → exit 0, package `2.480s`; quota 6/6 and real machine contract 1/1 each exited 0 before test edits. |
| RED | Focused Go → exit 1 with undefined `machineDiagnostic`/`machineDiagnosticMax`; quota → exit 1, 6/7; machine contract → exit 1, 0/1 because `diagnostics.record` returned 2; production integration → exit 1, 0/1 because no diagnostic call existed. |
| Focused GREEN | `go test -count=1 -run 'TestMachineDiagnostics|TestDoctor' .` → exit 0, package `1.104s`; `node --test integrations/opencode/test/quota.test.js` → exit 0, 7/7, `153.428075ms`. |
| Runtime harness | `node --test integrations/opencode/test/machine-contract.test.js` → exit 0, 1/1, `1152.413352ms`; production integration also passed 1/1 and observed `diagnostics.record`. |
| Live binary probe | Temporary binary recorded unsafe synthetic values as `unknown`, persisted mode `600`, surfaced `unknown.unknown.unknown: 1` plus `active leases: 0` in doctor, and was deleted. Raw persisted bytes: `{"generation":0,"operations":null,"diagnostics":[{"time":1787344891141,"component":"unknown","event":"unknown","outcome":"unknown","retryable":true}]}`. |
| Bounded ring | 257 live machine writes produced `ring_count=256 evicted_oldest=True retained_newest=True`; the Go test additionally seeded and evicted a record older than 24 hours. |
| Full suites | Full Node → exit 0, 33/33, `1567.430226ms`; Go → exit 0, package `3.951s`; race → exit 0, package `6.734s`. |
| Static and formatting | `gofmt -l .`, `go vet ./...`, `git diff --check`, and the required frozen-boundary diff all exited 0 with no output. |
| Rollback boundary | Revert only diagnostics additions in `machine.go`, `main.go`, `main_test.go`, `machine_test.go`, `integrations/opencode/{index,machine,quota}.js`, and the three changed Node tests. R4b login recovery and all earlier slices remain untouched. |

## R4b Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Safety net | `go test -count=1 ./...` → exit 0, package `4.337s`; `node --test integrations/opencode/test/machine-contract.test.js` → exit 0, 1/1, `1277.55525ms`. |
| RED | Initial focused Go → exit 1 with undefined `recoverMachineProfile` and `loginInteractive`; real-binary contract → exit 1 because quarantine remained `["alpha","beta"]`. Partial-login triangulation → exit 1 because unchanged credential bytes at process exit 0 incorrectly recovered the profile. |
| Focused GREEN | Focused Go command → exit 0, package `0.070s`; `node --test integrations/opencode/test/machine-contract.test.js` → exit 0, 1/1, `969.979992ms`. |
| Runtime harness | A temporary compiled binary returned non-retryable `credential_quarantined` at exit 69, ran `acm login claude alpha` through a local exit-only executable at exit 0, persisted only `beta` as quarantined with both cooling entries unchanged, recorded `oauth.recovery.recovered`, and selected `alpha` at exit 0. Temporary binary deletion was confirmed. |
| Generation and I/O safety | A simulated concurrent generation advance made successful login recovery return exit 75 with quarantine/cooling unchanged; injected atomic persistence failure returned exit 74 with byte-identical state. |
| Full suites | Full Node → exit 0, 33/33, `1478.729026ms`; Go → exit 0, package `3.490s`; race → exit 0, package `5.758s`. |
| Static and formatting | `gofmt -w machine.go machine_test.go main.go main_test.go`, `gofmt -l .`, `go vet ./...`, `git diff --check`, and the required frozen-boundary diff all exited 0 with no output. |
| Rollback boundary | Revert only R4b changes in `machine.go`, `main.go`, `machine_test.go`, `main_test.go`, and `integrations/opencode/test/machine-contract.test.js`; R4a diagnostics and all earlier response contracts remain intact. |

### R4b Bounded Correction: Restore Legacy Cooldown Clearing

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 9.2b correction | `main_test.go` | Command/filesystem | Focused 4/4 passed in `0.051s` before the assertion flip | Focused exit 1 in `0.046s`; successful, failed, partial, and aborted cases all retained the legacy cooldown file | Focused exit 0 in `0.053s`; all four cases removed the file while quarantine, machine cooling, generation, and diagnostics assertions remained unchanged | Existing four-case table covers exit 0 with changed credentials, failed exit, partial exit 0 with unchanged credentials, and abort | Relocated the existing removal with one `defer`; no other production logic changed |

| Evidence | Exact value |
|---|---|
| Focused test | `go test -count=1 -v -run TestLoginRecoversOnlySuccessfulClaudeProfile .` → exit 0; 4/4 cases passed; package `0.053s` |
| Runtime harness | `node --test integrations/opencode/test/*.test.js` → exit 0; 33/33 passed in `1675.305615ms`, including the real-binary machine contract |
| Full quality gates | `gofmt -l .` and `go vet ./...` → exit 0 with no output; `go test -count=1 ./...` → exit 0 in `3.403s`; `go test -count=1 -race ./...` → exit 0 in `8.274s` |
| Review budget | `git diff --shortstat` → 185 additions + 15 deletions = **200/200** changed lines; the correction is line-neutral |
| Rollback boundary | Revert only the legacy-cooldown removal relocation in `main.go` and restored assertion in `main_test.go`; quarantine recovery and machine-state cooling remain intact |

## R5 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Safety net | Relevant Node command → exit 0; 9/9 passed in `1187.347609ms`. `go test -count=1 -run '^TestMachineQuotaExhaust' .` → exit 0. |
| RED | `node --test integrations/opencode/test/{machine-contract,quota-integration}.test.js` → exit 1; 0/2 passed in `903.711735ms`. The contract lacked `replacement_available`, and the real replacement path returned `Retry-After: 212642783` instead of no header. |
| Focused GREEN | `node --test integrations/opencode/test/{machine-contract,quota-integration,quota}.test.js` → exit 0; 9/9 passed in `1007.921461ms`. Focused Go quota tests also exited 0. |
| Runtime harness | `node --test integrations/opencode/test/quota-integration.test.js` → exit 0; compiled the real binary outside the repository, used temporary `HOME`, `ACM_DIR`, and `ACM_OPENCODE_CONFIG_HOME`, and deleted the temporary tree. Raw mappings: replacement `status=429 headers={"content-type":"application/json"}`; only cooling `status=429 headers={"content-type":"application/json","retry-after":"90"}`; all quarantined `status=401 headers={"content-type":"application/json"}`; unconfirmed passthrough `status=401`, `status=429`, and `status=529`, each with `headers={"content-type":"application/json"}`. |
| Guard mutation | Deliberately replaced `retry_after` with `totally_bogus_field`; the focused contract command exited 1, 0/1, because the thrown message did not match `/retry_after/`. Restored guard passed 1/1 in `1042.29275ms`. |
| Full quality gates | Full Node → exit 0, 33/33, `1028.211991ms`; `gofmt -l .` and `go vet ./...` → exit 0 with no output; Go → exit 0, `2.992s`; race → exit 0, `4.899s`; `git diff --check` → exit 0. |
| Frozen boundary | `git diff --exit-code feat/opencode-plugin-r4b-login-recovery -- install.sh README.md opencode_lifecycle.go integrations/opencode/compat.js` → exit 0 with no output. |
| Rollback boundary | Revert only `machine.go` quota-response availability, `integrations/opencode/quota.js` response mapping, and R5 additions in `test/{machine-contract,quota-integration}.test.js`; R4b login recovery and every out-of-scope file remain intact. |

## PR 2 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `go test -v -run TestOAuthRefresh .` → exit 0; 4 top-level tests and 8 subtests passed; `ok github.com/Gefermanpernia/acm 0.161s` |
| Runtime harness | `go test -v -run TestMachineRefreshLeaseProcess .` → exit 0; compiled `acm`, created a lease through stdin, let the process exit before commit, then proved a second process received `lease_busy` with empty stderr; 1/1 passed; `ok github.com/Gefermanpernia/acm 0.600s` |
| Full suite | `go test ./...` → exit 0; `ok github.com/Gefermanpernia/acm (cached)`; preceding uncached refactor run completed in `1.329s` |
| Race suite | `go test -race ./...` → exit 0; `ok github.com/Gefermanpernia/acm (cached)`; preceding uncached refactor run completed in `2.414s` |
| Static analysis | `go vet ./...` → exit 0; no output |
| Formatting | `gofmt -w machine.go machine_test.go` → exit 0; then `gofmt -l .` → exit 0 with no output |
| Diff hygiene | `git diff --check` → exit 0; no output |
| Rollback boundary | Revert only PR 2 additions in `machine.go` and `machine_test.go`: refresh request fields, lease/quarantine state, begin/commit/abort handlers, atomic fsynced writer, and OAuth refresh tests. PR 1 protocol/ledger behavior remains intact. |

## PR 3 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/compat.test.js` → exit 0; 5/5 passed in `201.081781ms` |
| Runtime harness | `node --test integrations/opencode/test/machine-process.test.js` → exit 0; 7/7 passed in `663.31596ms`; exact `machine v1 diagnostics.status` argv plus timeout, malformed/oversized stdout, stderr, exit, and schema rejection |
| Full Node suite | `node --test integrations/opencode/test/*.test.js` → exit 0; 12/12 passed in `589.236208ms` before final focused proof |
| Go formatting | `gofmt -w *.go` → exit 0, no output; final `gofmt -l .` → exit 0, no output |
| Static analysis | `go vet ./...` → exit 0, no output |
| Full Go suite | `go test ./...` → exit 0; `ok github.com/Gefermanpernia/acm (cached)` |
| Race suite | `go test -race ./...` → exit 0; `ok github.com/Gefermanpernia/acm (cached)` |
| Diff hygiene | `git diff --check` and `git diff --exit-code -- machine.go machine_test.go main.go` → exit 0, no output |
| Rollback boundary | Remove `integrations/opencode/{index,machine,oauth,compat}.js`, `compatibility.json`, `scripts/check-compat.js`, Node tests/fixtures, and the package entry-point changes. PR 1–2 Go protocol and OAuth lease/commit behavior remain intact. |

## PR 4 Work Unit Evidence

| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/quota.test.js` → exit 0; 7/7 passed in `308.915442ms` |
| Runtime harness | `node --test integrations/opencode/test/quota-integration.test.js` → exit 0; 1/1 passed in `273.773943ms`; one provider send followed by retry metadata only |
| Full Node suite | `node --test integrations/opencode/test/*.test.js` → exit 0; 20/20 passed in `1096.861508ms` |
| Source-mutating formatter | `gofmt -w *.go` → exit 0, no output |
| Go formatting | `gofmt -l .` → exit 0, no output |
| Static analysis | `go vet ./...` → exit 0, no output |
| Full Go suite | `go test ./...` → exit 0; `ok github.com/Gefermanpernia/acm (cached)` |
| Race suite | `go test -race ./...` → exit 0; `ok github.com/Gefermanpernia/acm (cached)` |
| Diff hygiene | `git diff --check` and `git diff --exit-code feat/opencode-plugin-pr3 -- machine.go machine_test.go main.go` → exit 0, no output |
| Rollback boundary | Revert `integrations/opencode/{quota,diagnostics}.js`, PR 4 wiring in `index.js`, the `quota.exhaust` allowlist entry, the unrecoverable OAuth classification, and PR 4 fixtures/tests. PR 1–3 remain intact. |

## Security and Contract Evidence

- Strictly rejects unknown request fields, mismatched operations, malformed operation hashes, and unsupported command/schema versions.
- Reads at most 64 KiB from stdin and emits at most one 16 KiB JSON object with empty stderr.
- Returns no tokens, credential contents, or credential filenames; eligibility checks only recognized regular-file presence.
- Allows only canonical contained profile roots, with the explicit `principal` → default Claude home compatibility case; rejects symlink escape and non-regular credential targets.
- Persists only operation hashes, attempted profile names, timestamps, and generation state; records expire after 24 hours and are capped at 1024.
- Uses a nonblocking advisory state lock; contention returns retryable exit 75. This provides serialized attempts and does not claim exactly-once delivery.
- Refresh secrets are accepted only by `oauth.refresh.commit` through bounded stdin and are absent from stdout, persisted machine state, stable error messages, and process stderr.
- A random 128-bit lease binds profile and current generation for 120 seconds; busy, expired, stale, invalid, and quarantined paths fail closed.
- Credential replacement writes a random temporary regular file at `0600`, fsyncs it, closes it, and atomically renames it; write/fsync failures leave the prior credential bytes unchanged.
- Only `invalid_grant`, `revoked`, and `unrecoverable` abort reasons quarantine a profile; transient aborts release the lease without quarantine.
- The adapter uses `execFile` with a fixed allowlisted operation, 5-second production timeout, 16-KiB stdout cap, strict response identity/schema checks, and rejects any stderr without exposing child output.
- Machine failures retain only stable code, retryability, and integer reset metadata; raw child messages and output never reach OpenCode.
- The internal operation header is `SHA-256(sessionID + NUL + message.id)`, is removed before the provider call, and the adapter performs exactly one provider call per OpenCode attempt.
- Refresh tokens enter only the controlled commit request on child stdin; the adapter imports no write API, never calls `client.auth.set`, and never writes OpenCode `auth.json`.
- Quota rotation requires status 429, typed `rate_limit_error`, and unified status `rejected`; generic 401/429/529 responses preserve object identity and create no transition.
- Provider evidence parsing stops at 4 KiB, diagnostics retain only five allowlisted bounded fields, and tests prove tokens, refresh tokens, prompts, and raw payloads are absent.
- Valid reset epochs are normalized to seconds; absent, expired, or malformed reset values are omitted so ACM owns fallback cooldown policy.
- The Go backend accepts only a strict 64-hex operation ID, known selected profile, and current generation; access and refresh token fields are rejected before dispatch.
- Successful exhaustion persists cooling and a per-operation idempotence receipt in the atomically replaced machine state; it never writes quarantine.
- Refresh commit generation replaces the pre-refresh selection generation before `quota.exhaust`, preventing the real stale-generation no-op.
- `reset_at` is converted to delta-seconds `Retry-After`; quarantined selection returns a fixed `401` action and generic machine failures return bounded `503` metadata.
- Missing or expired reset epochs use ACM's configured fallback cooldown, while repeated requests return the original reset without extending it.
- OpenCode lifecycle mutation requires the exact `--confirm` opt-in, a Linux host, one unambiguous regular config origin, and a regular bundled adapter entry point.
- Compatibility support pins and observed runtime versions now originate independently: bounded `opencode --version` is the primary core source, identified package metadata is secondary, SDK uses identified package metadata, and Claude uses bounded CLI output; missing, unreadable, ambiguous, multiline, conflicting, or mismatched evidence rejects with a fixed message.
- JSON/JSONC input is bounded to 1 MiB; comments and unrelated content are retained while only the top-level plugin array is replaced.
- Migration writes checksummed rollback bytes and a manifest before atomic config replacement; failed post-write validation restores the original and removes partial backup artifacts.
- Rollback validates the allowlisted config filename, backup syntax, and SHA-256 before changing configuration; missing or corrupt backups fail closed.
- The lifecycle never reads or writes ACM account state, OpenCode `auth.json`, or credentials.
- Successful Claude login recovers only the profile quarantined at the captured generation; failed, partial, aborted, stale, or persistence-failed recovery leaves quarantine and cooling state unchanged.
- Recovery records only the allowlisted `oauth.recovery.recovered` diagnostic without profile, token, path, environment, or operation identifiers.

## Review Budget

PR 1 authored Go source/test candidate versus `HEAD`:

| File | Additions | Deletions |
|---|---:|---:|
| `main.go` | 3 | 0 |
| `machine.go` | 255 | 0 |
| `machine_test.go` | 140 | 0 |
| **Total** | **398** | **0** |

The removed `main_test.go` was an untracked prototype file and therefore contributes no PR diff lines. The 398-line authored source/test candidate remains below the 400-line review ceiling.

PR 2 authored Go source/test delta against the captured PR 1 file revisions:

| File | Additions | Deletions | Changed lines |
|---|---:|---:|---:|
| `machine.go` | 181 | 9 | 190 |
| `machine_test.go` | 127 | 1 | 128 |
| **Total** | **308** | **10** | **318** |

The PR 2 slice is 318 additions-plus-deletions, within the 320-line slice budget by 2 lines. OpenSpec progress/checkbox updates are delivery metadata and follow the same exclusion used by PR 1's authored source/test budget.

PR 3 authored adapter/test delta contains 292 additions and 0 tracked deletions. The three removed prototype files were untracked before this slice and therefore contribute no PR deletion lines. The 292 additions-plus-deletions are within the 360-line slice budget by 68 lines; OpenSpec progress/checkbox metadata remains excluded consistently with PRs 1–2.

PR 4 contains 345 additions and 4 deletions across adapter source, tests, and synthetic fixtures: **349 additions-plus-deletions**, within the 350-line slice budget by 1 line. OpenSpec progress/checkbox metadata remains excluded consistently with PRs 1–3.

PR 4a contains 165 additions and 5 deletions across `machine.go` and `machine_test.go`: **170 additions-plus-deletions**, within the 300-line slice budget by 130 lines. OpenSpec progress/checkbox metadata remains excluded consistently with PRs 1–4.

PR 5 contains 349 additions and 0 deletions across `opencode_lifecycle.go`, `opencode_lifecycle_test.go`, `main.go`, `install.sh`, and `README.md`: **349 additions-plus-deletions**, within the 350-line slice budget by 1 line. OpenSpec delivery metadata remains excluded consistently with PRs 1–4a.

R1a contains 132 additions and 0 deletions in `integrations/opencode/test/machine-contract.test.js`: **132 additions-plus-deletions**, within the 200-line hard limit by 68 lines. OpenSpec delivery metadata remains excluded consistently with earlier slices.

R1b contains 94 additions and 7 deletions across `machine.go`, `machine_test.go`, and `integrations/opencode/test/machine-contract.test.js`: **101 additions-plus-deletions**, within the 220-line hard limit by 119 lines. OpenSpec delivery metadata remains excluded consistently with earlier slices.

R2 contains 144 additions and 69 deletions across four adapter modules and four Node test files: **213 additions-plus-deletions**, within the 310-line hard limit by 97 lines. OpenSpec delivery metadata remains excluded consistently with earlier slices.

R3 contains 148 additions and 15 deletions across five adapter/test files: **163 additions-plus-deletions**, within the 220-line hard limit by 57 lines. OpenSpec delivery metadata remains excluded consistently with earlier slices.

R4a contains 205 additions and 29 deletions across ten production/test files: **234 additions-plus-deletions**, within the 240-line hard limit by 6 lines. OpenSpec delivery metadata remains excluded consistently with earlier slices.

R4b contains 185 additions and 15 deletions across five production/test files: **200 additions-plus-deletions**, exactly at the 200-line hard limit. OpenSpec delivery metadata remains excluded consistently with earlier slices.

R5 contains 54 additions and 24 deletions across four production/test files: **78 additions-plus-deletions**, within the 200-line hard limit by 122 lines. OpenSpec delivery metadata remains excluded consistently with earlier slices.

## Deviations and Issues

- Deviations from design: None for PRs 1–4 within their assigned slices. Directory-entry fsync is not claimed; PR 2 fsyncs credential and state temporary files before atomic rename as required by the scoped contract.
- Issues found: The direct prototype used argv operations, direct OpenCode auth writes, and in-plugin request replay without strict protocol identity or bounds. PR 3 replaced and removed that untracked prototype.
- R4b closes the quarantine recovery risk: only successful Claude login clears the reauthenticated profile at the captured generation, while all cooling and other quarantines remain unchanged.
- PR 3 intentionally returns provider responses unchanged and does not yet classify quota evidence; quota response synthesis and diagnostics remain assigned to PR 4.
- PR 4 planning gap resolved in PR 4a: the real Go binary now implements `quota.exhaust`; the held-aside JavaScript slice was not restored or modified.
- PR 5 deviation: fresh enable requires an existing `opencode.json` or `opencode.jsonc`; this keeps origin selection fail-closed rather than guessing which file OpenCode should create.
- PR 5 safety result: the live OpenCode configuration was never read or written; all lifecycle state was synthetic and temporary.
- R1a discovery: the real `quota.exhaust` response contains `reset_at`, while the current adapter reads optional `retry_after`; the guard's negative control proves that fictional field is rejected without changing adapter behavior in this slice.
- R1b intentionally leaves adapter error-metadata preservation and HTTP response mapping to Phase 7; no adapter production file changed.
- R2 closes CRITICAL findings 1 and 4: the adapter consumes real `reset_at`, propagates refresh-commit generation, maps real unavailable outcomes, and never exposes a raw machine exception.
- R2 deleted fictional quota response stubs for `retry_after`, `replacement`, and `quarantined`; the contract guard's required negative control mentioning `retry_after` remains unchanged.
- The full Node suite's pre-existing 200ms synthetic process timeout can fail under concurrent CPU load; the required exact command passed 21/21 when run without concurrent test jobs.
- R3 removes the production matrix echo. The local standalone OpenCode installation resolves core 1.18.19 from `opencode --version`; its SDK remains 1.17.9 rather than the 1.17.12 pin, so the real-machine checker correctly rejects for that genuine mismatch.
- The checker anti-tautology harness mutates only the temporary `opencode` command output, restores it, and deletes the temporary root; `compatibility.json` remains unchanged.
- R4a intentionally excluded `acm login` quarantine recovery; R4b now completes tasks 9.1b–9.3b without changing diagnostics or failover response contracts.
- `diagnostics.status` returns at most the newest 64 events to preserve the existing 16-KiB machine response ceiling, while persistence retains the newest 256 events for at most 24 hours.
- R5 matches the four-outcome design contract with no timers, waiting, replay, continuation, compatibility, lifecycle, or login-recovery changes.
- The `retry_after` negative control is now independently meaningful: its assertion requires the thrown diff to name that exact field, and the bogus-field mutation fails.

## Remaining Tasks

None.

## Status

39/39 tasks complete. R6 ecosystem compatibility convention is ready for SDD verification.

## R6 Ecosystem Compatibility Convention
- [x] 11.1–11.3 replaced the label matrix with `@opencode-ai/plugin: ^1.18.18`, retained real-state gates, made Claude CLI detection diagnostic-only, removed the checker, and recorded ADR 0001.
### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 11.1 | `test/compat.test.js` | Unit + plugin load | 17/17 passed | Focused test exited 1: missing `detectClaudeVersion` export. | Implemented by 11.2; focused passed 8/8. | Non-development `9.9.9` and unavailable CLI paths both load. | Removed matrix-only cases and fixtures. |
| 11.2 | `compat.js`, `index.js`, package | Plugin integration | 11.1 RED | 11.1 failed before production edits. | Focused 8/8; production entry point prints `LOADED`. | Linux/profile gates and diagnostic failure remain distinct. | One detector; package resolution owns API compatibility. |
| 11.3 | Compatibility and quota integration tests | Removal/approval | Focused 8/8 | N/A: removal/ADR used the passing suite. | Full quality gates and frozen boundary passed. | Exact package range and no-CLI continuation remain covered. | Rollback is limited to R6 files and ADR 0001. |
### Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/compat.test.js` → exit 0; 8/8 passed. |
| Runtime harness | Production import before: `REFUSED: unsupported OpenCode compatibility matrix`; after: `LOADED` (diagnostic recording failure remained non-blocking). |
| Full gates | Node 24/24; formatting, vet, Go, race, and frozen-boundary commands exited 0. Authored slice: 55 additions + 138 deletions = 193/200; OpenSpec metadata excluded per prior slices. |
| Rollback boundary | Restore the deleted matrix/checker and R5 versions of `compat.js`, `index.js`, package/fixture/tests; remove ADR 0001. |

## R7 Contract Coherence
- [x] 12.1 Added a RED contract guard proving package-range load with missing and `9.9.9` CLI evidence while rejecting the stale matrix requirement.
- [x] 12.2 Amended auth R3 to retain quarantine and platform/profile/credential gates while making CLI detection diagnostic-only.
- [x] 12.3 Amended ADR 0001 with the same-slice specification rule for future superseding compatibility decisions.
### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 12.1 | `test/contract-coherence.test.js` | Structural + plugin factory | R6 compat 8/8 | 0/2; stale matrix and missing ADR rule | Implemented by 12.2–12.3; 2/2 | Missing CLI and `9.9.9` both load | Guard kept at public package/spec/ADR boundaries |
| 12.2 | Same | Specification contract | 12.1 RED | Matrix language remained authoritative | 2/2 in `71.33182ms` | Hard gates and both diagnostic paths asserted | Requirement and scenario now match ADR 0001 |
| 12.3 | Same | Decision coherence | 12.1 RED | ADR lacked same-slice rule | 2/2; Node 26/26 | Future supersession has an explicit guard | ADR and spec remain one rollback unit |
### R7 Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/contract-coherence.test.js` → exit 0; 2/2 passed in `71.33182ms` |
| Runtime harness | Same command invoked the real plugin factory with missing and `9.9.9` CLI evidence; hooks loaded and diagnostics reported `unavailable`/`9.9.9` |
| Neighbor/full adapter tests | R6 compatibility 8/8; full Node suite 26/26; both exit 0 |
| Review budget | 74 additions + 12 deletions = 86/90 changed lines, including OpenSpec delivery metadata |
| Rollback boundary | Remove the contract guard and revert only auth R3 plus ADR 0001's same-slice rule; R6 adapter behavior remains intact |

## R7 Status
42/54 planned tasks complete. R7 is ready for the R8 apply slice; R8–R11 remain pending.

## R8 Distribution Integrity
- [x] 13.1 Added an offline installer fixture that runs the real `install.sh` with fake `curl` and `acm`, fails on missing shipped assets, and asserts the complete staged runtime bundle.
- [x] 13.2 Removed the stale `compatibility.json` fetch and moved a fully downloaded hidden staging directory into `ACM_SHARE_DIR/opencode`, with prior-bundle restoration on interrupted replacement.
- [x] 13.3 Added cleanup-time host canaries for aliases, credentials, OpenCode configuration, and default install targets; cleanup also proves the entire temporary fixture tree is deleted.

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 13.1 | `test/install.test.js` | Installer process integration | Node 26/26 | Exit 1; installer returned 22 with `fixture: missing shipped asset compatibility.json` | Implemented by 13.2; focused 1/1 | Complete seven-asset bundle plus staged-fetch rejection | Fixture owns every process, filesystem, and network boundary |
| 13.2 | `install.sh` + fixture | Distribution transaction | 13.1 RED | Stale eighth asset aborted before bundle placement | Focused 1/1; staged factory exposed `auth` and `chat.headers` | Success replaces the bundle; rejected `quota.js` fetch preserves it byte-for-byte | Hidden same-share staging and prior-bundle restoration keep partial files unreachable |
| 13.3 | `test/install.test.js` | Isolation/security refactor | Focused GREEN | Host-safety assertions were written before installer changes | Focused and full Node suites pass | Host canaries and sandbox state exercise distinct negative boundaries | Cleanup asserts canaries unchanged, logs contain only sandbox targets, `.acm` is absent, and the fixture root is removed |

### R8 Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/install.test.js` → exit 0; 1/1 passed; final run `182.193158ms` |
| RED evidence | Same command before `install.sh` changed → exit 1; 0/1 passed; actual installer exit 22; stderr `fixture: missing shipped asset compatibility.json` |
| Runtime harness | The focused test executes the real `sh install.sh` twice with temp `HOME`, `ACM_BIN_DIR`, `ACM_SHARE_DIR`, and `TMPDIR`; fake `curl` serves repository bytes offline, fake `acm` handles `version`/`init`, and the staged factory loads both public hooks |
| Regression suite | `node --test "integrations/opencode/test/*.test.js"` → exit 0; 27/27 passed; `990.875044ms` |
| Host isolation | Cleanup re-reads six byte-identical host canaries covering `.bashrc`, `.zshrc`, credentials, OpenCode config, default binary, and default plugin; the command log must exclude host-home and include only sandbox HOME/BIN/SHARE; sandbox `.acm` must not exist; final `access(root)` must return `ENOENT` |
| Static checks | `sh -n install.sh && git diff --check` → exit 0; no output |
| Rollback boundary | Revert only `install.sh`, `integrations/opencode/test/install.test.js`, and R8 task/progress metadata; R7 and all machine, migration, outcome, and local-auth behavior remain unchanged |

## R8 Status
45/54 planned tasks complete. R8 closes round-4 C1 and is ready for SDD verification; R9–R11 remain pending.

## R9 Guided Migration
- [x] 14.1 Corrected the conflict contract: `enable --confirm` now exits nonzero for upstream-only and dual-plugin configurations, preserves exact bytes, and creates no backup.
- [x] 14.2 Added the explicit `--replace-upstream` migration flag, restricted backup creation to that path, and documented the guided workflow.
- [x] 14.3 Shared upstream/ACM plugin classification between conflict detection and mutation without changing checksum, ambiguity, restoration, or rollback guards.

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 14.1 | `opencode_lifecycle_test.go` | Filesystem transaction | Existing lifecycle suite passed: exit 0, package `0.042s` | Exit 1: dual-plugin migration returned success and changed bytes | Implemented by 14.2; focused suite passed | Upstream-only and dual-plugin states both require explicit replacement and create no backup before confirmation | Exact-byte and absent-manifest/backup assertions remain at the command boundary |
| 14.2 | Same + real compiled binary | CLI process | 14.1 RED captured before production edits | Two focused tests failed for implicit replacement | `go test -run TestOpenCodeMigration .` passed: exit 0, package `9.612s` | Real binary exits 2 on conflict and 0 with `--replace-upstream`; JSON and JSONC migration paths remain covered | README describes separate fresh-enable and guided-replacement paths |
| 14.3 | Same | Approval/refactor | Focused GREEN passed before helper extraction | N/A: behavior-preserving refactor used the 14.1 approval cases; no artificial RED | Focused and regression suites remained green | Existing ambiguity, post-write restoration, corrupt-checksum, missing-backup, and successful rollback cases all pass | `classifyOpenCodePlugin` is shared by detection and mutation; diff review found no out-of-scope behavior change |

### R9 Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `go test -run TestOpenCodeMigration .` → exit 0; 4/4 top-level lifecycle tests passed; package `9.612s` |
| RED evidence | Same focused command before production changes → exit 1; dual-plugin case failed with `plugin conflict changed config or returned success`; upstream-only case failed with `migration accepted without explicit replacement` |
| Runtime harness | `TestOpenCodeMigrationRealBinaryRequiresExplicitReplacement` builds the real `acm` in a temporary root; conflict returns exact exit 2 with byte-identical config and no backup, while `--replace-upstream` returns 0, removes upstream, enables ACM, and creates both backup files |
| Regression commands | `go test ./...` → exit 0, package `12.685s`; `node --test "integrations/opencode/test/*.test.js"` → exit 0, 27/27 passed, `13714.717266ms` |
| Host isolation | Every lifecycle run used `t.TempDir()` plus temporary `HOME`, `ACM_DIR`, `ACM_OPENCODE_CONFIG_HOME`, `ACM_OPENCODE_PLUGIN_PATH`, `TMPDIR`, and `GOCACHE`; no command addressed real OpenCode configuration, ACM state, profiles, or credentials |
| Static and refactor checks | `gofmt -l`, `go vet ./...`, and `git diff --check` → exit 0 with no output; manual diff review confirmed only lifecycle migration, its tests, README guidance, and SDD metadata changed |
| Rollback boundary | Revert R9 changes in `opencode_lifecycle.go`, `opencode_lifecycle_test.go`, `README.md`, and Phase 14 task/progress metadata; R8 distribution and all machine, adapter, auth, and durability behavior remain unchanged |

## R9 Status
48/54 planned tasks complete. R9 closes round-4 C2; R10–R11 remain pending.

## R10 Outcome and Durability Boundary
- [x] 15.1 Extended the compiled-binary guards for quota cooling without a replacement, refresh-begin quarantine, state contention, invalid leases, unknown logical operations, dispatcher-invalid operations, and parent-directory sync after rename.
- [x] 15.2 Synced the parent directory after atomic rename, added bounded `Retry-After: 1` to retryable 503 outcomes, retained exact non-retryable codes, and documented every 503 class in `design.md`.
- [x] 15.3 Centralized adapter outcome classification, made every synthetic cooling fixture state `replacement_available: false` explicitly, and retained exact-key, secretlessness, and mutation guards.

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 15.1 | `machine_test.go`, `test/{machine-contract,quota-integration}.test.js` | Go filesystem + compiled-binary adapter | Go machine, machine contract, and quota integration all passed before edits | Directory guard failed with `sync order = [file]`; adapter guard reported both `invalid_lease: null !== 1` and `state_busy: null !== 1`; targeted mutations made cooling/no-replacement, begin-quarantine, unknown-operation, and dispatcher-invalid guards fail on their exact changed contract | Implemented by 15.2; all three focused commands passed | Real binary induced all six machine outcomes; Go seam proved file-sync → rename → directory-sync order | Existing exact-key, exit-taxonomy, mutation, and secret-absence assertions remain intact |
| 15.2 | Same + `design.md` | Durability + response mapping | 15.1 RED captured before production edits | Missing parent sync and missing bounded retry headers failed for their intended reasons | `go test -run TestMachine .` and both required Node guards passed | Retryable 503s carry `Retry-After: 1`; non-retryable 503s retain code/no header; cooling replacement/no-replacement and quarantine remain distinct | 503 policy is documented exhaustively without changing exit codes |
| 15.3 | `test/{machine-contract,quota-integration,quota}.test.js` | Approval/refactor | Focused GREEN passed before extraction | N/A: behavior-preserving extraction used the passing outcome matrix; no artificial RED | Focused Go, compiled-binary Node, quota fixtures, and full regressions remained green | Fixtures state `replacement_available: false`; real binary separately proves `true` and `false` | `machineOutcome` owns status/body/retry classification; diff review found no out-of-scope behavior change |

### R10 Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `go test -run TestMachine .` → exit 0, package `3.242s`; machine contract → 1/1 passed in `897.017319ms`; quota integration → 1/1 passed in `946.037735ms` |
| Runtime harness | Both Node guards build the real `acm` under temporary roots. They induced cooling with and without a replacement, begin quarantine, lock contention, invalid lease, unknown operation, dispatcher invalid operation, and mapped every captured envelope through production adapter code. |
| Regression suite | `go test ./...` → exit 0, package `11.967s`; `node --test "integrations/opencode/test/*.test.js"` → exit 0, 27/27 passed in `1105.952962ms` |
| Static checks | `gofmt -l`, `go vet ./...`, and `git diff --check` → exit 0 with no output |
| Host isolation | Verification exported temporary `HOME`, `ACM_DIR`, `ACM_OPENCODE_CONFIG_HOME`, `TMPDIR`, and `GOCACHE`; binaries, state, lock files, credentials, and caches stayed under the temporary root and cleanup was trap/test-owned. No real profile or credential path was addressed. |
| Retry regression | Replacement-available quota remained `429` with no `Retry-After`; no-replacement quota remained `429` with an explicit reset-derived header. |
| Rollback boundary | Revert R10 changes in `machine.go`, `machine_test.go`, `integrations/opencode/{quota.js,test/machine-contract.test.js,test/quota-integration.test.js,test/quota.test.js}`, `design.md`, and Phase 15 task/progress metadata. R9 migration and R11 local-auth handling remain untouched. |

## R10 Status
51/54 planned tasks complete. R10 closes W1, W2, W3, W7, and S1; R11 remains pending and was not started.

## R11 Local Auth Error Containment
- [x] 16.1 Added missing-credential and healthy-control factory cases proving `auth.fetch` exposes no temporary path or credential identifier and still forwards valid credentials.
- [x] 16.2 Normalized only local credential read and JSON parse failures to the fixed `ACM Claude credentials are unavailable` error; machine failures retain the R10 response mapping.
- [x] 16.3 Extracted `safeCredentialError`, reran the factory harness, and diff-reviewed the refactor without finding any out-of-scope behavior change.

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 16.1 | `test/quota-integration.test.js` | Factory + filesystem | Focused harness passed 1/1 before edits | Exit 1; ENOENT exposed the complete temporary credential path and `PRIVATE-CREDENTIAL-ID` | Implemented by 16.2; focused harness passed | Missing credential and valid 200 `ok-control` exercise failure and healthy paths | Cases share the real factory boundary and isolated synthetic filesystem |
| 16.2 | `index.js` | Local auth boundary | 16.1 RED captured before production edits | Raw local read error escaped `auth.fetch` | Focused harness passed 3/3 | Existing real-machine cooling, quarantine, 503, quota, and passthrough mappings remained green | Catch scope ends before refresh and catches neither selection nor machine response mapping |
| 16.3 | `index.js`, factory harness | Approval/refactor | Focused GREEN passed before helper extraction | N/A: behavior-preserving extraction used the passing 16.1 cases | Focused and regression suites passed | Fixed failure and healthy control both remained explicit subtests | `safeCredentialError` owns the non-informative error; diff review confirmed no out-of-scope change |

### R11 Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `node --test integrations/opencode/test/quota-integration.test.js` → exit 0; 3/3 passed in `978.747631ms` |
| RED evidence | Same command before production changes → exit 1; `leaked temp path: ENOENT ... /PRIVATE-CREDENTIAL-ID/.credentials.json` |
| Runtime harness | The focused command invoked the real plugin factory and `auth.fetch` against a nonexistent credential path beneath temporary `ACM_DIR`; the valid control returned status 200/body `ok-control` with the expected synthetic bearer token |
| Regression suite | `node --test "integrations/opencode/test/*.test.js"` → exit 0; 29/29 passed in `1060.594336ms`; `go test ./...` → exit 0 |
| Static checks | `gofmt -l .`, `go vet ./...`, and `git diff --check` → exit 0 with no output |
| Host isolation | The harness used a fresh `mkdtemp` root, temporary `HOME`, `ACM_DIR`, and `ACM_OPENCODE_CONFIG_HOME`; all credential paths and binaries stayed beneath it and `t.after` recursively removed it |
| Rollback boundary | Revert only the safe local read/parse normalization in `integrations/opencode/index.js`, the two factory cases in `test/quota-integration.test.js`, and Phase 16 metadata; R7–R10 behavior remains intact |

## Round-4 Remediation Chain Status
54/54 planned tasks complete. R7–R11 close contract coherence, distribution integrity, guided migration, machine outcome/durability, and local auth error containment. W4 non-corrupting idempotency was deliberately not planned for this chain and remains outside its remediation scope.

## R12 Restorable Plain Opt-In
- [x] 17.1 Added no-upstream JSONC and real compiled-binary JSON paths that require an exact checksummed backup, preserve the existing-backup guard, and restore byte-identical configuration on rollback.
- [x] 17.2 Restored one unconditional backup transaction before every successful enable write and documented rollback availability for plain opt-in and explicit upstream replacement.
- [x] 17.3 Kept conflict, checksum, ambiguity, post-write restoration, cleanup, and exit behavior unchanged; reran the focused, lifecycle, real-binary, Go, Node, formatting, vet, and diff checks.

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 17.1 | `opencode_lifecycle_test.go` | Filesystem transaction + compiled binary | Existing lifecycle suite passed 4/4 before edits (`go test -count=1 -run '^TestOpenCodeMigration' .`) | Required focused command exited 1 because plain enable created neither `opencode.jsonc.acm-backup` nor the manifest | Implemented by 17.2; required focused command passed | In-process JSONC and real-binary JSON paths both verify exact backup bytes, manifest checksum, rollback byte equality, and cleanup | Shared lifecycle setup now isolates HOME, ACM_DIR, config, bin, share, and plugin paths under `t.TempDir()` |
| 17.2 | `opencode_lifecycle.go`, `README.md` | Config transaction + user guidance | 17.1 RED captured before production edits | Plain enable wrote the config while leaving no rollback artifacts | Required focused command passed in `17.538s`; README now documents both restorable routes | Existing replacement tests and the new plain-enable tests exercise both branches through the same transaction | Backup and manifest writes execute once after the existing-backup guard and before the config write |
| 17.3 | Lifecycle suite | Approval/refactor | Focused GREEN passed before isolation and guard strengthening | N/A: behavior-preserving refactor used the passing 17.1 suite; no artificial RED | All focused and regression commands passed | Second enable must return exit 2 with `ya existe un respaldo` while preserving config, backup, and manifest bytes; conflict and invalid-backup paths remain distinct | Diff review confirms only backup reachability, isolated lifecycle tests, rollback documentation, and SDD metadata changed |

### R12 Work Unit Evidence
| Evidence | Exact value |
|---|---|
| Focused test | `go test -count=1 -run '^TestOpenCodeMigration(PlainEnableCreatesRestorableBackup|RollbackAndMissingBackup)$' .` → exit 0; package `17.538s` |
| RED evidence | Same command before production changes → exit 1; `plain enable backup = "", err = ... opencode.jsonc.acm-backup: no such file or directory` |
| Runtime harness | `go test -count=1 -v -run '^TestOpenCodeMigrationPlainEnableCreatesRestorableBackup$' .` built the real `acm`, ran plain `enable --confirm`, verified backup + manifest, ran `rollback --confirm`, and restored exactly 54 bytes with SHA-256 `1699ac0e251ff28ea53a07ad47734c28369f8512cc2d31a49571f53f1d08abcc`; exit 0 |
| Lifecycle regression | `go test -count=1 -v -run '^TestOpenCodeMigration' .` → exit 0; 5/5 top-level tests passed in `26.412s`, including ambiguity/post-write cleanup, conflict/checksum, real replacement, plain enable, and missing backup |
| Full Go suite | `go test -count=1 ./...` → exit 0; package `30.783s` |
| Full Node suite | Final isolated run of `node --test "integrations/opencode/test/*.test.js"` → exit 0; 29/29 passed in `1712.169022ms` after priming the same temporary Go build cache |
| Node flake disclosure | Two preceding cold-cache isolated runs hit the known 200 ms `machine-process.test.js` subprocess timeout (28/29, then 27/29); no Node source changed, and the final exact command passed with an isolated warmed cache |
| Static checks | `gofmt -l .`, `go vet ./...`, and `git diff --check` → exit 0 with no output |
| Host isolation | Every runtime command used `env -i` plus a fresh `/tmp/opencode/acm-r12-*` root. Tests additionally used `t.TempDir()` for HOME, ACM_DIR, config, bin, share, plugin, TMPDIR, GOCACHE, binary, backup, and manifest paths. No real OpenCode config, ACM state, profile, credential, binary, or plugin path was addressed; each shell sandbox was trap-removed. |
| Review budget | Production/test/docs delta: 84 additions + 11 deletions = **95/190 changed lines** |
| Rollback boundary | Revert only the unconditional backup transaction in `opencode_lifecycle.go`, R12 additions/isolation in `opencode_lifecycle_test.go`, the plain-opt-in rollback sentence in `README.md`, and Phase 17 metadata. R7–R11 behavior remains intact. |

## Round-5 Remediation Chain Status
57/69 planned tasks complete. R12 closes C1(R5) and is ready for independent SDD verification; R13–R16 remain pending and were not started.
