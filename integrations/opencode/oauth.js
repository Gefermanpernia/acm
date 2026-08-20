const tokenURL = "https://platform.claude.com/v1/oauth/token";
const clientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e";

export async function refreshCredentials(selection, operationID, source, dependencies) {
  const { machine, send, now = Date.now } = dependencies;
  const common = { operation_id: operationID, profile: selection.profile, generation: selection.generation };
  const lease = await machine("oauth.refresh.begin", common);
  let response;
  try {
    response = await send(tokenURL, {
      method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ grant_type: "refresh_token", refresh_token: source.refreshToken, client_id: clientID }),
    });
    const body = await response.json();
    if (!response.ok || typeof body.access_token !== "string" || typeof body.refresh_token !== "string" || typeof body.expires_in !== "number") {
      const reason = ["invalid_grant", "revoked"].includes(body?.error) ? body.error : "transient";
      await machine("oauth.refresh.abort", { ...common, lease_id: lease.lease_id, reason });
      throw new Error("Claude OAuth refresh failed");
    }
    const credentials = { access: body.access_token, refresh: body.refresh_token, expires: now() + body.expires_in * 1000 };
    await machine("oauth.refresh.commit", {
      ...common, lease_id: lease.lease_id, access_token: credentials.access,
      refresh_token: credentials.refresh, expires_at: credentials.expires,
    });
    return credentials;
  } catch (error) {
    if (!response) await machine("oauth.refresh.abort", { ...common, lease_id: lease.lease_id, reason: "transient" });
    throw error;
  }
}
