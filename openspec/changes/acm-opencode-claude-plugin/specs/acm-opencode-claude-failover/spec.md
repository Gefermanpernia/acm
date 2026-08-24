# acm-opencode-claude-failover Specification

## Purpose

Transition ACM-managed Claude profiles after confirmed quota exhaustion while leaving logical-operation recovery to OpenCode.

## Requirements

### Requirement: Evidence-Based Quota Transition

The plugin MUST mark the active profile unavailable only after an Anthropic response contains the required confirmed quota-exhaustion evidence. It MUST apply ACM cooldown/reset policy and select the next eligible profile. Generic 401, 429, or 529 responses MUST NOT alone be classified as confirmed quota exhaustion.

#### Scenario: Confirmed quota exhaustion selects another profile

- GIVEN the active profile receives required Anthropic quota-exhaustion evidence
- WHEN the plugin records the response
- THEN ACM SHALL mark it unavailable using its cooldown/reset policy and select the next eligible profile.

#### Scenario: Generic rate-limit-like response is received

- GIVEN a response is a 401, 429, or 529 without required quota evidence
- WHEN the plugin classifies it
- THEN it MUST preserve the active profile's availability state.

### Requirement: OpenCode-Owned Retry Continuation

After a profile transition, the plugin MUST NOT replay the failed request, resume a stream, continue an agent, or replay tools. It MUST expose the newly selected profile for the subsequent OpenCode retry.

#### Scenario: OpenCode retries after transition

- GIVEN ACM selected a replacement profile after confirmed exhaustion
- WHEN OpenCode performs its next retry for the logical operation
- THEN the retry SHALL use the replacement profile without an internal plugin replay.

### Requirement: Bounded Logical-Operation Attempts

Across successive OpenCode retries, ACM MUST ensure each eligible profile is attempted at most once for the logical operation according to ACM ordering and cooldowns. Concurrent transitions MUST serialize per profile and reject stale generations.

#### Scenario: Multiple OpenCode retries consume candidates

- GIVEN two eligible profiles and successive retries for one logical operation
- WHEN the first profile is confirmed exhausted
- THEN ACM SHALL select the second once and MUST NOT reselect either exhausted attempt while that operation remains active.

#### Scenario: Concurrent stale transition arrives

- GIVEN another OpenCode instance already advanced a profile generation
- WHEN a stale exhaustion transition is submitted
- THEN ACM MUST reject it without overwriting the newer state.

### Requirement: No Eligible Profile Outcome

When no profile is eligible and one or more profiles are cooling down, ACM MUST return the earliest cooldown reset through standards-compatible retry metadata, including `Retry-After` when representable; OpenCode MUST determine waiting and retry. When every remaining profile is quarantined, revoked, or unrecoverable and no cooldown reset exists, ACM MUST return a non-retryable actionable unavailable result requiring `acm login`, MUST NOT fabricate retry metadata, and MUST NOT wait. For mixed quarantined and cooling profiles, ACM MUST derive reset metadata only from cooling profiles.

#### Scenario: Cooling profile supplies retry metadata

- GIVEN no profile is eligible and at least one profile is cooling down
- WHEN OpenCode requests a subsequent retry
- THEN ACM SHALL return the earliest cooldown reset through retry metadata and MUST NOT wait internally.

#### Scenario: Only quarantined profiles remain

- GIVEN every remaining profile is quarantined, revoked, or unrecoverable
- WHEN OpenCode requests a subsequent retry
- THEN ACM MUST return a non-retryable unavailable result requiring `acm login` without retry metadata.

#### Scenario: Cooling and quarantined profiles are mixed

- GIVEN no profile is eligible, with both cooling and quarantined profiles
- WHEN ACM produces retry metadata
- THEN it MUST use the earliest reset among cooling profiles only.
