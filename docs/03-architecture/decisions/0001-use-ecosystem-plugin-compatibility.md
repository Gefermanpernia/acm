# 0001 Use Ecosystem Plugin Compatibility
- Status: Accepted
- Date: 2026-08-22
- Owner: ACM maintainers; approved by the user
## Context
Round-3 production evidence showed the exact OpenCode/SDK/Claude triple denied service on a shipping installation. The adapter never uses the SDK or Claude binary at runtime; package resolution already governs the OpenCode plugin API, credential bytes validate their real shape, and Anthropic's wire format is independent of the CLI label. The working `opencode-anthropic-login-via-cli@1.6.1` precedent uses only a caret plugin dependency. Confidence is high from local production-path evidence and source inspection.
## Decision
Adopt the simplest viable convention: declare `@opencode-ai/plugin` as `^1.18.18`, retain Linux, ACM-profile, and credential-shape hard gates, and treat best-effort Claude CLI detection as diagnostics only. Validation is the production entry-point load plus focused and full Node suites.
## Alternatives and consequences
Keeping or correcting the three-pin matrix was rejected because every pin labels an unused or separately enforced concern and any exact replacement repeats the same denial-of-service failure. A warning-only matrix was rejected as redundant machinery. The accepted risk is that a semantic plugin API change with an unchanged shape will surface as a runtime request failure rather than up-front rejection; this is acceptable because the stateless adapter holds credentials in memory for one request and ACM owns persistent state, limiting blast radius to a failed request rather than corrupted state or leaked credentials. The exact pin did not catch that case and only required hand-editing JSON to restore service.
## Revisit trigger
Revisit if the adapter begins using the SDK or Claude binary, stores persistent state, or an observed plugin API semantic break escapes package-shape compatibility.
