package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testMachineRequest = `{"schema_version":1,"operation":"credential.select","operation_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
const testCredential = `{"other":"kept","claudeAiOauth":{"accessToken":"old-access","refreshToken":"old-refresh","expiresAt":1}}`

type machineResponse struct {
	Profile    string `json:"profile"`
	ConfigDir  string `json:"config_dir"`
	Generation uint64 `json:"generation"`
	LeaseID    string `json:"lease_id"`
	Error      *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func check(t *testing.T, ok bool, format string, args ...any) {
	if !ok {
		t.Fatalf(format, args...)
	}
}

func setupMachineV1Test(t *testing.T, profiles ...string) *tool {
	oldHome, oldACM, oldProf, oldState, oldCool, oldTools := homeDir, acmDir, profDir, stateDir, coolDir, tools
	t.Cleanup(func() {
		homeDir, acmDir, profDir, stateDir, coolDir, tools = oldHome, oldACM, oldProf, oldState, oldCool, oldTools
	})
	homeDir = t.TempDir()
	acmDir, profDir = filepath.Join(homeDir, ".acm"), filepath.Join(homeDir, ".acm", "profiles")
	stateDir, coolDir = filepath.Join(acmDir, "state"), filepath.Join(acmDir, "state", "cooldown")
	claude := &tool{name: "claude", credFile: ".credentials.json", defaultHome: filepath.Join(homeDir, ".claude")}
	tools = map[string]*tool{"claude": claude}
	for _, name := range profiles {
		dir := profilePath(claude, name)
		check(t, os.MkdirAll(dir, 0o755) == nil, "create profile %s", name)
		check(t, os.WriteFile(filepath.Join(dir, claude.credFile), []byte(testCredential), 0o600) == nil, "create credential %s", name)
	}
	return claude
}

func oauthRequest(operation string, id byte, fields string) string {
	return fmt.Sprintf(`{"schema_version":1,"operation":%q,"operation_id":%q,%s}`, operation, strings.Repeat(string(id), 64), fields)
}

func beginOAuthRefresh(t *testing.T, id byte) (machineResponse, uint64) {
	code, selection, _ := invokeMachine(t, "credential.select", testMachineRequest)
	check(t, code == 0, "select response = %+v, exit = %d", selection, code)
	input := oauthRequest("oauth.refresh.begin", id, fmt.Sprintf(`"profile":"alpha","generation":%d`, selection.Generation))
	code, response, _ := invokeMachine(t, "oauth.refresh.begin", input)
	check(t, code == 0 && response.LeaseID != "", "begin response = %+v, exit = %d", response, code)
	return response, selection.Generation
}

func TestOAuthRefreshLeaseIsBusyThenReusableAfterExpiry(t *testing.T) {
	setupMachineV1Test(t, "alpha")
	first, generation := beginOAuthRefresh(t, 'b')
	code, busy, _ := invokeMachine(t, "oauth.refresh.begin", oauthRequest("oauth.refresh.begin", 'c', `"profile":"alpha","generation":1`))
	check(t, code == 75 && busy.Error != nil && busy.Error.Code == "lease_busy", "busy response = %+v, exit = %d", busy, code)

	state, err := loadMachineState()
	check(t, err == nil && len(state.Leases) == 1, "load lease state: %+v, %v", state, err)
	state.Leases[0].ExpiresAt = time.Now().Add(-time.Second).Unix()
	check(t, saveMachineState(state) == nil, "expire lease")
	input := oauthRequest("oauth.refresh.begin", 'd', fmt.Sprintf(`"profile":"alpha","generation":%d`, generation))
	code, second, _ := invokeMachine(t, "oauth.refresh.begin", input)
	check(t, code == 0, "second begin = %+v, exit = %d", second, code)
	check(t, second.LeaseID != first.LeaseID, "expired lease was reused")
}

func TestOAuthRefreshCommitFailsClosedOnStaleWriteAndFsync(t *testing.T) {
	tests := []struct {
		name    string
		breakIO func()
		code    string
	}{
		{"stale generation", func() {
			state, _ := loadMachineState()
			state.Generation++
			check(t, saveMachineState(state) == nil, "advance generation")
		}, "stale_generation"},
		{"expired lease", func() {
			state, _ := loadMachineState()
			state.Leases[0].ExpiresAt = time.Now().Add(-time.Second).Unix()
			check(t, saveMachineState(state) == nil, "expire lease")
		}, "lease_expired"},
		{"write failure", func() { machinePersist = func(string, []byte) error { return errors.New("secret write detail") } }, "persistence_failed"},
		{"fsync failure", func() { machineSync = func(*os.File) error { return errors.New("secret sync detail") } }, "persistence_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupMachineV1Test(t, "alpha")
			lease, generation := beginOAuthRefresh(t, 'b')
			oldPersist, oldSync := machinePersist, machineSync
			defer func() { machinePersist, machineSync = oldPersist, oldSync }()
			test.breakIO()
			input := oauthRequest("oauth.refresh.commit", 'c', fmt.Sprintf(`"profile":"alpha","generation":%d,"lease_id":%q,"access_token":"new-access","refresh_token":"new-refresh","expires_at":99`, generation, lease.LeaseID))
			code, response, raw := invokeMachine(t, "oauth.refresh.commit", input)
			check(t, code != 0 && response.Error != nil && response.Error.Code == test.code, "commit response = %+v, exit = %d", response, code)
			credential, _ := os.ReadFile(filepath.Join(profilePath(tools["claude"], "alpha"), ".credentials.json"))
			check(t, string(credential) == testCredential, "credential changed after failure: %s", credential)
			check(t, !strings.Contains(raw, "new-access") && !strings.Contains(raw, "new-refresh") && !strings.Contains(raw, "detail"), "secret leaked: %s", raw)
		})
	}
}

func TestOAuthRefreshCommitIsAtomic0600AndSecretless(t *testing.T) {
	setupMachineV1Test(t, "alpha")
	lease, generation := beginOAuthRefresh(t, 'b')
	input := oauthRequest("oauth.refresh.commit", 'c', fmt.Sprintf(`"profile":"alpha","generation":%d,"lease_id":%q,"access_token":"new-access","refresh_token":"new-refresh","expires_at":99`, generation, lease.LeaseID))
	code, response, raw := invokeMachine(t, "oauth.refresh.commit", input)
	check(t, code == 0 && response.Generation == generation+1 && strings.Contains(raw, `"outcome":"committed"`), "commit response = %+v, exit = %d", response, code)
	path := filepath.Join(profilePath(tools["claude"], "alpha"), ".credentials.json")
	credential, err := os.ReadFile(path)
	check(t, err == nil && strings.Contains(string(credential), `"other":"kept"`) && strings.Contains(string(credential), "new-refresh"), "credential = %s, err = %v", credential, err)
	info, _ := os.Stat(path)
	check(t, info.Mode().Perm() == 0o600, "credential mode = %o", info.Mode().Perm())
	temps, _ := filepath.Glob(path + ".tmp-*")
	state, _ := os.ReadFile(machineStateFile())
	check(t, len(temps) == 0 && !strings.Contains(raw+string(state), "new-access") && !strings.Contains(raw+string(state), "new-refresh"), "secret or temporary file escaped commit")
}

func TestOAuthRefreshAbortQuarantinesOnlyTerminalReasons(t *testing.T) {
	tests := []struct {
		reason      string
		quarantined bool
	}{{"invalid_grant", true}, {"revoked", true}, {"unrecoverable", true}, {"network_error", false}}
	for _, test := range tests {
		t.Run(test.reason, func(t *testing.T) {
			setupMachineV1Test(t, "alpha")
			lease, _ := beginOAuthRefresh(t, 'b')
			input := oauthRequest("oauth.refresh.abort", 'c', fmt.Sprintf(`"profile":"alpha","lease_id":%q,"reason":%q`, lease.LeaseID, test.reason))
			code, response, _ := invokeMachine(t, "oauth.refresh.abort", input)
			check(t, code == 0, "abort response = %+v, exit = %d", response, code)
			code, _, _ = invokeMachine(t, "credential.select", strings.Replace(testMachineRequest, strings.Repeat("a", 64), strings.Repeat("d", 64), 1))
			check(t, (code == machineExitUnavailable) == test.quarantined, "reason %q quarantine=%v exit=%d", test.reason, test.quarantined, code)
		})
	}
}

func TestMachineRefreshLeaseProcessKillBeforeCommit(t *testing.T) {
	setupMachineV1Test(t, "alpha")
	check(t, withMachineState(func() error { return saveMachineState(machineState{Generation: 1}) }) == nil, "seed generation")
	bin := filepath.Join(t.TempDir(), "acm")
	output, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	check(t, err == nil, "build: %v\n%s", err, output)
	run := func(operation, input string) (int, string, string) {
		cmd := exec.Command(bin, "machine", "v1", operation)
		cmd.Env, cmd.Stdin = append(os.Environ(), "HOME="+homeDir, "ACM_DIR="+acmDir), strings.NewReader(input)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		if err == nil {
			return 0, string(stdout), stderr.String()
		}
		return err.(*exec.ExitError).ExitCode(), string(stdout), stderr.String()
	}
	begin := oauthRequest("oauth.refresh.begin", 'b', `"profile":"alpha","generation":1`)
	code, stdout, stderr := run("oauth.refresh.begin", begin)
	check(t, code == 0 && stderr == "" && strings.Contains(stdout, `"lease_id"`), "process begin exit=%d stdout=%s stderr=%q", code, stdout, stderr)
	code, stdout, stderr = run("oauth.refresh.begin", oauthRequest("oauth.refresh.begin", 'c', `"profile":"alpha","generation":1`))
	check(t, code == 75 && stderr == "" && strings.Contains(stdout, "lease_busy"), "post-kill begin exit=%d stdout=%s stderr=%q", code, stdout, stderr)
}

func invokeMachine(t *testing.T, operation, input string) (int, machineResponse, string) {
	var output bytes.Buffer
	code := runMachine([]string{"v1", operation}, strings.NewReader(input), &output)
	var response machineResponse
	check(t, json.Unmarshal(output.Bytes(), &response) == nil, "decode %s", output.String())
	return code, response, output.String()
}

func assertMachineError(t *testing.T, args []string, input, want string) {
	var output bytes.Buffer
	code := runMachine(args, strings.NewReader(input), &output)
	var response machineResponse
	check(t, json.Unmarshal(output.Bytes(), &response) == nil, "decode %s", output.String())
	check(t, code == machineExitInvalid && response.Error != nil && response.Error.Code == want, "response = %+v, exit = %d", response, code)
}

func TestMachineProtocolRejectsUnknownVersionAndField(t *testing.T) {
	assertMachineError(t, []string{"v2", "credential.select"}, testMachineRequest, "unsupported_version")
	assertMachineError(t, []string{"v1", "credential.select"}, strings.TrimSuffix(testMachineRequest, "}")+`,"token":"secret"}`, "invalid_request")
}

func TestMachineSelectIsDeterministicSecretlessAndCanonical(t *testing.T) {
	claude := setupMachineV1Test(t)
	check(t, os.MkdirAll(claude.defaultHome, 0o755) == nil, "create principal")
	check(t, os.WriteFile(filepath.Join(claude.defaultHome, claude.credFile), nil, 0o600) == nil, "create principal credential")
	check(t, os.MkdirAll(filepath.Dir(profilePath(claude, "principal")), 0o755) == nil, "create profile root")
	check(t, os.Symlink(claude.defaultHome, profilePath(claude, "principal")) == nil, "link principal")

	code, response, raw := invokeMachine(t, "credential.select", testMachineRequest)
	check(t, code == 0 && response.Profile == "principal" && response.ConfigDir == claude.defaultHome && response.Generation == 1, "response = %+v, exit = %d", response, code)
	want := fmt.Sprintf("{\"config_dir\":%q,\"generation\":1,\"ok\":true,\"operation\":\"credential.select\",\"operation_id\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"profile\":\"principal\",\"schema_version\":1}\n", claude.defaultHome)
	check(t, raw == want, "non-canonical JSON:\n got %s want %s", raw, want)
	check(t, !strings.Contains(strings.ToLower(raw), "token") && !strings.Contains(raw, ".credentials.json"), "secret-bearing output: %s", raw)
	statusCode, status, _ := invokeMachine(t, "diagnostics.status", strings.Replace(testMachineRequest, "credential.select", "diagnostics.status", 1))
	check(t, statusCode == 0 && status.Generation == 1, "status = %+v, exit = %d", status, statusCode)
}

func TestMachineSelectRejectsSymlinkEscape(t *testing.T) {
	claude := setupMachineV1Test(t)
	escape := filepath.Join(homeDir, "escape")
	check(t, os.MkdirAll(escape, 0o755) == nil, "create escape")
	check(t, os.WriteFile(filepath.Join(escape, claude.credFile), nil, 0o600) == nil, "create escaped credential")
	check(t, os.MkdirAll(filepath.Dir(profilePath(claude, "evil")), 0o755) == nil, "create profile root")
	check(t, os.Symlink(escape, profilePath(claude, "evil")) == nil, "link escape")
	code, response, _ := invokeMachine(t, "credential.select", testMachineRequest)
	check(t, code == machineExitInvalid && response.Error != nil && response.Error.Code == "invalid_profile_path", "response = %+v, exit = %d", response, code)
}

func TestMachineLedgerIsOncePerProfileStaleAndBounded(t *testing.T) {
	setupMachineV1Test(t, "alpha", "beta")
	firstCode, first, _ := invokeMachine(t, "credential.select", testMachineRequest)
	secondCode, second, _ := invokeMachine(t, "credential.select", testMachineRequest)
	thirdCode, third, _ := invokeMachine(t, "credential.select", testMachineRequest)
	check(t, firstCode == 0 && secondCode == 0 && first.Profile != second.Profile, "profiles were not consumed once: %+v %+v", first, second)
	check(t, thirdCode == machineExitUnavailable && third.Error != nil && third.Error.Code == "no_available_profile", "third response = %+v, exit = %d", third, thirdCode)
	state, err := loadMachineState()
	check(t, err == nil, "load state: %v", err)
	state.Operations[0].UpdatedAt = time.Now().Add(-machineLedgerTTL - time.Second).Unix()
	for i := 0; i <= machineLedgerMax; i++ {
		state.Operations = append(state.Operations, machineOperation{ID: fmt.Sprintf("%064x", i+1), UpdatedAt: time.Now().Unix()})
	}
	state.Operations = pruneMachineLedger(state.Operations, time.Now().Unix())
	check(t, len(state.Operations) == machineLedgerMax, "ledger records = %d, want %d", len(state.Operations), machineLedgerMax)
	check(t, saveMachineState(state) == nil, "save state")
	code, response, _ := invokeMachine(t, "credential.select", testMachineRequest)
	check(t, code == 0, "stale operation was not reusable: %+v, exit = %d", response, code)
}

func TestMachineCLIProcessBounds(t *testing.T) {
	setupMachineV1Test(t, "alpha")
	bin := filepath.Join(t.TempDir(), "acm")
	build := exec.Command("go", "build", "-o", bin, ".")
	output, err := build.CombinedOutput()
	check(t, err == nil, "build: %v\n%s", err, output)
	run := func(input string) (int, string, string) {
		cmd := exec.Command(bin, "machine", "v1", "credential.select")
		cmd.Env = append(os.Environ(), "HOME="+homeDir, "ACM_DIR="+acmDir)
		cmd.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if err == nil {
			return 0, stdout.String(), stderr.String()
		}
		return err.(*exec.ExitError).ExitCode(), stdout.String(), stderr.String()
	}
	code, stdout, stderr := run(testMachineRequest)
	check(t, code == 0 && stderr == "" && len(stdout) <= machineMaxOutputBytes, "normal process: exit=%d stdout=%d stderr=%q", code, len(stdout), stderr)
	code, stdout, stderr = run(strings.Repeat("x", machineMaxInputBytes+1))
	check(t, code == machineExitInvalid && stderr == "" && len(stdout) <= machineMaxOutputBytes, "oversized process: exit=%d stdout=%d stderr=%q", code, len(stdout), stderr)
}
