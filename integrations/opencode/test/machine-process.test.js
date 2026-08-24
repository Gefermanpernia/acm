import assert from "node:assert/strict";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { runMachine } from "../machine.js";

const stub = fileURLToPath(new URL("./fixtures/machine-stub.js", import.meta.url));
const id = "a".repeat(64);
// Only the dedicated timeout mode may race the clock. Every other mode spawns a
// Node interpreter whose startup can exceed 200ms under load, so those runs use
// a generous timeout to keep their failure classification independent of time.
const invoke = (mode = "ok") => runMachine("diagnostics.status", { operation_id: id }, {
  binary: process.execPath,
  prefixArgs: [stub],
  timeout: mode === "timeout" ? 200 : 5000,
  env: { ...process.env, ACM_STUB_MODE: mode },
});

test("executes only the fixed machine v1 operation", async () => {
  const result = await invoke();
  assert.equal(result.operation, "diagnostics.status");
  assert.deepEqual(result.args, ["machine", "v1", "diagnostics.status"]);
  await assert.rejects(() => runMachine("shell", { operation_id: id }), { code: "invalid_operation" });
});

for (const [name, mode, code] of [
  ["timeout", "timeout", "machine_timeout"],
  ["malformed stdout", "malformed", "invalid_machine_response"],
  ["oversized stdout", "oversized", "invalid_machine_response"],
  ["nonempty stderr", "stderr", "invalid_machine_response"],
  ["unexpected exit", "exit", "machine_failed"],
  ["schema rejection", "schema", "invalid_machine_response"],
]) test(`rejects ${name} without exposing process output`, async () => {
  await assert.rejects(invoke(mode), (error) => error.code === code && !error.message.includes("unsafe"));
});
