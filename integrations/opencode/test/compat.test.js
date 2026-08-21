import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { assertCompatibility, operationId, resolveVersions, transformRequest } from "../compat.js";
import { createPlugin } from "../index.js";
import { refreshCredentials } from "../oauth.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/cases.json", import.meta.url)));

async function versionBoundary(t, observed) {
  const root = await mkdtemp(join(tmpdir(), "acm-compat-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const bin = join(root, "bin");
  const sdk = join(root, "node_modules", "@opencode-ai", "sdk");
  await mkdir(bin, { recursive: true });
  await mkdir(sdk, { recursive: true });
  const opencodeOutput = observed.opencodeOutput ?? observed.opencode ?? "";
  if (observed.rootPackage !== undefined) await writeFile(join(root, "package.json"), JSON.stringify(observed.rootPackage));
  else if (observed.opencode !== undefined) await writeFile(join(root, "package.json"), JSON.stringify({ name: "opencode-ai", version: observed.opencode }));
  if (observed.sdk !== undefined) await writeFile(join(sdk, "package.json"), JSON.stringify({ name: "@opencode-ai/sdk", version: observed.sdk }));
  await writeFile(join(bin, "opencode"), `#!/bin/sh\nprintf '${opencodeOutput}\\n'\n`, { mode: 0o700 });
  if (observed.claude !== undefined) await writeFile(join(bin, "claude"), `#!/bin/sh\nprintf '${observed.claude} (Claude Code)\\n'\n`, { mode: 0o700 });
  return { root, execFile(command, _args, _options, callback) {
    const output = command === "which" ? `${join(bin, "opencode")}\n`
      : command === "opencode" ? `${opencodeOutput}\n`
      : observed.claude === undefined ? "" : `claude ${observed.claude}\n`;
    callback(output ? null : new Error("unavailable"), output, "");
  } };
}

test("resolves OpenCode core from the standalone version command", async (t) => {
  const boundary = await versionBoundary(t, {
    ...fixture.versions,
    rootPackage: { dependencies: { "@opencode-ai/plugin": "1.17.9" } },
  });
  assert.deepEqual(await resolveVersions(boundary), fixture.versions);
});

test("falls back to identified core package metadata when the command is unavailable", async (t) => {
  const boundary = await versionBoundary(t, { ...fixture.versions, opencodeOutput: "" });
  assert.equal((await resolveVersions(boundary)).opencode, fixture.versions.opencode);
});

for (const [scenario, opencodeOutput] of Object.entries({ unparseable: "OpenCode current", ambiguous: "1.18.19 1.18.20", multiline: "1.18.19\n1.18.20" })) {
  test(`rejects ${scenario} OpenCode version output`, async (t) => {
    const boundary = await versionBoundary(t, {
      ...fixture.versions, opencodeOutput,
      rootPackage: { dependencies: { "@opencode-ai/plugin": "1.17.9" } },
    });
    assert.equal((await resolveVersions(boundary)).opencode, null);
  });
}

test("allows only Linux ACM profiles on the pinned compatibility matrix", () => {
  assert.doesNotThrow(() => assertCompatibility("linux", true, fixture.versions));
  assert.throws(() => assertCompatibility("darwin", true, fixture.versions), /unsupported/);
  assert.throws(() => assertCompatibility("linux", false, fixture.versions), /ACM-managed/);
  assert.throws(() => assertCompatibility("linux", true, { ...fixture.versions, sdk: "2.0.0" }), /unsupported/);
});

for (const [key, value] of Object.entries({ opencode: "9.9.9", sdk: "invalid", claude: "9.9.9" })) {
  test(`production gate rejects a missing observed ${key} version`, async (t) => {
    const observed = { ...fixture.versions };
    delete observed[key];
    await assert.rejects(createPlugin({ platform: "linux", versionIO: await versionBoundary(t, observed) })(), /unsupported/);
  });
  test(`production gate rejects a mismatched observed ${key} version`, async (t) => {
    const observed = { ...fixture.versions, [key]: value };
    await assert.rejects(createPlugin({ platform: "linux", versionIO: await versionBoundary(t, observed) })(), /unsupported/);
  });
}

test("checker rejects genuine observed disagreement and accepts after restore", async (t) => {
  const boundary = await versionBoundary(t, fixture.versions);
  const script = fileURLToPath(new URL("../scripts/check-compat.js", import.meta.url));
  const env = { ...process.env, PATH: `${join(boundary.root, "bin")}:${process.env.PATH}` };
  const run = () => spawnSync(process.execPath, [script], { encoding: "utf8", env });
  const accepted = run();
  assert.equal(accepted.status, 0, accepted.stderr);
  await writeFile(join(boundary.root, "bin", "opencode"), "#!/bin/sh\nprintf '9.9.9\\n'\n", { mode: 0o700 });
  const rejected = run();
  assert.equal(rejected.status, 1);
  assert.equal(rejected.stderr, "OpenCode compatibility check failed\n");
  await writeFile(join(boundary.root, "bin", "opencode"), `#!/bin/sh\nprintf '${fixture.versions.opencode}\\n'\n`, { mode: 0o700 });
  const restored = run();
  assert.equal(restored.status, 0, restored.stderr);
  t.diagnostic(`accept=${accepted.stdout.trim()} reject=${rejected.stderr.trim()} restore=${restored.stdout.trim()}`);
});

test("hashes stable OpenCode session and message identity", () => {
  assert.equal(operationId(fixture.hash.session, fixture.hash.message), fixture.hash.expected);
});

test("applies the pinned Claude request transform", () => {
  const output = transformRequest(fixture.request.input);
  assert.equal(output.system[0].text, fixture.request.system);
  assert.equal(output.tools[0].name, fixture.request.tool);
});

test("refreshes through ACM begin and commit without an auth-file write", async () => {
  const calls = [];
  const result = await refreshCredentials({ profile: "alpha", generation: 7 }, "b".repeat(64), fixture.credentials.source.claudeAiOauth, {
    machine: async (operation, body) => (calls.push([operation, body]), operation.endsWith("begin") ? { lease_id: "lease" } : { outcome: "committed", generation: 8 }),
    send: async () => new Response(JSON.stringify(fixture.credentials.response), { status: 200 }),
    now: () => 1000,
  });
  assert.equal(result.access, "new-access");
  assert.equal(result.generation, 8);
  assert.deepEqual(calls.map(([operation]) => operation), ["oauth.refresh.begin", "oauth.refresh.commit"]);
});

test("returns metadata for one OpenCode-owned attempt without replay or continuation hooks", async (t) => {
  let sends = 0;
  const plugin = await createPlugin({
    platform: "linux", versionIO: await versionBoundary(t, fixture.versions),
    machine: async () => ({ ok: true, profile: "alpha", config_dir: "/synthetic", generation: 1 }),
    read: async () => JSON.stringify({ claudeAiOauth: { accessToken: "access", refreshToken: "refresh", expiresAt: 9999999999999 } }),
    send: async (request) => (sends += 1, new Response(request.headers.get("x-acm-operation-id") ?? "clean", { status: 429 })),
  })({ client: { auth: { set: () => assert.fail("plugin must not write OpenCode auth") } } });
  const output = { headers: {} };
  await plugin["chat.headers"]({ sessionID: fixture.hash.session, message: { id: fixture.hash.message } }, output);
  const loaded = await plugin.auth.loader(async () => ({ type: "oauth" }), { models: {} });
  const response = await loaded.fetch(new Request("https://example.invalid", { headers: output.headers }));
  assert.equal(await response.text(), "clean");
  assert.equal(sends, 1);
  assert.deepEqual(Object.keys(plugin).sort(), ["auth", "chat.headers"]);
});
