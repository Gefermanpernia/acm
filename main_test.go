package main

import (
	"io"
	"os"
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
