import { randomUUID } from "node:crypto";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { assertCompatibility, detectClaudeVersion, operationId, transformRequest } from "./compat.js";
import { runMachine } from "./machine.js";
import { refreshCredentials } from "./oauth.js";
import { handleQuotaResponse, mapMachineResponse } from "./quota.js";

function safeCredentialError() {
  return new Error("ACM Claude credentials are unavailable");
}

export function createPlugin(overrides = {}) {
  const deps = { platform: process.platform, machine: runMachine, read: readFile, send: fetch, now: Date.now, ...overrides };
  deps.diagnosticError ??= (code) => console.error(`ACM diagnostics: ${code}`);
  deps.diagnostic ??= async ({ component, event, outcome, retryable }) => {
    try {
      await deps.machine("diagnostics.record", { operation_id: operationId("diagnostics", randomUUID()), component, event, outcome, retryable });
    } catch {
      deps.diagnosticError("record_failed");
    }
  };
  return async function AcmOpenCodePlugin() {
    assertCompatibility(deps.platform, true);
    const version = await detectClaudeVersion(deps.versionIO);
    await Promise.resolve(deps.diagnostic({ component: "adapter", event: "compatibility", outcome: version ? "recovered" : "unavailable", retryable: false, version })).catch(() => deps.diagnosticError("record_failed"));
    async function credentials(selection, id) {
      assertCompatibility(deps.platform, Boolean(selection?.profile && selection?.config_dir));
      let document;
      try {
        document = JSON.parse(await deps.read(join(selection.config_dir, ".credentials.json"), "utf8"));
      } catch {
        throw safeCredentialError();
      }
      const source = document?.claudeAiOauth;
      if (typeof source?.accessToken !== "string" || typeof source?.refreshToken !== "string" || typeof source?.expiresAt !== "number") {
        throw new Error("ACM Claude credentials are invalid");
      }
      if (source.expiresAt >= deps.now() + 300000) return { access: source.accessToken, refresh: source.refreshToken, expires: source.expiresAt };
      return refreshCredentials(selection, id, source, deps);
    }
    async function load(id) {
      const selection = await deps.machine("credential.select", { operation_id: id });
      const loaded = await credentials(selection, id);
      const { generation = selection.generation, ...auth } = loaded;
      return [{ ...selection, generation }, auth];
    }
    return {
      "chat.headers": async (input, output) => {
        output.headers["x-acm-operation-id"] = operationId(input.sessionID, input.message?.id);
      },
      auth: {
        provider: "anthropic",
        loader: async (getAuth) => (await getAuth())?.type !== "oauth" ? {} : {
          apiKey: "",
          fetch: async (input, init) => {
            const request = new Request(input, init);
            const id = request.headers.get("x-acm-operation-id");
            if (!id) throw new Error("OpenCode operation identity is missing");
            let selection, auth;
            try {
              [selection, auth] = await load(id);
            } catch (error) {
              if (error?.machine) return mapMachineResponse(error, deps.now);
              throw error;
            }
            const headers = new Headers(request.headers);
            headers.delete("x-acm-operation-id");
            headers.delete("x-api-key");
            headers.set("authorization", `Bearer ${auth.access}`);
            let body;
            if (request.body && headers.get("content-type")?.includes("application/json")) body = JSON.stringify(transformRequest(await request.clone().json()));
            const providerResponse = await deps.send(new Request(
              request,
              body === undefined ? { headers } : { headers, body },
            ));
            return handleQuotaResponse(providerResponse, {
              operationID: id,
              selection,
            }, deps);
          },
        },
        methods: [{ type: "oauth", label: "ACM Claude profile", authorize: async () => ({
          url: "https://claude.ai", instructions: "Load the selected ACM profile.", method: "auto",
          callback: async () => {
            const [, auth] = await load(operationId("opencode-auth", randomUUID()));
            return { type: "success", ...auth };
          },
        }) }],
      },
    };
  };
}

export default createPlugin();
