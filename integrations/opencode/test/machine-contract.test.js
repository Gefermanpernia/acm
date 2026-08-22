import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, rm, stat, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { mapMachineResponse } from "../quota.js";

const repository = fileURLToPath(new URL("../../..", import.meta.url));
const common = {
  schema_version: "number", ok: "boolean", operation: "string", operation_id: "string",
};

function invoke(binary, env, operation, operationID, fields = {}) {
  const child = spawnSync(binary, ["machine", "v1", operation], {
    cwd: repository,
    encoding: "utf8",
    env,
    input: JSON.stringify({ schema_version: 1, operation, operation_id: operationID, ...fields }),
  });
  return { status: child.status, stderr: child.stderr, operationID, response: JSON.parse(child.stdout) };
}

function assertContract(result, operation, fields, exit = 0, responseID = result.operationID) {
  const expected = { ...common, ...fields };
  assert.equal(result.status, exit);
  assert.equal(result.stderr, "");
  assert.equal(result.response.operation, operation);
  assert.equal(result.response.operation_id, responseID);
  assert.deepEqual(Object.keys(result.response).sort(), Object.keys(expected).sort());
  assert.deepEqual(Object.fromEntries(Object.keys(expected).map((key) => [key, typeof result.response[key]])), expected);
  assert.equal(Object.keys(expected).filter((key) => expected[key] === "number")
    .every((key) => Number.isInteger(result.response[key])), true);
}

function assertError(result, operation, exit, code, retryable, responseID, fields = {}) {
  assertContract(result, operation, { error: "object", ...fields }, exit, responseID);
  assert.deepEqual(Object.keys(result.response.error).sort(), ["code", "message", "retryable"]);
  assert.equal(result.response.error.code, code);
  assert.equal(typeof result.response.error.message, "string");
  assert.equal(result.response.error.retryable, retryable);
}

function assertAdapterFields(response, fields) {
  assert.deepEqual(
    Object.fromEntries(Object.entries(fields).map(([key, type]) => [key, typeof response[key]])),
    fields,
  );
}

