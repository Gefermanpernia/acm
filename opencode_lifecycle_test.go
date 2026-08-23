package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupOpenCodeLifecycle(t *testing.T, name, config string) (string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("ACM_OPENCODE_CONFIG_HOME", home)
	plugin := filepath.Join(t.TempDir(), "index.js")
	check(t, os.WriteFile(plugin, []byte("export default {}\n"), 0o600) == nil, "write plugin")
	t.Setenv("ACM_OPENCODE_PLUGIN_PATH", plugin)
	path := filepath.Join(home, name)
	check(t, os.WriteFile(path, []byte(config), 0o600) == nil, "write config")
	return home, path
}

func runOpenCodeTest(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var output bytes.Buffer
	code := runOpenCodeLifecycle(args, &output, &output)
	return code, output.String()
}

func TestOpenCodeMigrationRollbackOnJSONCConflict(t *testing.T) {
	home, jsonPath := setupOpenCodeLifecycle(t, "opencode.json", `{"plugin":["upstream-json"]}`)
	jsoncPath := filepath.Join(home, "opencode.jsonc")
	check(t, os.WriteFile(jsoncPath, []byte("// keep\n{\"plugin\":[\"upstream-jsonc\"]}\n"), 0o600) == nil, "write JSONC")
	jsonBefore, _ := os.ReadFile(jsonPath)
	jsoncBefore, _ := os.ReadFile(jsoncPath)
	code, _ := runOpenCodeTest(t, "enable", "--confirm")
	jsonAfter, _ := os.ReadFile(jsonPath)
	jsoncAfter, _ := os.ReadFile(jsoncPath)
	_, backupErr := os.Stat(filepath.Join(home, ".acm-opencode-backup.json"))
	check(t, code != 0 && bytes.Equal(jsonAfter, jsonBefore) && bytes.Equal(jsoncAfter, jsoncBefore) && os.IsNotExist(backupErr), "ambiguous origins were not preserved")
	check(t, os.Remove(jsonPath) == nil, "remove conflicting JSON origin")
	oldValidate, calls := validateOpenCode, 0
	validateOpenCode = func([]byte, string) bool { calls++; return calls == 1 }
	t.Cleanup(func() { validateOpenCode = oldValidate })
	code, _ = runOpenCodeTest(t, "enable", "--confirm")
	jsoncAfter, _ = os.ReadFile(jsoncPath)
	_, backupErr = os.Stat(filepath.Join(home, ".acm-opencode-backup.json"))
	check(t, code != 0 && bytes.Equal(jsoncAfter, jsoncBefore) && os.IsNotExist(backupErr), "post-write conflict did not restore every touched file")
}

func TestOpenCodeMigrationEnforcesExclusivityAndChecksum(t *testing.T) {
	home, path := setupOpenCodeLifecycle(t, "opencode.jsonc", "// keep this comment\n{\"plugin\":[\"opencode-anthropic-login-via-cli@1.6.1\",\"file:///tmp/acm/opencode/index.js\"],}\n")
	original, _ := os.ReadFile(path)
	code, _ := runOpenCodeTest(t, "enable", "--confirm")
	unchanged, _ := os.ReadFile(path)
	_, manifestErr := os.Stat(filepath.Join(home, openCodeManifest))
	_, backupErr := os.Stat(path + ".acm-backup")
	check(t, code != 0 && bytes.Equal(unchanged, original), "plugin conflict changed config or returned success")
	check(t, os.IsNotExist(manifestErr) && os.IsNotExist(backupErr), "plugin conflict created a backup")

	code, output := runOpenCodeTest(t, "enable", "--confirm", "--replace-upstream")
	changed, _ := os.ReadFile(path)
	check(t, code == 0 && strings.Contains(output, "Reinicia OpenCode") && strings.Contains(string(changed), "// keep this comment"), "migration failed: %s", output)
	check(t, strings.Contains(string(changed), "index.js") && !strings.Contains(string(changed), "opencode-anthropic-login-via-cli"), "plugins are not exclusive: %s", changed)
	backup := filepath.Join(home, "opencode.jsonc.acm-backup")
	check(t, os.WriteFile(backup, []byte("corrupt"), 0o600) == nil, "corrupt backup")
	code, _ = runOpenCodeTest(t, "rollback", "--confirm")
	after, _ := os.ReadFile(path)
	check(t, code != 0 && bytes.Equal(after, changed), "invalid checksum changed config")
}

