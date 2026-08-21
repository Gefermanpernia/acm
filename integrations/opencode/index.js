import { randomUUID } from "node:crypto";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { assertCompatibility, operationId, transformRequest } from "./compat.js";
import { runMachine } from "./machine.js";
import { refreshCredentials } from "./oauth.js";
import { handleQuotaResponse } from "./quota.js";
import versions from "./compatibility.json" with { type: "json" };

export function createPlugin(overrides = {}) {
  const deps = { platform: process.platform, versions, machine: runMachine, read: readFile, send: fetch, now: Date.now, ...overrides };
  return async function AcmOpenCodePlugin() {
    async function credentials(selection, id) {
      assertCompatibility(deps.platform, Boolean(selection?.profile && selection?.config_dir), deps.versions);
      const document = JSON.parse(await deps.read(join(selection.config_dir, ".credentials.json"), "utf8"));
      const source = document?.claudeAiOauth;
      if (typeof source?.accessToken !== "string" || typeof source?.refreshToken !== "string" || typeof source?.expiresAt !== "number") {
        throw new Error("ACM Claude credentials are invalid");
      }
      if (source.expiresAt >= deps.now() + 300000) return { access: source.accessToken, refresh: source.refreshToken, expires: source.expiresAt };
      return refreshCredentials(selection, id, source, deps);
    }
    async function load(id) {
      const selection = await deps.machine("credential.select", { operation_id: id });
      return [selection, await credentials(selection, id)];
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
            const [selection, auth] = await load(id);
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
