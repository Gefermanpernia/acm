# Archive Report: ACM-Owned OpenCode Claude Plugin

## Final Status

- Change: `acm-opencode-claude-plugin`
- Status: fully delivered and merged to `main`
- Main commit: `2fd63eb`
- Verification: PASS, round 6
- Requirements: 11/11
- Scenarios: 23/23 (7 authentication, 8 failover, 8 lifecycle)
- In-scope findings: 0
- Mutation guards: 3/3 turned the focused suites red and were reverted byte-identically

The final state supersedes stale intermediate claims in `apply-progress.md` or earlier verification snapshots. The flaky machine-process timeout was fixed in `5b27806`; the dedicated timeout stub remains at 200ms and other modes use 5000ms. The maintainer-resolved R15 decision keeps doctor profile names redacted as `unknown`; real names remain available through `acm ls`. This is closed, not an open item.

## Delivery

The feature-branch chain and tracker integration were fully merged. PRs #1–#10 are merged, including tracker merge commit `2fd63eb`; all related local and remote feature branches were deleted. The repository has no CI workflows; verification relied on the local Node/Go suites and mutation testing. The unrelated untracked `.pi/` directory was left untouched.

## Accepted Exclusions

- W4: non-corrupting idempotency gap; same-operation, same-profile ledger replay remains intentional and returns the current generation before persistence.
- S2: installer asset-list derivation remains explicit.
- S4: legacy login cooldown clearing remains outside this change.
- S5: broader coverage investment remains outside scope; current assurance relies on real-binary and offline E2E contracts.

These are accepted maintenance-scope exclusions, not blockers or open findings.

## Artifacts Read

- OpenSpec: `proposal.md`, `design.md`, `tasks.md`, `apply-progress.md`, `verify-report.md`
- Delta specs: `acm-opencode-claude-auth/spec.md`, `acm-opencode-claude-failover/spec.md`, `acm-opencode-plugin-lifecycle/spec.md`
- Configuration: `openspec/config.yaml`

## Archive Operations

- Synced all three delta specs into `openspec/specs/` by mechanical copy.
- Moved the complete change directory to `openspec/changes/archive/2026-08-24-acm-opencode-claude-plugin/`.
- Persisted this report to Engram topic `sdd/acm-opencode-claude-plugin/archive-report`.

### Mechanical Readback

The required `diff -r` output for each of the three spec copies was empty. The required recursive pre-move snapshot comparison was empty. No byte differences were reported.

## Completion

All 69 implementation task checkboxes in the persisted `tasks.md` were checked. The SDD cycle is complete.
