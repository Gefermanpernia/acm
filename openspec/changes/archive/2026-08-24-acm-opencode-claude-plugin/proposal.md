# Proposal: ACM-Owned OpenCode Claude Plugin

## Intent

`opencode-anthropic-login-via-cli@1.6.1` cannot consume ACM state. OpenCode remains recovery/session owner.

## Scope

### In Scope
- Linux-only ACM-managed plugin bundled with ACM.
- Go exposes secretless selection/status responses plus a controlled refresh-commit input: secrets use non-argv/non-log transport; profile identity and generation validate under lock, persist atomically, and never echo.
- Supported hooks: ephemeral credentials, Claude-compatible formatting, refresh, quota classification, and transition.
- Guided mutually-exclusive migration, backup, compatibility matrix, and rollback.

### Out of Scope
- Reimplementing OpenCode retry/backoff, queues, session state, continuation, tool-loop recovery, or stream resumption.
- Direct `auth.json`/profile writes, blind replay, non-ACM profiles, macOS, or production adoption of the spike.
- First slice: quota rotation, packaging, and migration; deliver selection plus compatible auth/formatting.

## Capabilities

### New Capabilities
- `acm-opencode-claude-auth`: Selected Claude authentication, secret boundaries, compatibility gating, and doctor.
- `acm-opencode-claude-failover`: confirmed quota transitions and retryable no-profile metadata for OpenCode recovery.
- `acm-opencode-plugin-lifecycle`: opt-in installation, migration backup, exclusivity, and rollback.

### Modified Capabilities
None; no existing capability specifications.

## Approach

Use a Go/JavaScript boundary. Go owns state and atomic secret persistence; the plugin owns hooks, credentials, formatting, refresh, transitions. OpenCode owns recovery. Serialize refresh/transition per profile; failed persistence fails closed. On confirmed exhaustion, try each eligible profile once; quarantine revoked profiles until `acm login` and return the earliest reset when none remains.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `main.go`, `main_test.go` | Modified | Machine contracts and state safety |
| `integrations/opencode/` | Modified | Plugin and compatibility tests |
| `install.sh`, `README.md` | Modified | Opt-in lifecycle and guidance |
| `openspec/` | New | Capability and change specifications |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Token split-brain or leakage | Med | Atomic commit, secretless Go responses, redaction, fail closed |
| SDK/protocol drift | Med | Pinned compatibility matrix and safe blocking |
| Unsafe replay/race | Med | OpenCode recovery ownership; serialized generation-aware transitions |

## Rollback Plan

Restore the backed-up OpenCode configuration with the upstream plugin, reconnect Anthropic, and disable the ACM plugin. No ACM state migration is required.

## Dependencies

- Supported OpenCode/Claude CLI compatibility matrix and hook contract.
- Transparent plugin refresh with ACM atomic persistence; `acm login` only for quarantined rejected, revoked, or unrecoverable credentials.

## Success Criteria

- [ ] Linux ACM profiles authenticate through the selected profile without token exposure or direct OpenCode auth-file writes.
- [ ] Unsupported sensitive operations block safely; doctor emits bounded redacted diagnostics.
- [ ] Confirmed exhaustion tries each eligible profile once, then returns the earliest reset metadata to OpenCode.
- [ ] Migration and rollback restore upstream operation without ACM state migration.