test("characterizes every adapter machine operation against the real binary", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "acm-machine-contract-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const binary = join(root, "acm");
  const acmDir = join(root, "acm-state");
  const profileDir = join(acmDir, "profiles", "claude", "alpha");
  const statePath = join(acmDir, "state", "opencode-machine-v1.json");
  const secrets = ["old-access", "old-refresh", "new-access", "new-refresh"];
  await mkdir(profileDir, { recursive: true });
  await writeFile(join(profileDir, ".credentials.json"), JSON.stringify({
    claudeAiOauth: { accessToken: secrets[0], refreshToken: secrets[1], expiresAt: 1 },
  }), { mode: 0o600 });
  const env = {
    ...process.env,
    HOME: join(root, "home"),
    ACM_DIR: acmDir,
    ACM_OPENCODE_CONFIG_HOME: join(root, "opencode-config"),
    ACM_DEFAULT_COOLDOWN_MIN: "1",
  };
  const build = spawnSync("go", ["build", "-o", binary, "."], { cwd: repository, encoding: "utf8" });
  assert.equal(build.status, 0, build.stderr);

  const selected = invoke(binary, env, "credential.select", "a".repeat(64));
  assertContract(selected, "credential.select", {
    profile: "string", config_dir: "string", generation: "number",
  });
  assertAdapterFields(selected.response, { profile: "string", config_dir: "string", generation: "number" });
  const unavailable = invoke(binary, env, "credential.select", "a".repeat(64));
  assertError(unavailable, "credential.select", 69, "no_available_profile", false);

  const firstLease = invoke(binary, env, "oauth.refresh.begin", "b".repeat(64), {
    profile: "alpha", generation: selected.response.generation,
  });
  assertContract(firstLease, "oauth.refresh.begin", {
    lease_id: "string", expires_at: "number", generation: "number",
  });
  assertAdapterFields(firstLease.response, { lease_id: "string" });

  const aborted = invoke(binary, env, "oauth.refresh.abort", "c".repeat(64), {
    profile: "alpha", lease_id: firstLease.response.lease_id, reason: "transient",
  });
  assertContract(aborted, "oauth.refresh.abort", { outcome: "string", generation: "number" });
  assert.equal(aborted.response.outcome, "aborted");

  const secondLease = invoke(binary, env, "oauth.refresh.begin", "d".repeat(64), {
    profile: "alpha", generation: aborted.response.generation,
  });
  const committed = invoke(binary, env, "oauth.refresh.commit", "e".repeat(64), {
    profile: "alpha", generation: secondLease.response.generation,
    lease_id: secondLease.response.lease_id, access_token: secrets[2], refresh_token: secrets[3], expires_at: 2000000000000,
  });
  assertContract(committed, "oauth.refresh.commit", { outcome: "string", generation: "number" });
  assert.equal(committed.response.outcome, "committed");

  const betaDir = join(acmDir, "profiles", "claude", "beta");
  await mkdir(betaDir, { recursive: true });
  await writeFile(join(betaDir, ".credentials.json"), "{}", { mode: 0o600 });
  const resetAt = 2000000000;
  const exhausted = invoke(binary, env, "quota.exhaust", "a".repeat(64), {
    profile: "alpha", generation: committed.response.generation, reset_at: resetAt,
  });
  assertContract(exhausted, "quota.exhaust", {
    outcome: "string", generation: "number", reset_at: "number", replacement_available: "boolean",
  });
  assert.equal(exhausted.response.outcome, "cooling");
  assert.equal(exhausted.response.reset_at, resetAt);
  assert.equal(exhausted.response.replacement_available, true);
  assertAdapterFields(exhausted.response, { outcome: "string" });
  assert.throws(() => assertContract(exhausted, "quota.exhaust", {
    outcome: "string", generation: "number", retry_after: "number",
  }), { code: "ERR_ASSERTION", message: /retry_after/ });

  const replacement = invoke(binary, env, "credential.select", "a".repeat(64));
  assert.equal(replacement.response.profile, "beta");
  const fullyExhausted = invoke(binary, env, "quota.exhaust", "a".repeat(64), {
    profile: "beta", generation: replacement.response.generation, reset_at: resetAt + 10,
  });
  assertContract(fullyExhausted, "quota.exhaust", {
    outcome: "string", generation: "number", reset_at: "number", replacement_available: "boolean",
  });
  assert.equal(fullyExhausted.response.replacement_available, false);

  const recorded = invoke(binary, env, "diagnostics.record", "0".repeat(64), { component: profileDir, event: secrets[0], outcome: "private-identifier", retryable: true });
  assertContract(recorded, "diagnostics.record", { outcome: "string" });
  const status = invoke(binary, env, "diagnostics.status", "f".repeat(64));
  assertContract(status, "diagnostics.status", { generation: "number", diagnostics: "object", active_leases: "number" });
  assert.deepEqual(status.response.diagnostics.at(-1), { time: status.response.diagnostics.at(-1).time, component: "unknown", event: "unknown", outcome: "unknown", retryable: true });
  assert.equal((await stat(statePath)).mode & 0o777, 0o600);
  assertError(invoke(binary, env, "diagnostics.record", "4".repeat(64), { body: { refresh_token: secrets[1], request: "private-request-body" } }), "diagnostics.record", 2, "invalid_request", false, "");
  assert.doesNotMatch(await readFile(statePath, "utf8"), /old-access|old-refresh|private-request-body|private-identifier|acm-machine-contract-|\/profiles\//);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], cooling: { alpha: resetAt + 20, beta: resetAt + 10 } }));
  const cooling = invoke(binary, env, "credential.select", "1".repeat(64));
  assertError(cooling, "credential.select", 75, "no_available_profile", true, undefined, { reset_at: "number" });
  assert.equal(cooling.response.reset_at, resetAt + 10);
  const coolingResponse = mapMachineResponse(cooling.response, () => resetAt * 1000);
  assert.equal(coolingResponse.status, 429);
  assert.equal(coolingResponse.headers.get("retry-after"), "10");

  const coolingState = { alpha: 1, beta: resetAt + 20 };
  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], quarantined: ["alpha", "beta"], cooling: coolingState }));
  const quarantined = invoke(binary, env, "credential.select", "2".repeat(64));
  assertError(quarantined, "credential.select", 69, "credential_quarantined", false);
  assert.match(quarantined.response.error.message, /acm login/);
  const quarantinedResponse = mapMachineResponse(quarantined.response, () => resetAt * 1000);
  assert.equal(quarantinedResponse.status, 401);
  assert.equal(quarantinedResponse.headers.get("retry-after"), null);
  assert.deepEqual(await quarantinedResponse.json(), { action: "acm login", outcome: "quarantined", retryable: false });

  const login = join(root, "synthetic-claude");
  await writeFile(login, "#!/bin/sh\nprintf '%s\\n' '{\"claudeAiOauth\":{\"accessToken\":\"synthetic-new\",\"refreshToken\":\"synthetic-new\",\"expiresAt\":2000000000000}}' > \"$CLAUDE_CONFIG_DIR/.credentials.json\"\n", { mode: 0o700 });
  const recovered = spawnSync(binary, ["login", "claude", "alpha"], { encoding: "utf8", env: { ...env, ACM_BIN_claude: login } });
  assert.equal(recovered.status, 0, recovered.stderr);
  assert.doesNotMatch(recovered.stdout + recovered.stderr, /old-access|old-refresh|synthetic-new|acm-machine-contract-|\/profiles\//);
  const recoveredState = JSON.parse(await readFile(statePath, "utf8"));
  assert.deepEqual(recoveredState.quarantined, ["beta"]);
  assert.deepEqual(recoveredState.cooling, coolingState);
  assert.deepEqual(recoveredState.diagnostics.at(-1), { time: recoveredState.diagnostics.at(-1).time, component: "oauth", event: "recovery", outcome: "recovered", retryable: false });
  const selectedAgain = invoke(binary, env, "credential.select", "5".repeat(64));
  assertContract(selectedAgain, "credential.select", { profile: "string", config_dir: "string", generation: "number" });
  assert.equal(selectedAgain.response.profile, "alpha");

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], quarantined: ["alpha"], cooling: { alpha: resetAt + 5, beta: resetAt + 15 } }));
  const mixed = invoke(binary, env, "credential.select", "3".repeat(64));
  assertError(mixed, "credential.select", 75, "no_available_profile", true, undefined, { reset_at: "number" });
  assert.equal(mixed.response.reset_at, resetAt + 15);
  assert.equal(mapMachineResponse(mixed.response, () => resetAt * 1000).headers.get("retry-after"), "15");
  assertError(invoke(binary, env, "diagnostics.status", "short"), "diagnostics.status", 2, "invalid_request", false, "");
  assertError(invoke(binary, env, "oauth.refresh.begin", "b".repeat(64), {
    profile: "alpha", generation: 1,
  }), "oauth.refresh.begin", 75, "stale_generation", true);
  const blocked = join(root, "blocked-acm-dir");
  await writeFile(blocked, "not a directory");
  assertError(invoke(binary, { ...env, ACM_DIR: blocked }, "diagnostics.status", "f".repeat(64)),
    "diagnostics.status", 74, "state_unavailable", false);

  const output = JSON.stringify([selected, firstLease, aborted, committed, exhausted, status, unavailable, cooling, quarantined, mixed]);
  assert.doesNotMatch(output, /old-access|old-refresh|new-access|new-refresh|access_token|refresh_token/);
});
