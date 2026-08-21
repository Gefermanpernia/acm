import { redactDiagnostic } from "./diagnostics.js";

const maxEvidenceBytes = 4 << 10;

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
  const detail = value?.error ?? value;
  const code = /^[a-z][a-z0-9_]{0,63}$/.test(detail?.code ?? "") ? detail.code : "machine_failed";
  if (code === "credential_quarantined") {
    return Response.json({
      action: "acm login",
      outcome: "quarantined",
      retryable: false,
    }, { status: 401 });
  }
  const headers = new Headers();
  const cooling = value?.outcome === "cooling" || code === "no_available_profile" && detail?.retryable === true;
  const delay = retryAfter(value?.reset_at, now);
  if (cooling && delay !== undefined) {
    headers.set("retry-after", String(delay));
  }
  return Response.json({
    ...(cooling ? {} : { code }),
    outcome: cooling ? "cooling" : "unavailable",
    retryable: cooling || detail?.retryable === true,
  }, { status: cooling ? 429 : 503, headers });
}

export async function handleQuotaResponse(response, context, dependencies) {
  if (response.status !== 429) return response;
  const evidence = await readEvidence(response);
  if (evidence?.error?.type !== "rate_limit_error" ||
      response.headers.get("anthropic-ratelimit-unified-status") !== "rejected") {
    return response;
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
    if (error?.code === "stale_generation") return response;
    return mapMachineResponse(error, dependencies.now ?? Date.now);
  }
  dependencies.diagnostic?.(redactDiagnostic({
    time: (dependencies.now ?? Date.now)(),
    component: "quota",
    event: "transition",
    outcome: transition.outcome,
    retryable: transition.outcome !== "quarantined",
  }));
  return mapMachineResponse(transition, dependencies.now ?? Date.now);
}
