import { assertCompatibility, resolveVersions } from "../compat.js";

try {
  const observed = await resolveVersions();
  assertCompatibility(process.platform, true, observed);
  process.stdout.write(`OpenCode ${observed.opencode}, SDK ${observed.sdk}, Claude CLI ${observed.claude}\n`);
} catch {
  process.stderr.write("OpenCode compatibility check failed\n");
  process.exitCode = 1;
}
