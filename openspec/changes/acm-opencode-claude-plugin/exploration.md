## Exploration: ACM-owned OpenCode Claude integration

### Current State

ACM and the installed plugin currently solve adjacent problems but do not share state:

| Owner | Responsibilities today | Overlap and gap |
| --- | --- | --- |
| Go ACM CLI | Discovers isolated Claude profiles, resolves `CLAUDE_CONFIG_DIR`, owns current-profile and cooldown files, rotates through `orderedProfiles`/`nextAvailable`, and retries non-interactive `claude -p` in `cmdRun`. `cmdLaunch` selects an available profile but does not transparently replay interactive work. | ACM is the intended account-state authority, but OpenCode cannot consume that authority through a stable interface in the committed baseline. |
| `opencode-anthropic-login-via-cli@1.6.1` | Reads the default Claude credential store (or macOS Keychain), discovers CCS instances, snapshots OAuth credentials into OpenCode through `client.auth.set`, refreshes OAuth tokens, applies Claude Code request/response transforms, and retries once with alternate credentials for 401/429/529 recovery. | It ignores ACM profile/current/cooldown state and `CLAUDE_CONFIG_DIR`. Generic 429/529 recovery can switch credentials without confirming account-window exhaustion or notifying ACM. |
| Direct prototype | Adds a versioned secretless `acm machine select/exhaust` boundary, atomic state transitions, an ACM-specific auth loader, confirmed-quota classification, request cloning, bounded retry, and tests. | It proves important mechanics but creates unresolved token-refresh ownership, OpenCode auth synchronization races, macOS assumptions, and a copied compatibility surface. It is evidence only. |

Verified compatibility evidence: OpenCode is `1.18.19`; the installed plugin is `1.6.1`; its resolved `@opencode-ai/plugin` is `1.17.12`. The current SDK still types `auth.loader`, OAuth callbacks, custom `fetch`, and `client.auth.set`, but these are version-coupled integration APIs rather than an ACM-owned stable contract. The installed package exports only its default plugin; credential and fetch internals are bundled closures despite declaration files existing in `dist`.

Quantified observations:

- The upstream plugin consumes zero ACM profile, current-profile, or cooldown records; it searches the default Claude store plus CCS instances instead.
- Upstream recovery handles three broad statuses (401, 429, and 529); the prototype rotates only on a typed 429 plus rejected unified-quota headers.
- Prototype retries are bounded to `min(available_profiles, 8)` and never retry a generic 429 or 529.
- The prototype is approximately 1,205 authored changed lines, over three times the 400-line review budget. It declares five Go state tests and eight Bun classifier/retry cases, but Bun is unavailable and no tests were executed during exploration.
- Any implementation using `client.auth.set` retains two credential locations: Claude's canonical store and an OpenCode OAuth snapshot. “Single source of truth” must therefore mean ACM/Claude governs selection and freshness while OpenCode's copy is explicitly non-authoritative.

Relative comparison (1 = weak, 5 = strong; qualitative evidence, not a weighted decision score):

| Strategy | UX continuity | State coherence | Drift control | Security containment | Maintainability | Version independence | Testability | Rollback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Upstream plus adapter | 2 | 2 | 2 | 3 | 4 | 3 | 2 | 5 |
| ACM wrapper/composition | 3 | 3 | 3 | 3 | 2 | 1 | 2 | 4 |
| ACM replacement | 5 | 5 | 5 | 3 | 2 | 2 | 5 | 4 |
| Upstream contribution only | 1 now / 4 if accepted | 1 / 4 | 1 / 4 | 4 | 5 | 4 | 3 | 5 |

### Affected Areas

- `main.go` — ACM profile selection, cooldown transitions, existing credential-reading quota path, and the prototype machine API.
- `main_test.go` — prototype contract, traversal, unavailable-state, and concurrency tests.
- `integrations/opencode/acm-anthropic-plugin.js` — prototype auth hook, credential access, OAuth refresh, request transforms, and OpenCode auth synchronization.
- `integrations/opencode/lib/failover.js` — confirmed-quota classifier and bounded same-request replay.
- `integrations/opencode/failover.test.js` — replay and false-positive protection fixtures.
- `integrations/opencode/package.json` — plugin package/test boundary; currently lacks an explicit SDK compatibility/typecheck contract.
- `README.md` — prototype installation, behavior, and limitations; currently describes an unapproved design.
- `install.sh` — currently installs only the ACM binary and shell aliases; future plugin lifecycle ownership must be explicit rather than silently added.
- `~/.cache/opencode/packages/opencode-anthropic-login-via-cli@1.6.1/...` — read-only evidence for upstream credential, refresh, transform, and recovery behavior.
- `~/.config/opencode/opencode.json` — currently selects the upstream plugin; rollout may instruct a plugin-list change but must not directly edit credentials.

