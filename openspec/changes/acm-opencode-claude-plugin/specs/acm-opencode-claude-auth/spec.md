# acm-opencode-claude-auth Specification

## Purpose

Provide ACM-managed Claude authentication to supported OpenCode installations without exposing credentials or taking ownership of OpenCode recovery.

## Requirements

### Requirement: ACM-Managed Ephemeral Authentication

The plugin MUST operate only on Linux and only with ACM-managed profiles. Selection and status results MUST NOT expose tokens. The plugin MUST NOT write OpenCode authentication files directly or implement retry/backoff, waiting, stream resumption, continuation, partial-output handling, blind tool replay, or tool-loop recovery.

#### Scenario: Selected profile supplies authentication

- GIVEN Linux, a supported installation, and an eligible ACM profile
- WHEN OpenCode requests Claude authentication
- THEN the plugin SHALL supply only the selected profile's ephemeral credentials
- AND OpenCode SHALL retain recovery ownership.

#### Scenario: Non-ACM or unsupported host is requested

- GIVEN a non-ACM profile or a non-Linux host
- WHEN authentication is requested
- THEN the plugin MUST refuse the operation without changing credentials.

### Requirement: Controlled Refresh Commit

The plugin MUST refresh normally expired credentials and commit them through the controlled ACM operation. Secrets MUST NOT be accepted through argv, echoed, or logged. ACM MUST serialize per-profile commits, validate profile identity and generation, persist atomically, and fail closed on validation or persistence failure.

#### Scenario: Normal expiry refresh succeeds

- GIVEN the selected profile has normally expired credentials
- WHEN the provider refresh succeeds and the generation remains current
- THEN ACM SHALL atomically commit the refreshed credentials without exposing them.

#### Scenario: Stale or failed refresh commit

- GIVEN a concurrent transition changes the profile generation, or persistence fails
- WHEN the refresh commit is attempted
- THEN ACM MUST reject the commit and MUST NOT leave partial credential state.

### Requirement: Credential Quarantine and Safe Compatibility

Rejected, revoked, or unrecoverable credentials MUST be quarantined and require `acm login`; the plugin MUST NOT retry them on every request. The declared `@opencode-ai/plugin` package range SHALL govern plugin API compatibility. Linux, ACM-managed profile, and credential-shape validation MUST remain hard gates. Claude CLI detection MUST be diagnostic-only; missing or non-exact CLI evidence MUST NOT block plugin load.

#### Scenario: Refresh credentials are revoked

- GIVEN refresh returns a rejected, revoked, or unrecoverable credential outcome
- WHEN the plugin processes the outcome
- THEN it MUST quarantine the profile and require `acm login` before reuse.

#### Scenario: Claude CLI evidence is diagnostic only

- GIVEN the declared OpenCode plugin package range resolves and Claude CLI evidence is missing or non-exact
- WHEN the plugin loads on Linux for an ACM-managed profile with valid credentials
- THEN the plugin SHALL continue without a CLI version gate
- AND it SHALL record only bounded compatibility diagnostics.

### Requirement: Redacted Diagnostics

Doctor and bounded local diagnostic events MUST expose operational state only and MUST NOT contain tokens, request/response bodies, prompts, or credential identifiers that expose private data.

#### Scenario: Doctor collects a failed refresh event

- GIVEN a refresh failure has been recorded locally
- WHEN doctor or diagnostics are requested
- THEN output SHALL contain a bounded redacted event and no sensitive value.
