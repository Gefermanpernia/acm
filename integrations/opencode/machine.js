import { execFile } from "node:child_process";

const operations = new Set([
  "credential.select",
  "diagnostics.status",
  "oauth.refresh.begin",
  "oauth.refresh.commit",
  "oauth.refresh.abort",
  "quota.exhaust",
]);
const maxOutput = 16 << 10;

function failure(code, metadata = {}) {
  return Object.assign(new Error(`ACM machine interface failed (${code})`), { code, machine: true, ...metadata });
}

export function runMachine(operation, fields, options = {}) {
  if (!operations.has(operation)) return Promise.reject(failure("invalid_operation"));
  const request = { schema_version: 1, operation, ...fields };
  return new Promise((resolve, reject) => {
    const child = execFile(options.binary ?? process.env.ACM_BIN ?? "acm", [
      ...(options.prefixArgs ?? []), "machine", "v1", operation,
    ], {
      encoding: "utf8", maxBuffer: maxOutput, timeout: options.timeout ?? 5000,
      windowsHide: true, env: options.env,
    }, (error, stdout = "", stderr = "") => {
      if (error?.code === "ERR_CHILD_PROCESS_STDIO_MAXBUFFER") return reject(failure("invalid_machine_response"));
      if (error?.killed) return reject(failure("machine_timeout"));
      if (stderr || Buffer.byteLength(stdout) > maxOutput) return reject(failure("invalid_machine_response"));
      let response;
      try { response = JSON.parse(stdout); } catch { return reject(failure("invalid_machine_response")); }
      if (response?.schema_version !== 1 || response.operation !== operation ||
          response.operation_id !== request.operation_id || typeof response.ok !== "boolean") {
        return reject(failure("invalid_machine_response"));
      }
      if (error && response.ok) return reject(failure("machine_failed"));
      if (!response.ok) {
        const detail = response.error;
        if (typeof detail?.code !== "string" || typeof detail.message !== "string" ||
            typeof detail.retryable !== "boolean" ||
            (response.reset_at !== undefined && !Number.isInteger(response.reset_at))) {
          return reject(failure("invalid_machine_response"));
        }
        return reject(failure(detail.code, { retryable: detail.retryable, reset_at: response.reset_at }));
      }
      resolve(response);
    });
    child.stdin.end(JSON.stringify(request));
  });
}