### Approaches

1. **Keep upstream plugin plus an ACM adapter** — Add a small ACM command or launcher that tries to synchronize the selected profile before OpenCode starts while leaving upstream auth and fetch behavior intact.
   - Pros: Smallest owned code surface, upstream retains OAuth/request compatibility, easiest rollback and upgrades.
   - Cons: Cannot provide reliable same-request failover; upstream ignores `CLAUDE_CONFIG_DIR`, performs recovery before ACM can classify the original response, and can diverge from ACM current/cooldowns. Synchronizing through default credential locations or `auth.json` is prohibited.
   - Prototype evidence: Undermined. The spike exists because upstream has no ACM state boundary and generic recovery remains inside its fetch closure.
   - Effort: Low initially, but it does not meet the integrated intent.

2. **ACM-owned wrapper/composition** — Instantiate the upstream default plugin, preserve selected methods/transforms, and wrap or replace its returned auth loader/fetch.
   - Pros: Could reuse upstream browser/API methods and reduce copied UX if stable composition points existed.
   - Cons: The package exports only a default plugin; its credential/fetch functions and original responses are inaccessible after internal recovery. Replacing the loader means ACM owns nearly all substantive behavior anyway, while monkey-patching `client` or cache bundles is unsafe and prohibited.
   - Prototype evidence: Mostly undermined. Its copied request boundary demonstrates that composition cannot reach the required classifier point. A wrapper could reuse labels/methods, but that value is too small for the coupling introduced.
   - Effort: Medium to high with the highest upgrade fragility.

3. **ACM-owned replacement plugin** — Make ACM the account-state authority and own the narrow OpenCode Anthropic auth/fetch integration, while treating Claude's credential store as canonical and OpenCode auth as a synchronized snapshot.
   - Pros: Only approach that can align account discovery/current/cooldowns, classify confirmed exhaustion, retry the same replayable request, and test failure behavior deterministically. It removes upstream/ACM double rotation and gives ACM explicit rollout/rollback ownership.
   - Cons: ACM inherits OpenCode hook compatibility, Claude OAuth/keychain behavior, request transforms, beta headers, stream transforms, packaging, upgrades, and incident support. The plugin necessarily handles OAuth secrets in memory and may synchronize a copy through `client.auth.set`.
   - Prototype evidence: Validates the secretless Go boundary, conservative classifier, bounded retry, and supported API direction. It undermines production readiness by directly refreshing rotating tokens without updating Claude's canonical store, calling `client.auth.set` on every credential read, guessing macOS Keychain service names, and carrying copied transforms without conformance tests.
   - Effort: High, but the work is separable into reviewable slices.

4. **Upstream contribution only** — Propose `CLAUDE_CONFIG_DIR` support and explicit credential-provider, classifier, transition, or fetch composition points upstream; keep ACM out of plugin ownership until accepted.
   - Pros: Best long-term maintenance and broadest ecosystem benefit; upstream remains responsible for OAuth and request compatibility.
   - Cons: Acceptance, API design, release timing, and generic retry policy are outside ACM control. It provides no near-term coherent ACM experience and may never expose the state/transition contract ACM needs.
   - Prototype evidence: Useful as a requirements spike for proposed extension points, but it does not prove upstream willingness or a stable composition contract.
   - Effort: Medium contribution effort with uncertain delivery.

### Recommendation

Replacement is justified **only because confirmed transparent same-request failover and ACM state coherence are core requirements**. If the goal were merely occasional manual use of one Claude account, the upstream plugin should remain: replacement maintenance would not earn its cost. Under the stated integrated intent, the adapter and wrapper strategies cannot satisfy the decisive behavior, so proceed toward an ACM-owned replacement with release gates rather than approve the current prototype.

Recommended ownership boundary:

| Go ACM CLI owns | JavaScript OpenCode plugin owns |
| --- | --- |
| Profile discovery and identity metadata; current/cooldown policy; deterministic ordering; atomic cross-process selection/exhaustion; monotonic selection generation; reset fallback; secretless versioned machine responses. | OpenCode hooks and supported `client.auth.set` calls; ephemeral credential reading for the selected profile; request/response compatibility; confirmed-quota classification; replayability decision; bounded retries; stream pass-through/transforms; redacted diagnostics. |
| Never returns access or refresh tokens and never edits OpenCode auth/config files. Existing quota code may use an access token internally but must not become the plugin transport. | Never writes Claude credential files, Keychain entries, OpenCode `auth.json`, or ACM state files directly; never logs token-bearing bodies/headers. |

