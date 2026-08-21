import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { createPlugin } from "../index.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/quota.json", import.meta.url)));

test("returns cooling metadata after exactly one provider call", async () => {
  const operations = [];
  let providerCalls = 0;
  const plugin = await createPlugin({
    platform: "linux",
    versions: { opencode: "1.18.19", sdk: "1.17.12", claude: "2.1.236" },
    read: async () => JSON.stringify(fixture.credentials),
    machine: async (operation) => {
      operations.push(operation);
      return operation === "credential.select"
        ? fixture.selection
        : { outcome: "cooling", retry_after: 45 };
    },
    send: async () => {
      providerCalls += 1;
      return responseFor(fixture.confirmed);
    },
  })();
  const output = { headers: {} };
  await plugin["chat.headers"]({ sessionID: "session", message: { id: "message" } }, output);
  const auth = await plugin.auth.loader(async () => ({ type: "oauth" }));
  const result = await auth.fetch(new Request("https://example.invalid", { headers: output.headers }));

  assert.equal(providerCalls, 1);
  assert.deepEqual(operations, ["credential.select", "quota.exhaust"]);
  assert.equal(result.status, 429);
  assert.equal(result.headers.get("retry-after"), "45");
});

function responseFor(value) {
  return new Response(JSON.stringify(value.body), {
    status: value.status,
    headers: value.headers,
  });
}
