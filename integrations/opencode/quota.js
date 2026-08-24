import { redactDiagnostic } from "./diagnostics.js";

const maxEvidenceBytes = 4 << 10;

function reportDiagnosticFailure(dependencies, code) {
  try { (dependencies.diagnosticError ?? console.error)(code); } catch {}
}

async function readEvidence(response) {
  const reader = response.clone().body?.getReader();
  if (!reader) return null;
  const chunks = [];
  let size = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > maxEvidenceBytes) {
      await reader.cancel();
      return null;
    }
    chunks.push(value);
  }
  try {
    return JSON.parse(Buffer.concat(chunks).toString("utf8"));
  } catch {
    return null;
  }
}

function validReset(headers, now) {
  const raw = headers.get("anthropic-ratelimit-unified-reset");
  if (!/^\d{10,13}$/.test(raw ?? "")) return undefined;
  const parsed = Number(raw);
  const seconds = parsed >= 1e12 ? Math.floor(parsed / 1000) : parsed;
  return Number.isSafeInteger(seconds) && seconds > Math.floor(now() / 1000)
    ? seconds
    : undefined;
}

function retryAfter(resetAt, now) {
  const current = Math.floor(now() / 1000);
  return Number.isInteger(resetAt) && resetAt > current ? resetAt - current : undefined;
}

export function mapMachineResponse(value, now = Date.now) {
  if (value instanceof Response) return value;
  const detail = value?.error ?? value;
  const code = /^[a-z][a-z0-9_]{0,63}$/.test(detail?.code ?? "") ? detail.code : "machine_failed";
  const quarantined = code === "credential_quarantined";
  const cooling = value?.outcome === "cooling" || code === "no_available_profile" && detail?.retryable === true;
  const headers = new Headers();
  const delay = retryAfter(value?.reset_at, now);
  if (cooling && value?.replacement_available !== true && delay !== undefined) {
    headers.set("retry-after", String(delay));
  }
  const body = quarantined ? {
    action: "acm login",
    outcome: "quarantined",
    retryable: false,
  } : {
    ...(cooling ? {} : { code }),
    outcome: cooling ? "cooling" : "unavailable",
    retryable: cooling || detail?.retryable === true,
  };
  return Response.json(body, { status: quarantined ? 401 : cooling ? 429 : 503, headers });
}

export async function handleQuotaResponse(response, context, dependencies) {
  if (response.status !== 429) return mapMachineResponse(response);
  const evidence = await readEvidence(response);
  if (evidence?.error?.type !== "rate_limit_error" ||
      response.headers.get("anthropic-ratelimit-unified-status") !== "rejected") {
    return mapMachineResponse(response);
  }
  const request = {
    operation_id: context.operationID,
    profile: context.selection.profile,
    generation: context.selection.generation,
  };
  const resetAt = validReset(response.headers, dependencies.now ?? Date.now);
  if (resetAt !== undefined) request.reset_at = resetAt;
  let transition;
  try {
    transition = await dependencies.machine("quota.exhaust", request);
  } catch (error) {
    if (error?.code === "stale_generation") return mapMachineResponse(response);
    return mapMachineResponse(error, dependencies.now ?? Date.now);
  }
  const diagnostic = redactDiagnostic({
    time: (dependencies.now ?? Date.now)(),
    component: "quota",
    event: "transition",
    outcome: transition.outcome,
    retryable: transition.outcome !== "quarantined",
  });
  if (typeof dependencies.diagnostic !== "function") {
    reportDiagnosticFailure(dependencies, "missing_diagnostic_sink");
  } else {
    try { await dependencies.diagnostic(diagnostic); }
    catch { reportDiagnosticFailure(dependencies, "record_failed"); }
  }
  return mapMachineResponse(transition, dependencies.now ?? Date.now);
}