func TestOpenCodeMigrationRealBinaryRequiresExplicitReplacement(t *testing.T) {
	home, path := setupOpenCodeLifecycle(t, "opencode.json", `{"plugin":["opencode-anthropic-login-via-cli@1.6.1","file:///tmp/acm/opencode/index.js"]}`)
	original, _ := os.ReadFile(path)
	runtimeRoot, buildRoot := t.TempDir(), t.TempDir()
	bin := filepath.Join(buildRoot, "acm")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "HOME="+runtimeRoot, "GOCACHE="+filepath.Join(runtimeRoot, "go-cache"), "TMPDIR="+runtimeRoot)
	buildOutput, err := build.CombinedOutput()
	check(t, err == nil, "build real binary: %v\n%s", err, buildOutput)
	run := func(args ...string) (int, string) {
		command := exec.Command(bin, append([]string{"opencode"}, args...)...)
		command.Env = append(os.Environ(), "HOME="+runtimeRoot, "ACM_DIR="+filepath.Join(runtimeRoot, ".acm"), "ACM_OPENCODE_CONFIG_HOME="+home, "TMPDIR="+runtimeRoot)
		output, err := command.CombinedOutput()
		if err == nil {
			return 0, string(output)
		}
		return err.(*exec.ExitError).ExitCode(), string(output)
	}
	code, output := run("enable", "--confirm")
	unchanged, _ := os.ReadFile(path)
	_, manifestErr := os.Stat(filepath.Join(home, openCodeManifest))
	_, backupErr := os.Stat(path + ".acm-backup")
	check(t, code == 2 && strings.Contains(output, "--replace-upstream") && bytes.Equal(unchanged, original), "real binary did not stop safely: exit=%d output=%s", code, output)
	check(t, os.IsNotExist(manifestErr) && os.IsNotExist(backupErr), "real binary created backup before replacement")
	code, output = run("enable", "--confirm", "--replace-upstream")
	changed, _ := os.ReadFile(path)
	_, manifestErr = os.Stat(filepath.Join(home, openCodeManifest))
	_, backupErr = os.Stat(path + ".acm-backup")
	check(t, code == 0 && strings.Contains(output, "Reinicia OpenCode") && strings.Contains(string(changed), "index.js") && !strings.Contains(string(changed), "opencode-anthropic-login-via-cli"), "real binary replacement failed: exit=%d output=%s", code, output)
	check(t, manifestErr == nil && backupErr == nil, "real binary replacement did not create rollback backup")
}

func TestOpenCodeMigrationRollbackAndMissingBackup(t *testing.T) {
	original := `{"plugin":["opencode-anthropic-login-via-cli@1.6.1"]}`
	home, path := setupOpenCodeLifecycle(t, "opencode.json", original)
	code, _ := runOpenCodeTest(t, "enable")
	check(t, code != 0, "enable accepted without opt-in")
	code, _ = runOpenCodeTest(t, "enable", "--confirm")
	unchanged, _ := os.ReadFile(path)
	_, manifestErr := os.Stat(filepath.Join(home, openCodeManifest))
	check(t, code != 0 && string(unchanged) == original && os.IsNotExist(manifestErr), "migration accepted without explicit replacement")
	code, _ = runOpenCodeTest(t, "enable", "--confirm", "--replace-upstream")
	check(t, code == 0, "confirmed replacement failed")
	code, output := runOpenCodeTest(t, "rollback", "--confirm")
	restored, _ := os.ReadFile(path)
	check(t, code == 0 && string(restored) == original && strings.Contains(output, "Reinicia OpenCode"), "rollback did not restore config: %s", output)
	_, missingPath := setupOpenCodeLifecycle(t, "opencode.json", original)
	before, _ := os.ReadFile(missingPath)
	code, _ = runOpenCodeTest(t, "rollback", "--confirm")
	after, _ := os.ReadFile(missingPath)
	check(t, code != 0 && bytes.Equal(after, before), "missing backup changed config")
}
