package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testMachineRequest = `{"schema_version":1,"operation":"credential.select","operation_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`

type machineResponse struct {
	Profile    string `json:"profile"`
	ConfigDir  string `json:"config_dir"`
	Generation uint64 `json:"generation"`
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
		check(t, os.WriteFile(filepath.Join(dir, claude.credFile), nil, 0o600) == nil, "create credential %s", name)
	}
	return claude
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
