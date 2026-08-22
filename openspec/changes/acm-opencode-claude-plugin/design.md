# Design: ACM-Owned OpenCode Claude Plugin

## Technical Approach

Replace the spike with a Linux-only bundled plugin. OpenCode 1.18.19 `SessionRetry` reruns the same input, honors `Retry-After`, and SDK 1.17.12 `chat.headers` exposes stable `sessionID` and `message.id`. OpenCode owns retry/continuation; the plugin makes one provider call per attempt.

## Architecture Decisions

| Decision | Choice and rationale | Rejected |
|---|---|---|
| Boundary | Versioned stdin/stdout protocol; secrets never enter argv, stderr, or responses. | Ad-hoc argv and direct file edits. |
| Refresh | Expiring per-profile lease spans the OAuth network call; generation-checked commit. This prevents concurrent refreshes but does not claim exactly-once. | Locking only the write; plugin persistence. |
| Operation identity | `SHA-256(sessionID + NUL + message.id)` injected by `chat.headers`, consumed and removed by custom fetch. OpenCode retries preserve both IDs. ACM stores only the hash and attempted profiles (24h/1024-record cap). | Prompt/body hashing; restart-lost memory correlation. |
| Compatibility | Minimal clean implementation adapted from MIT upstream 1.6.1, split into transforms/OAuth/quota modules and fixture-checked against OpenCode 1.18.19, SDK 1.17.12, Claude CLI 2.1.236. | Opaque composition, cache patching, whole-bundle vendoring. |

## Data Flow

    chat.headers → local operation hash → auth fetch → ACM select → credential read
       → [lease begin → OAuth refresh → commit|abort] → Anthropic once
       → confirmed quota → ACM exhaust → synthetic retry-compatible response
       → OpenCode SessionRetry → next fetch/selection

`refresh.begin(profile,generation)` creates an expiring random lease. Busy leases fail before network. `refresh.commit` validates lease/profile/generation/expiry, atomically replaces `.credentials.json` at `0600`, fsyncs, increments generation, then removes the lease. `refresh.abort` releases it or quarantines classified `invalid_grant`. A crash after rotation but before commit can lose the new token; subsequent rejection requires `acm login`.

## Interfaces / Contracts

`acm machine v1 <operation>` reads one JSON object from fd 0 (64 KiB), writes one deterministic JSON object (16 KiB), and leaves stderr empty. Operations: `credential.select`, `oauth.refresh.begin|commit|abort`, `quota.exhaust`, `credential.quarantine`, `diagnostics.status`. Every request has `schema_version`, `operation`, and `operation_id`; unknown fields/versions fail. Commit stdin alone carries `lease_id`, expected profile/generation, access/refresh tokens, and expiry; output returns only generation/outcome. Exit taxonomy: `0` success; `2` invalid/version/path; `69` non-retryable auth unavailable; `74` persistence/I/O; `75` transient contention/cooling. Errors contain stable code, safe message, retryability, and operation ID.

Allowlisted profile names and canonical credential paths must remain beneath the profile root; reject symlink escape and non-regular targets.

Confirmed quota requires 429 + typed `rate_limit_error` + rejected unified-quota headers. ACM generation-checks the transition and excludes profiles already recorded for the operation. Replacement available: plugin returns bounded 429 without provider `Retry-After`, prompting SessionRetry. Only cooling profiles: 429 with earliest-reset `Retry-After`. All quarantined: non-retryable 401 with `acm login` action and no retry header. Unconfirmed 401/429/529 pass through unchanged.

## File Changes / Prototype Disposition

| Files | Action |
|---|---|
| `machine.go`, `machine_test.go` | Create protocol, leases, generations, operation ledger, quarantine, atomic persistence. Redesign spike `runMachine`/global lock. |
| `main.go`, `main_test.go` | Modify routing/doctor; remove spike machine tests after migration. |
| `integrations/opencode/{index,machine,oauth,compat,quota,diagnostics}.js` | Create modular plugin; delete `acm-anthropic-plugin.js` and `lib/failover.js` (waiting/replay/direct persistence). |
| `integrations/opencode/test/*.test.js`, `fixtures/**`, `compatibility.json`, `package.json` | Replace Bun replay tests with Node conformance tests. |
| `opencode_lifecycle.go`, tests, `install.sh`, `README.md` | Bundle disabled plugin; guided enable/rollback. |

## Testing Strategy

Strict RED→GREEN with `go test ./...`, `go test -race ./...`, and Node 26 `node --test`; Go passes, while Bun is absent. Synthetic OAuth/quota/SSE fixtures cover transforms, redaction, generic statuses, and retry responses. Process tests inject slow refresh, concurrency, expiry, kill-before-commit, stale generation, write/fsync failure, oversized JSON, and no real credentials.

## Threat Matrix

| Boundary | Applicability; safe/failure behavior | Planned RED test |
|---|---|---|
| Documentation-like paths | N/A: no executable classification. | — |
| Git repository selection | N/A: no Git invocation. | — |
| Commit state | N/A: no commits. | — |
| Push state | N/A: no pushes. | — |
| PR commands | N/A: no PR automation. | — |
| ACM subprocess | Applicable: `execFile`, fixed binary/operation, bounded stdin/stdout/timeout; reject schema, overflow, timeout, nonempty stderr. | `TestMachineProcessBoundaryRejectsMalformedAndOversizedIO` |
| OpenCode config editing | Applicable: detect JSON/JSONC origins, confirm, back up each file, token-aware edit, atomic replace, validate effective config; any ambiguity restores all. | `TestOpenCodeMigrationRollbackOnJSONCConflict` |

## Migration / Rollout

`acm opencode enable` requires opt-in, compatibility/origin checks, exclusivity confirmation, checksummed backups, JSON/JSONC-preserving edits, validation, then OpenCode restart. Rollback verifies checksums and restores configuration only. Diagnostics are a 256-event/64-KiB `0600` ring of time, component, event code, outcome, retryability, and versions—never prompts, bodies, tokens, IDs, paths, or private identifiers; doctor reports aggregates and compatibility/lease health.

Review slices, each under 400 authored changed lines: (1) protocol/state ledger; (2) OAuth lease/commit; (3) compatibility plugin and fixtures; (4) quota responses/diagnostics; (5) lifecycle/package/docs. Each depends on the preceding contract and is independently testable/rollbackable.

## Open Questions

None.
