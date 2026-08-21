import { createHash } from "node:crypto";
import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import { dirname, isAbsolute, join } from "node:path";
import matrix from "./compatibility.json" with { type: "json" };

const identity = "You are a Claude agent, built on Anthropic's Claude Agent SDK.";
const versionPattern = /^\d+\.\d+\.\d+$/;
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
  return matches?.length === 1 && versionPattern.test(matches[0]) ? matches[0] : null;
}

async function packageVersion(path, name) {
  try {
    const source = await readFile(path, "utf8");
    if (Buffer.byteLength(source) > maxEvidence) return null;
    const metadata = JSON.parse(source);
    return metadata?.name === name && versionPattern.test(metadata.version) ? metadata.version : null;
  } catch { return null; }
}

export async function resolveVersions(boundary = {}) {
  const execute = boundary.execFile ?? execFile;
  const executable = await runVersionCommand("which", ["opencode"], execute);
  const root = isAbsolute(executable) && !executable.includes("\n") ? dirname(dirname(executable)) : null;
  const opencodeOutput = await runVersionCommand("opencode", ["--version"], execute);
  const claudeOutput = await runVersionCommand("claude", ["--version"], execute);
  const opencodeCommand = versionPattern.test(opencodeOutput) ? opencodeOutput : null;
  const opencodePackage = root ? await packageVersion(join(root, "package.json"), "opencode-ai") : null;
  return {
    opencode: opencodeCommand && opencodePackage && opencodeCommand !== opencodePackage ? null : opencodeCommand ?? opencodePackage,
    sdk: root ? await packageVersion(join(root, "node_modules", "@opencode-ai", "sdk", "package.json"), "@opencode-ai/sdk") : null,
    claude: parseVersion(claudeOutput),
  };
}

export function assertCompatibility(platform, managed, versions) {
  if (platform !== "linux") throw new Error("unsupported platform");
  if (!managed) throw new Error("profile is not ACM-managed");
  if (Object.keys(matrix).some((key) => versions?.[key] !== matrix[key])) {
    throw new Error("unsupported OpenCode compatibility matrix");
  }
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
