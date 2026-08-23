import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { createPlugin } from "../index.js";
import { runMachine } from "../machine.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/quota.json", import.meta.url)));
const repository = fileURLToPath(new URL("../../..", import.meta.url));
test("maps real machine transitions without owning retry or continuation", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "acm-quota-integration-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const binary = join(root, "acm");
  const acmDir = join(root, "acm-state");
  const stateDir = join(acmDir, "state");
  const statePath = join(stateDir, "opencode-machine-v1.json");
  const env = { ...process.env, HOME: join(root, "home"), ACM_DIR: acmDir,
    ACM_OPENCODE_CONFIG_HOME: join(root, "opencode-config"), ACM_DEFAULT_COOLDOWN_MIN: "1" };
  const build = spawnSync("go", ["build", "-o", binary, "."], { cwd: repository, encoding: "utf8" });
  assert.equal(build.status, 0, build.stderr);
  await mkdir(stateDir, { recursive: true });
  const versionIO = { execFile(_command, _args, _options, callback) {
    callback(null, "2.1.236 (Claude Code)\n", "");
  } };
  const credential = (expiresAt) => JSON.stringify({ claudeAiOauth: {
    accessToken: "synthetic-access", refreshToken: "synthetic-refresh", expiresAt,
  } });
  const factoryFetch = async (selection, send) => {
    const plugin = await createPlugin({ platform: "linux", versionIO,
      machine: async (operation) => operation === "credential.select" ? selection : {},
      diagnostic: async () => {}, send })();
    const auth = await plugin.auth.loader(async () => ({ type: "oauth" }));
    return auth.fetch(new Request("https://example.invalid", {
      headers: { "x-acm-operation-id": "a".repeat(64) },
    }));
  };
  const privateProfile = "PRIVATE-CREDENTIAL-ID";
  const localAuthDir = join(acmDir, "local-auth");
  await t.test("normalizes missing local credential errors", async () => {
    const missingCredentialDir = join(localAuthDir, privateProfile);
    await assert.rejects(factoryFetch({ profile: privateProfile, config_dir: missingCredentialDir, generation: 1 },
      async () => assert.fail("provider must not be called")), (error) => {
      assert.equal(error.message.includes(missingCredentialDir), false, `leaked temp path: ${error.message}`);
      assert.equal(error.message.includes(privateProfile), false, `leaked credential identifier: ${error.message}`);
      assert.equal(error.message, "ACM Claude credentials are unavailable");
      return true;
    });
  });

  await t.test("keeps valid local credentials working", async () => {
    const validCredentialDir = join(localAuthDir, "valid-control");
    await mkdir(validCredentialDir, { recursive: true });
    await writeFile(join(validCredentialDir, ".credentials.json"), credential(9999999999999), { mode: 0o600 });
    const validControl = await factoryFetch({ profile: "valid-control", config_dir: validCredentialDir, generation: 1 },
      async (request) => {
        assert.equal(request.headers.get("authorization"), "Bearer synthetic-access");
        return new Response("ok-control");
      });
    assert.equal(validControl.status, 200);
    assert.equal(await validControl.text(), "ok-control");
  });

  for (const profile of ["alpha", "beta"]) {
    const dir = join(acmDir, "profiles", "claude", profile);
    await mkdir(dir, { recursive: true });
    await writeFile(join(dir, ".credentials.json"), credential(9999999999999), { mode: 0o600 });
  }
  const machineCalls = [];
  const machine = async (operation, fields) => {
    const response = await runMachine(operation, fields, { binary, env });
    machineCalls.push([operation, fields, response]);
    return response;
  };
  const attempt = async (session, now, send = async () => assert.fail("provider must not be called")) => {
    const plugin = await createPlugin({ platform: "linux", versionIO, machine, now, send })();
    const output = { headers: {} };
    await plugin["chat.headers"]({ sessionID: session, message: { id: "message" } }, output);
    const auth = await plugin.auth.loader(async () => ({ type: "oauth" }));
    const response = await auth.fetch(new Request("https://example.invalid", { headers: output.headers }));
    return { operationID: output.headers["x-acm-operation-id"], plugin, response };
  };
  const epoch = Math.floor(Date.now() / 1000);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], cooling: { alpha: epoch + 90, beta: epoch + 120 } }));
  const cooling = await attempt("cooling", () => epoch * 1000);
  assert.equal(cooling.response.status, 429);
  assert.equal(cooling.response.headers.get("retry-after"), "90");
  t.diagnostic(`real binary cooling mapping: status=${cooling.response.status} headers=${JSON.stringify(Object.fromEntries(cooling.response.headers))}`);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], quarantined: ["alpha", "beta"] }));
  const quarantined = await attempt("quarantined", () => epoch * 1000);
  assert.equal(quarantined.response.status, 401);
  assert.deepEqual(await quarantined.response.json(), { action: "acm login", outcome: "quarantined", retryable: false });
  t.diagnostic(`real binary quarantined mapping: status=${quarantined.response.status} headers=${JSON.stringify(Object.fromEntries(quarantined.response.headers))}`);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], quarantined: ["alpha"], cooling: { alpha: epoch + 30, beta: epoch + 150 } }));
  const mixed = await attempt("mixed", () => epoch * 1000);
  assert.equal(mixed.response.headers.get("retry-after"), "150");

  await writeFile(statePath, JSON.stringify({ generation: 0, operations: [] }));
  await writeFile(join(acmDir, "profiles", "claude", "alpha", ".credentials.json"), credential(1), { mode: 0o600 });
  machineCalls.length = 0;
  let providerCalls = 0;
  const refreshed = await attempt("refresh-quota", () => epoch * 1000, async (input) => {
    if (typeof input === "string") return Response.json({ access_token: "new-access", refresh_token: "new-refresh", expires_in: 3600 });
    providerCalls += 1;
    return responseFor(fixture.confirmed);
  });
  const quota = machineCalls.find(([operation]) => operation === "quota.exhaust");
  const diagnostic = machineCalls.find(([operation, fields]) => operation === "diagnostics.record" && fields.component === "quota");
  assert.equal(providerCalls, 1);
  assert.equal(quota[1].generation, 2);
  assert.equal(quota[2].generation, 3);
  assert.equal(refreshed.response.headers.get("retry-after"), null);
  t.diagnostic(`real binary replacement mapping: status=${refreshed.response.status} headers=${JSON.stringify(Object.fromEntries(refreshed.response.headers))}`);
  assert.deepEqual(diagnostic[1], { operation_id: diagnostic[1].operation_id, component: "quota", event: "transition", outcome: "cooling", retryable: true });
  const replacement = await runMachine("credential.select", { operation_id: refreshed.operationID }, { binary, env });
  assert.equal(replacement.profile, "beta");
  assert.deepEqual(Object.keys(refreshed.plugin).sort(), ["auth", "chat.headers"]);
  t.diagnostic(`real binary refresh/quota rotation: request-generation=${quota[1].generation} response-generation=${quota[2].generation} replacement=${replacement.profile}`);

  const fullyExhausted = await attempt("quota-no-replacement", () => epoch * 1000,
    async () => responseFor(fixture.confirmed));
  assert.equal(fullyExhausted.response.status, 429);
  assert.equal(fullyExhausted.response.headers.get("retry-after"), String(2000000000 - epoch));
  assert.deepEqual(await fullyExhausted.response.json(), { outcome: "cooling", retryable: true });
  const finalQuota = machineCalls.filter(([operation]) => operation === "quota.exhaust").at(-1);
  assert.equal(finalQuota[2].replacement_available, false);
  t.diagnostic(`real binary no-replacement mapping: status=${fullyExhausted.response.status} headers=${JSON.stringify(Object.fromEntries(fullyExhausted.response.headers))}`);

  for (const status of [401, 429, 529]) {
    await writeFile(statePath, JSON.stringify({ generation: 0, operations: [] }));
    const original = Response.json({ error: { type: "overloaded_error" } }, { status });
    const passthrough = await attempt(`passthrough-${status}`, () => epoch * 1000, async () => original);
    assert.equal(passthrough.response, original);
    assert.equal(passthrough.response.headers.get("retry-after"), null);
    t.diagnostic(`real binary unconfirmed passthrough: status=${passthrough.response.status} headers=${JSON.stringify(Object.fromEntries(passthrough.response.headers))}`);
  }

  const blocked = join(root, "blocked-acm-dir");
  await writeFile(blocked, "not a directory");
  env.ACM_DIR = blocked;
  const unavailable = await attempt("unavailable", () => epoch * 1000);
  assert.equal(unavailable.response.status, 503);
  assert.deepEqual(await unavailable.response.json(), {
    code: "state_unavailable", outcome: "unavailable", retryable: false,
  });
});

function responseFor(value) {
  return new Response(JSON.stringify(value.body), {
    status: value.status,
    headers: value.headers,
  });
}