Claude CLI should own OAuth refresh and canonical credential persistence. The plugin must not repeat the prototype's direct refresh-token grant unless a later, separately approved design proves how rotated credentials are safely committed to every supported Claude store. OpenCode auth synchronization should be serialized and generation-aware, occur only at connect or accepted profile transition, and be treated as a non-authoritative snapshot. Per-request `client.auth.set` should be discarded.

The smallest valuable first slice is **ACM-selected OpenCode authentication without automatic replay**: a compact secretless selector contract, one “ACM Claude profile (auto)” method, ephemeral reading of a fresh selected credential, one serialized supported OpenCode auth synchronization, and deterministic fixture-based tests. This immediately removes default-account drift and validates Linux/OpenCode compatibility before ACM assumes retry and refresh complexity. A second slice can add confirmed-quota cooldown plus one replay for materialized/cloneable pre-stream JSON requests; later slices can expand bounded multi-account retry and packaging.

Explicit first-slice non-goals: direct OAuth refresh; writing any credential store; replay after a 200/SSE stream begins; retrying generic 429/529/401; browser OAuth or CCS parity; custom proxy/insecure TLS options; automatic edits to OpenCode config; automatic installation/upgrades; macOS multi-profile support before a real Keychain contract test; real credentials in unit tests.

Prototype disposition for later implementation:

- Keep conceptually: versioned secretless machine boundary, atomic same-directory writes, bounded state lock, confirmed-quota classifier, replay attempt ceiling, fail-closed behavior, and fixture-based tests.
- Redesign: add generation-aware synchronization; separate compatibility transforms from failover policy; cap error-body reads before allocation; define clone/materialization limits; serialize auth snapshot updates; make token freshness/refresh ownership explicit; add current/minimum OpenCode compatibility tests and macOS contract tests.
- Discard: per-request `client.auth.set`, plugin-owned refresh that leaves Claude's source token stale, unverified Keychain lookup assumptions, any claim that the prototype is install-ready, and any plan to patch installed bundles or write `auth.json` directly.

Delivery should use automatic chained PRs. The 1,205-line spike forecasts high 400-line-budget risk; suggested autonomous slices are: selector/state contract, pure classifier/replay policy, minimal auth/plugin binding, then compatibility/packaging/docs. Rollout should be opt-in with mutually exclusive auth plugins; rollback is restoring `opencode-anthropic-login-via-cli@1.6.1` and reconnecting Anthropic through OpenCode, without state migration.

### Risks

- **Token refresh split-brain — MITIGATE:** Claude CLI remains canonical owner; fail closed with actionable re-login until a persistence-safe refresh contract is proven.
- **Concurrent account/auth synchronization races — MITIGATE:** return a monotonic selection generation from Go and serialize/ignore stale OpenCode snapshot writes.
- **False-positive 429 failover — MITIGATE:** require typed `rate_limit_error` plus rejected unified quota status; preserve generic 429/529 unchanged.
- **Unsafe request replay — MITIGATE:** retry only materialized or provably cloneable bodies, bound attempts and body size, and return the original response when transition or replay safety is uncertain.
- **Mid-stream failure — DEFER:** never replay after a successful response stream starts; revisit only if OpenCode/Anthropic exposes a safe resumable protocol. Owner: ACM maintainer; closure condition: documented replay-safe stream semantics.
- **OpenCode/plugin SDK and private Anthropic behavior drift — MITIGATE:** pin a supported version matrix, isolate compatibility transforms, use sanitized conformance fixtures, and require an upgrade checklist before release.
- **macOS Keychain profile ambiguity — DEFER:** do not claim support until tested against current Claude Code using isolated profiles. Owner: ACM maintainer; closure condition: deterministic service/account lookup without exposing secrets.
- **Security surface expansion — MITIGATE:** keep the Go boundary secretless, minimize token lifetime in JavaScript memory, redact all diagnostics, use `execFile` without a shell, and prohibit direct credential/auth-file writes.
- **Testability without real credentials — MITIGATE:** dependency-inject selector, credential reader, clock, fetch, and auth client; use synthetic tokens/responses, with any real-account smoke test opt-in and non-recording.
- **Upgrade and support burden — ACCEPT CONDITIONALLY:** ACM owns compatibility and rollback if replacement ships; review at every supported OpenCode or Claude Code upgrade and reconsider upstream composition if stable extension points appear.

### Ready for Proposal

Yes. The proposal should state that replacement is conditionally justified for ACM-grade transparent failover, not for simple credential sync; reject the direct prototype as production architecture; preserve it as spike evidence; require refresh, macOS, and OpenCode compatibility gates; and plan chained delivery under the 400-line budget.
