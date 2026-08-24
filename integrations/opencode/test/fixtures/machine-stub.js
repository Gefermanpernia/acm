#!/usr/bin/env node
const mode = process.env.ACM_STUB_MODE;
let input = "";
process.stdin.on("data", (chunk) => (input += chunk));
process.stdin.on("end", () => {
  const operation = process.argv.at(-1);
  const request = JSON.parse(input);
  const reply = { schema_version: 1, ok: true, operation, operation_id: request.operation_id,
    args: process.argv.slice(-3) };
  if (mode === "timeout") return setTimeout(() => {}, 1000);
  if (mode === "malformed") return process.stdout.write("not-json");
  if (mode === "oversized") return process.stdout.write("x".repeat(17000));
  if (mode === "stderr") process.stderr.write("unsafe diagnostic");
  if (mode === "schema") reply.schema_version = 2;
  process.stdout.write(JSON.stringify(reply));
  if (mode === "exit") process.exitCode = 3;
});
