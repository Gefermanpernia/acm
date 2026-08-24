import { assertCompatibility } from "../compat.js";
import versions from "../compatibility.json" with { type: "json" };

assertCompatibility("linux", true, versions);
process.stdout.write(`OpenCode ${versions.opencode}, SDK ${versions.sdk}, Claude CLI ${versions.claude}\n`);
