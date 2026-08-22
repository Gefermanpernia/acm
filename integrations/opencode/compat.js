import { createHash } from "node:crypto";
import { execFile } from "node:child_process";

const identity = "You are a Claude agent, built on Anthropic's Claude Agent SDK.";
const maxEvidence = 16 << 10;

function runVersionCommand(command, args, execute) {
  return new Promise((resolve) => {
    try {
      execute(command, args, { encoding: "utf8", maxBuffer: maxEvidence, timeout: 5000, windowsHide: true },
        (error, stdout = "", stderr = "") => resolve(!error && !stderr && Buffer.byteLength(stdout) <= maxEvidence ? stdout.trim() : ""));
    } catch { resolve(""); }
  });
}

function parseVersion(output) {
  if (!output || output.includes("\n")) return null;
  const matches = output.match(/\b\d+\.\d+\.\d+\b/g);
  return matches?.length === 1 ? matches[0] : null;
}

export async function detectClaudeVersion(boundary = {}) {
  const execute = boundary.execFile ?? execFile;
  return parseVersion(await runVersionCommand("claude", ["--version"], execute));
}

export function assertCompatibility(platform, managed) {
  if (platform !== "linux") throw new Error("unsupported platform");
  if (!managed) throw new Error("profile is not ACM-managed");
}

export function operationId(session, message) {
  if (!session || !message) throw new Error("OpenCode session identity is required");
  return createHash("sha256").update(`${session}\0${message}`).digest("hex");
}

export function transformRequest(value) {
  const body = structuredClone(value);
  const system = Array.isArray(body.system) ? body.system : body.system ? [{ type: "text", text: body.system }] : [];
  if (system[0]?.text !== identity) system.unshift({ type: "text", text: identity });
  body.system = system;
  body.tools = body.tools?.map((tool) => typeof tool?.name === "string" && !tool.name.startsWith("mcp_")
    ? { ...tool, name: `mcp_${tool.name[0].toUpperCase()}${tool.name.slice(1)}` } : tool);
  return body;
}
