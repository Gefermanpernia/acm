import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { access, mkdir, mkdtemp, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";

const repository = fileURLToPath(new URL("../../..", import.meta.url));
const installer = join(repository, "install.sh");
const runtimeAssets = [
  "compat.js", "diagnostics.js", "index.js", "machine.js",
  "oauth.js", "package.json", "quota.js",
];

async function createHostCanaries(root) {
  const home = join(root, "host-home");
  const customShare = join(root, "host-custom-share");
  const paths = [
    join(home, ".bashrc"),
    join(home, ".zshrc"),
    join(home, ".acm", "profiles", "claude", "real", ".credentials.json"),
    join(home, ".config", "opencode", "opencode.json"),
    join(home, ".local", "bin", "acm"),
    join(home, ".local", "share", "acm", "opencode", "index.js"),
    join(customShare, "opencode", "index.js"),
  ];
  await Promise.all(paths.map(async (path) => {
    await mkdir(join(path, ".."), { recursive: true });
    await writeFile(path, `host-canary:${path}\n`);
  }));
  return { home, customShare, paths, bytes: await Promise.all(paths.map((path) => readFile(path))) };
}

async function createOfflineCommands(root) {
  const bin = join(root, "fixture-bin");
  const log = join(root, "fixture.log");
  await mkdir(bin, { recursive: true });
  await writeFile(join(bin, "curl"), `#!/bin/sh
set -eu
url= output=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) output=$2; shift 2 ;;
    -*) shift ;;
    *) url=$1; shift ;;
  esac
done
printf 'curl|%s|%s\\n' "$url" "$output" >> "$ACM_FIXTURE_LOG"
case "$url" in
  */releases/*) cp "$ACM_FAKE_ACM" "$output" ;;
  */integrations/opencode/*)
    asset=\${url##*/}
    if [ "$asset" = "\${ACM_FAIL_ASSET:-}" ]; then
      printf 'fixture: rejected staged asset %s\\n' "$asset" >&2
      exit 22
    fi
    source="$ACM_FIXTURE_REPOSITORY/integrations/opencode/$asset"
    if [ ! -f "$source" ]; then
      printf 'fixture: missing shipped asset %s\\n' "$asset" >&2
      exit 22
    fi
    cp "$source" "$output" ;;
  *) printf 'fixture: unexpected URL %s\\n' "$url" >&2; exit 64 ;;
esac
`, { mode: 0o755 });
  return { bin, log };
}

async function buildRealACM(root) {
  const buildHome = join(root, "build-home");
  const buildTmp = join(root, "build-tmp");
  const goCache = join(root, "go-cache");
  const goModCache = join(root, "go-mod-cache");
  const binary = join(root, "release", "acm");
  await Promise.all([
    buildHome, buildTmp, goCache, goModCache, join(root, "release"),
  ].map((path) => mkdir(path, { recursive: true })));
  const built = spawnSync("go", ["build", "-o", binary, "."], {
    cwd: repository,
    env: {
      HOME: buildHome,
      PATH: process.env.PATH ?? "/usr/bin:/bin",
      TMPDIR: buildTmp,
      GOCACHE: goCache,
      GOMODCACHE: goModCache,
      GOPROXY: "off",
      GOSUMDB: "off",
      GOENV: "off",
      CGO_ENABLED: "0",
    },
    encoding: "utf8",
  });
  assert.equal(built.status, 0, built.stderr);
  return binary;
}

test("installs, enables, loads, and rolls back the custom-share OpenCode runtime without touching host state", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "acm-install-"));
  const sandboxHome = join(root, "sandbox-home");
  const installBin = join(root, "installed-bin");
  const share = join(root, "installed-share");
  const configHome = join(root, "opencode-config");
  const acmDir = join(root, "acm-state");
  const temporary = join(root, "tmp");
  const host = await createHostCanaries(root);
  const commands = await createOfflineCommands(root);
  await Promise.all([
    sandboxHome, installBin, share, configHome, acmDir, temporary,
  ].map((path) => mkdir(path, { recursive: true })));
  const releaseBinary = await buildRealACM(root);
  const configPath = join(configHome, "opencode.json");
  const originalConfig = Buffer.from('{"model":"anthropic/claude"}\n');
  await writeFile(configPath, originalConfig, { mode: 0o600 });
  t.after(async () => {
    assert.deepEqual(await Promise.all(host.paths.map((path) => readFile(path))), host.bytes,
      "host aliases, credentials, configuration, and real install targets changed");
    const log = await readFile(commands.log, "utf8");
    assert.doesNotMatch(log, new RegExp(host.home.replaceAll("/", "\\/")));
    assert.doesNotMatch(log, new RegExp(host.customShare.replaceAll("/", "\\/")));
    await rm(root, { recursive: true, force: true });
    await assert.rejects(access(root), { code: "ENOENT" });
    t.diagnostic("host_canaries=unchanged sandbox_removed=true");
  });
  const env = {
    HOME: sandboxHome,
    PATH: `${installBin}:${commands.bin}:/usr/bin:/bin`,
    TMPDIR: temporary,
    GOCACHE: join(root, "runtime-go-cache"),
    ACM_VERSION: "v-fixture",
    ACM_DIR: acmDir,
    ACM_BIN_DIR: installBin,
    ACM_SHARE_DIR: share,
    ACM_OPENCODE_CONFIG_HOME: configHome,
    ACM_FIXTURE_LOG: commands.log,
    ACM_FAKE_ACM: releaseBinary,
    ACM_FIXTURE_REPOSITORY: repository,
  };
  const run = (extra = {}) => spawnSync("sh", [installer], {
    cwd: root, env: { ...env, ...extra }, encoding: "utf8",
  });

  const installed = run();
  assert.equal(installed.status, 0, installed.stderr);
  const pluginDir = join(share, "opencode");
  assert.deepEqual((await readdir(pluginDir)).sort(), runtimeAssets);
  const enabled = spawnSync("acm", ["opencode", "enable", "--confirm"], {
    cwd: root, env, encoding: "utf8",
  });
  assert.equal(enabled.status, 0, enabled.stderr);
  const enabledConfig = JSON.parse(await readFile(configPath, "utf8"));
  assert.deepEqual(enabledConfig.plugin, [pathToFileURL(join(pluginDir, "index.js")).href]);
  const module = await import(`${enabledConfig.plugin[0]}?fixture=${Date.now()}`);
  const hooks = await module.createPlugin({
    platform: "linux",
    diagnostic: () => {},
    versionIO: { execFile: (_command, _args, _options, done) => done(null, "fixture-cli\n", "") },
  })();
  assert.deepEqual(Object.keys(hooks).sort(), ["auth", "chat.headers"]);
  const rolledBack = spawnSync("acm", ["opencode", "rollback", "--confirm"], {
    cwd: root, env, encoding: "utf8",
  });
  assert.equal(rolledBack.status, 0, rolledBack.stderr);
  assert.deepEqual(await readFile(configPath), originalConfig);
  await assert.rejects(access(`${configPath}.acm-backup`), { code: "ENOENT" });
  await assert.rejects(access(join(configHome, ".acm-opencode-backup.json")), { code: "ENOENT" });
  t.diagnostic(`install_exit=${installed.status}`);
  t.diagnostic(`enable_exit=${enabled.status}`);
  t.diagnostic(`enabled_plugin=${enabledConfig.plugin[0]}`);
  t.diagnostic(`loaded_hooks=${Object.keys(hooks).sort().join(",")}`);
  t.diagnostic(`rollback_exit=${rolledBack.status} restored_bytes=${originalConfig.length}`);

  const defaultPlugin = join(sandboxHome, ".local", "share", "acm", "opencode", "index.js");
  await mkdir(join(defaultPlugin, ".."), { recursive: true });
  await writeFile(defaultPlugin, "export default {}\n", { mode: 0o600 });
  const missingShare = join(root, "missing-custom-share");
  const missingCustom = spawnSync("acm", ["opencode", "enable", "--confirm"], {
    cwd: root, env: { ...env, ACM_SHARE_DIR: missingShare }, encoding: "utf8",
  });
  assert.equal(missingCustom.status, 2, missingCustom.stderr);
  assert.match(missingCustom.stderr, /adaptador OpenCode de ACM no está instalado/);
  assert.deepEqual(await readFile(configPath), originalConfig);
  await assert.rejects(access(`${configPath}.acm-backup`), { code: "ENOENT" });
  await assert.rejects(access(join(configHome, ".acm-opencode-backup.json")), { code: "ENOENT" });
  t.diagnostic("missing_custom_share_exit=2 default_fallback=false config_unchanged=true");

  const bundle = await Promise.all(runtimeAssets.map((asset) => readFile(join(pluginDir, asset))));

  const rejected = run({ ACM_FAIL_ASSET: "quota.js" });
  assert.equal(rejected.status, 22);
  assert.match(rejected.stderr, /rejected staged asset quota\.js/);
  assert.deepEqual(await Promise.all(runtimeAssets.map((asset) => readFile(join(pluginDir, asset)))), bundle);
  assert.deepEqual(await readdir(share), ["opencode"]);
  await assert.rejects(access(join(sandboxHome, ".acm")), { code: "ENOENT" });
});
