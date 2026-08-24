package main

import (
	"bytes"
	"os"
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
	code, output := runOpenCodeTest(t, "enable", "--confirm")
	changed, _ := os.ReadFile(path)
	check(t, code == 0 && strings.Contains(output, "Reinicia OpenCode") && strings.Contains(string(changed), "// keep this comment"), "migration failed: %s", output)
	check(t, strings.Contains(string(changed), "index.js") && !strings.Contains(string(changed), "opencode-anthropic-login-via-cli"), "plugins are not exclusive: %s", changed)
	backup := filepath.Join(home, "opencode.jsonc.acm-backup")
	check(t, os.WriteFile(backup, []byte("corrupt"), 0o600) == nil, "corrupt backup")
	code, _ = runOpenCodeTest(t, "rollback", "--confirm")
	after, _ := os.ReadFile(path)
	check(t, code != 0 && bytes.Equal(after, changed), "invalid checksum changed config")
}

func TestOpenCodeMigrationRollbackAndMissingBackup(t *testing.T) {
	original := `{"plugin":["opencode-anthropic-login-via-cli@1.6.1"]}`
	_, path := setupOpenCodeLifecycle(t, "opencode.json", original)
	code, _ := runOpenCodeTest(t, "enable")
	check(t, code != 0, "enable accepted without opt-in")
	code, _ = runOpenCodeTest(t, "enable", "--confirm")
	check(t, code == 0, "enable failed")
	code, output := runOpenCodeTest(t, "rollback", "--confirm")
	restored, _ := os.ReadFile(path)
	check(t, code == 0 && string(restored) == original && strings.Contains(output, "Reinicia OpenCode"), "rollback did not restore config: %s", output)
	_, missingPath := setupOpenCodeLifecycle(t, "opencode.json", original)
	before, _ := os.ReadFile(missingPath)
	code, _ = runOpenCodeTest(t, "rollback", "--confirm")
	after, _ := os.ReadFile(missingPath)
	check(t, code != 0 && bytes.Equal(after, before), "missing backup changed config")
}
