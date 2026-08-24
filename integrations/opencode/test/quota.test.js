import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { boundedDiagnostics } from "../diagnostics.js";
import { refreshCredentials } from "../oauth.js";
import { handleQuotaResponse } from "../quota.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/quota.json", import.meta.url)));
const operationID = "b".repeat(64);

function responseFor(value) {
  return new Response(JSON.stringify(value.body), {
    status: value.status,
    headers: value.headers,
  });
}

test("transitions only confirmed Anthropic quota rejection", async () => {
  const calls = [];
  const diagnostics = [];
  const result = await handleQuotaResponse(responseFor(fixture.confirmed), {
    operationID,
    selection: fixture.selection,
  }, {
    machine: async (operation, fields) => {
      calls.push([operation, fields]);
      return { outcome: "cooling", generation: 8, reset_at: 2000000000 };
    },
    now: () => 1999999955000,
    diagnostic: async (event) => diagnostics.push(event),
  });

  assert.equal(result.status, 429);
  assert.equal(result.headers.get("retry-after"), "45");
  assert.deepEqual(calls, [["quota.exhaust", {
    operation_id: operationID,
    profile: "alpha",
    generation: 7,
    reset_at: 2000000000,
  }]]);
  assert.deepEqual(diagnostics, [{ time: 1999999955000, component: "quota", event: "transition", outcome: "cooling", retryable: true }]);
});

test("keeps quota recovery working while making missing and failed diagnostics observable", async () => {
  const errors = [];
  const result = await handleQuotaResponse(responseFor(fixture.confirmed), {
    operationID, selection: fixture.selection,
  }, {
    machine: async () => ({ outcome: "cooling", generation: 8, reset_at: 2000000000 }),
    now: () => 1999999955000,
    diagnosticError: (code) => errors.push(code),
  });
  assert.equal(result.status, 429);
  const failed = await handleQuotaResponse(responseFor(fixture.confirmed), { operationID, selection: fixture.selection }, {
    machine: async () => ({ outcome: "cooling", reset_at: 2000000000 }), diagnostic: async () => { throw new Error("private"); }, diagnosticError: (code) => errors.push(code), now: () => 1999999955000,
  });
  assert.equal(failed.status, 429);
  assert.deepEqual(errors, ["missing_diagnostic_sink", "record_failed"]);
});

test("preserves generic 401, 429, and 529 responses unchanged", async () => {
  const responses = fixture.generic.map(responseFor);
  const calls = [];
  const results = await Promise.all(responses.map((response) => handleQuotaResponse(response, {
    operationID,
    selection: fixture.selection,
  }, {
    machine: async (...args) => calls.push(args),
  })));

  assert.deepEqual(results, responses);
  assert.deepEqual(calls, []);
});

test("preserves a confirmed response when the transition is stale", async () => {
  const original = responseFor(fixture.confirmed);
  const result = await handleQuotaResponse(original, {
    operationID,
    selection: fixture.selection,
  }, {
    machine: async () => {
      throw Object.assign(new Error("stale"), { code: "stale_generation" });
    },
  });

  assert.equal(result, original);
});

test("leaves fallback cooldown selection to ACM for an invalid reset", async () => {
  const value = structuredClone(fixture.confirmed);
  value.headers["anthropic-ratelimit-unified-reset"] = "not-an-epoch";
  let request;
  await handleQuotaResponse(responseFor(value), {
    operationID,
    selection: fixture.selection,
  }, {
    machine: async (_operation, fields) => {
      request = fields;
      return { outcome: "cooling", generation: 8, reset_at: 2000000090 };
    },
    diagnostic: async () => {},
  });

  assert.equal("reset_at" in request, false);
});

test("quarantines an unrecoverable refresh through the ACM lease", async () => {
  const calls = [];
  await assert.rejects(() => refreshCredentials(fixture.selection, operationID, {
    refreshToken: "synthetic-refresh",
  }, {
    machine: async (operation, fields) => {
      calls.push([operation, fields]);
      return { lease_id: "synthetic-lease" };
    },
    send: async () => Response.json({ error: "unrecoverable" }, { status: 400 }),
  }));

  assert.deepEqual(calls.map(([operation]) => operation), ["oauth.refresh.begin", "oauth.refresh.abort"]);
  assert.equal(calls[1][1].reason, "unrecoverable");
});

test("bounds diagnostics and excludes private inputs", () => {
  const events = Array.from({ length: 300 }, (_, index) => ({
    time: index,
    component: "quota".repeat(20),
    event: "transition".repeat(20),
    outcome: "cooling",
    retryable: true,
    token: "secret-access",
    refresh_token: "secret-refresh",
    raw_payload: { prompt: "private prompt", response: "private response" },
  }));
  const result = boundedDiagnostics(events);
  const serialized = JSON.stringify(result);

  assert.equal(result.length, 256);
  assert.ok(Buffer.byteLength(serialized) <= 65536);
  assert.equal(serialized.includes("secret"), false);
  assert.equal(serialized.includes("private"), false);
  assert.equal(result[0].component.length, 32);
});
