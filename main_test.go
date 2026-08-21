package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDoctorReportsRedactedAdapterDiagnosticsAndLeaseHealth(t *testing.T) {
	setupMachineV1Test(t)
	oldOrder := toolOrder
	toolOrder = nil
	t.Cleanup(func() { toolOrder = oldOrder })
	state := machineState{
		Diagnostics: []machineDiagnostic{{Time: time.Now().UnixMilli(), Component: "oauth", Event: "refresh", Outcome: "failed", Retryable: false}},
		Leases:      []machineLease{{ID: "private-lease-identifier", Profile: "private-profile", ExpiresAt: time.Now().Add(time.Minute).Unix()}},
	}
	check(t, os.MkdirAll(stateDir, 0o755) == nil && saveMachineState(state) == nil, "seed doctor state")
	reader, writer, err := os.Pipe()
	check(t, err == nil, "pipe: %v", err)
	oldStdout := os.Stdout
	os.Stdout = writer
	code := cmdDoctor()
	writer.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(reader)
	text := string(output)
	check(t, code == 0 && strings.Contains(text, "opencode diagnostics: 1; active leases: 1") && strings.Contains(text, "oauth.refresh.failed: 1"), "doctor output: %s", text)
	check(t, !strings.Contains(text, acmDir) && !strings.Contains(text, "private-lease") && !strings.Contains(text, "private-profile"), "doctor leaked private state: %s", text)
}

func TestLoginRecoversOnlySuccessfulClaudeProfile(t *testing.T) {
	for _, test := range []struct {
		name        string
		remaining   []string
		login, want int
		change      bool
	}{
		{"successful login", []string{"beta"}, 0, 0, true},
		{"failed login", []string{"alpha", "beta"}, 1, 1, false},
		{"partial login", []string{"alpha", "beta"}, 0, machineExitUnavailable, false},
		{"aborted login", []string{"alpha", "beta"}, 130, 130, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupMachineV1Test(t, "alpha", "beta")
			initial := machineState{Generation: 7, Quarantined: []string{"alpha", "beta"}, Cooling: map[string]int64{"alpha": 11, "beta": 22}}
			check(t, os.MkdirAll(stateDir, 0o755) == nil && saveMachineState(initial) == nil, "seed login state")
			check(t, os.MkdirAll(coolDir, 0o755) == nil && os.WriteFile(coolFile(tools["claude"], "alpha"), []byte("99\n"), 0o600) == nil, "seed legacy cooling")
			old := loginInteractive
			loginInteractive = func(tool *tool, profile string, _ []string) int {
				if test.change {
					check(t, os.WriteFile(filepath.Join(resolvedDir(tool, profile), tool.credFile), []byte(`{"claudeAiOauth":{"accessToken":"synthetic-new","refreshToken":"synthetic-new","expiresAt":2000000000000}}`), 0o600) == nil, "replace credential")
				}
				return test.login
			}
			t.Cleanup(func() { loginInteractive = old })
			check(t, cmdLogin([]string{"claude", "alpha"}) == test.want, "login exit")
			state, err := loadMachineState()
			check(t, err == nil && strings.Join(state.Quarantined, ",") == strings.Join(test.remaining, ","), "quarantine changed incorrectly: %+v", state)
			check(t, state.Cooling["alpha"] == 11 && state.Cooling["beta"] == 22, "login changed cooling: %+v", state.Cooling)
			_, legacyErr := os.Stat(coolFile(tools["claude"], "alpha"))
			check(t, os.IsNotExist(legacyErr), "login retained legacy cooling: %v", legacyErr)
			if test.change {
				check(t, state.Generation == 8 && len(state.Diagnostics) == 1 && state.Diagnostics[0].Event == "recovery", "recovery not recorded: %+v", state)
			} else {
				check(t, state.Generation == 7 && len(state.Diagnostics) == 0, "failed login changed state: %+v", state)
			}
		})
	}
}

func TestLoginRecoveryRejectsConcurrentGenerationChange(t *testing.T) {
	setupMachineV1Test(t, "alpha", "beta")
	initial := machineState{Generation: 7, Quarantined: []string{"alpha", "beta"}, Cooling: map[string]int64{"beta": 22}}
	check(t, os.MkdirAll(stateDir, 0o755) == nil && saveMachineState(initial) == nil, "seed login state")
	old := loginInteractive
	loginInteractive = func(*tool, string, []string) int {
		state, _ := loadMachineState()
		state.Generation++
		check(t, saveMachineState(state) == nil, "advance generation")
		check(t, os.WriteFile(filepath.Join(resolvedDir(tools["claude"], "alpha"), tools["claude"].credFile), []byte(`{"claudeAiOauth":{"accessToken":"synthetic-new","refreshToken":"synthetic-new","expiresAt":2000000000000}}`), 0o600) == nil, "replace credential")
		return 0
	}
	t.Cleanup(func() { loginInteractive = old })

	check(t, cmdLogin([]string{"claude", "alpha"}) == 75, "stale recovery must be transient")
	state, _ := loadMachineState()
	check(t, state.Generation == 8 && len(state.Quarantined) == 2 && len(state.Diagnostics) == 0 && state.Cooling["beta"] == 22, "stale recovery changed state: %+v", state)
}
