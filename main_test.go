package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDoctorRestoresStateAndRedactedProfileVisibility(t *testing.T) {
	for _, test := range []struct {
		name, diagnostic string
		unavailable      bool
	}{
		{"available diagnostics", "opencode diagnostics: 1; active leases: 1", false},
		{"unavailable diagnostics", "opencode diagnostics: unavailable", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			profile := "private-profile-identifier"
			setupMachineV1Test(t, profile)
			oldOrder := toolOrder
			toolOrder = []string{"claude"}
			t.Cleanup(func() { toolOrder = oldOrder })
			fake := filepath.Join(t.TempDir(), "claude")
			check(t, os.WriteFile(fake, []byte("#!/bin/sh\nprintf 'fake 1.0\\n'\n"), 0o700) == nil, "create fake tool")
			t.Setenv("ACM_BIN_claude", fake)
			check(t, os.WriteFile(filepath.Join(profilePath(tools["claude"], profile), ".claude.json"), []byte(`{"emailAddress":"private@example.test"}`), 0o600) == nil, "seed private identity")
			check(t, os.MkdirAll(stateDir, 0o755) == nil, "create state directory")
			if test.unavailable {
				check(t, os.WriteFile(machineStateFile(), []byte("invalid"), 0o600) == nil, "seed invalid state")
			} else {
				state := machineState{
					Diagnostics: []machineDiagnostic{{Time: time.Now().UnixMilli(), Component: "oauth", Event: "refresh", Outcome: "failed"}},
					Leases:      []machineLease{{ID: "private-lease-identifier", Profile: profile, ExpiresAt: time.Now().Add(time.Minute).Unix()}},
				}
				check(t, saveMachineState(state) == nil, "seed doctor state")
			}
			text, code := captureStdout(t, cmdDoctor)
			check(t, code == 0 && strings.Contains(text, "estado : "+acmDir) && strings.Contains(text, test.diagnostic), "doctor output: %s", text)
			check(t, test.unavailable || strings.Contains(text, "oauth.refresh.failed: 1"), "doctor omitted diagnostic aggregate: %s", text)
			check(t, strings.Contains(text, "unknown") && strings.Contains(text, "disponible"), "doctor omitted redacted profile listing: %s", text)
			for _, private := range []string{profile, "private-lease-identifier", "private@example.test"} {
				check(t, !strings.Contains(text, private), "doctor leaked %q: %s", private, text)
			}
		})
	}
}

func captureStdout(t *testing.T, run func() int) (string, int) {
	t.Helper()
	reader, writer, err := os.Pipe()
	check(t, err == nil, "pipe: %v", err)
	oldStdout := os.Stdout
	os.Stdout = writer
	code := run()
	check(t, writer.Close() == nil, "close output")
	os.Stdout = oldStdout
	output, err := io.ReadAll(reader)
	check(t, err == nil, "read output: %v", err)
	return string(output), code
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
