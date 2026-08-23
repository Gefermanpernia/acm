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
  const paths = [
    join(home, ".bashrc"),
    join(home, ".zshrc"),
    join(home, ".acm", "profiles", "claude", "real", ".credentials.json"),
    join(home, ".config", "opencode", "opencode.json"),
    join(home, ".local", "bin", "acm"),
    join(home, ".local", "share", "acm", "opencode", "index.js"),
  ];
  await Promise.all(paths.map(async (path) => {
    await mkdir(join(path, ".."), { recursive: true });
    await writeFile(path, `host-canary:${path}\n`);
  }));
  return { home, paths, bytes: await Promise.all(paths.map((path) => readFile(path))) };
}

async function createOfflineCommands(root) {
  const bin = join(root, "fixture-bin");
  const log = join(root, "fixture.log");
  const acm = join(bin, "acm");
  await mkdir(bin, { recursive: true });
  await writeFile(acm, `#!/bin/sh
set -eu
printf 'acm|%s|%s|%s|%s\\n' "$HOME" "$ACM_BIN_DIR" "$ACM_SHARE_DIR" "$*" >> "$ACM_FIXTURE_LOG"
case "\${1:-}" in
  version) printf 'acm fixture\\n' ;;
  init) exit 0 ;;
  *) exit 64 ;;
esac
`, { mode: 0o755 });
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
  return { bin, log, acm };
}

test("installs the complete OpenCode runtime atomically without touching host state", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "acm-install-"));
  const sandboxHome = join(root, "sandbox-home");
  const installBin = join(root, "installed-bin");
  const share = join(root, "installed-share");
  const temporary = join(root, "tmp");
  const host = await createHostCanaries(root);
  const commands = await createOfflineCommands(root);
  await Promise.all([sandboxHome, installBin, share, temporary].map((path) => mkdir(path, { recursive: true })));
  t.after(async () => {
    assert.deepEqual(await Promise.all(host.paths.map((path) => readFile(path))), host.bytes,
      "host aliases, credentials, configuration, and real install targets changed");
    const log = await readFile(commands.log, "utf8");
    assert.doesNotMatch(log, new RegExp(host.home.replaceAll("/", "\\/")));
    assert.match(log, new RegExp(`acm\\|${sandboxHome.replaceAll("/", "\\/")}\\|${installBin.replaceAll("/", "\\/")}\\|${share.replaceAll("/", "\\/")}\\|init`));
    await rm(root, { recursive: true, force: true });
    await assert.rejects(access(root), { code: "ENOENT" });
  });
  const env = {
    ...process.env,
    HOME: sandboxHome,
    PATH: `${commands.bin}:/usr/bin:/bin`,
    TMPDIR: temporary,
    ACM_VERSION: "v-fixture",
    ACM_BIN_DIR: installBin,
    ACM_SHARE_DIR: share,
    ACM_FIXTURE_LOG: commands.log,
    ACM_FAKE_ACM: commands.acm,
    ACM_FIXTURE_REPOSITORY: repository,
  };
  const run = (extra = {}) => spawnSync("sh", [installer], {
    cwd: root, env: { ...env, ...extra }, encoding: "utf8",
  });

  const installed = run();
  assert.equal(installed.status, 0, installed.stderr);
  const pluginDir = join(share, "opencode");
  assert.deepEqual((await readdir(pluginDir)).sort(), runtimeAssets);
  const module = await import(`${pathToFileURL(join(pluginDir, "index.js")).href}?fixture=${Date.now()}`);
  const hooks = await module.createPlugin({
    platform: "linux",
    diagnostic: () => {},
    versionIO: { execFile: (_command, _args, _options, done) => done(null, "fixture-cli\n", "") },
  })();
  assert.deepEqual(Object.keys(hooks).sort(), ["auth", "chat.headers"]);
  const bundle = await Promise.all(runtimeAssets.map((asset) => readFile(join(pluginDir, asset))));

  const rejected = run({ ACM_FAIL_ASSET: "quota.js" });
  assert.equal(rejected.status, 22);
  assert.match(rejected.stderr, /rejected staged asset quota\.js/);
  assert.deepEqual(await Promise.all(runtimeAssets.map((asset) => readFile(join(pluginDir, asset)))), bundle);
  assert.deepEqual(await readdir(share), ["opencode"]);
  await assert.rejects(access(join(sandboxHome, ".acm")), { code: "ENOENT" });
});
