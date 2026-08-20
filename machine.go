package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"syscall"
	"time"
)

const (
	machineMaxInputBytes   = 64 << 10
	machineMaxOutputBytes  = 16 << 10
	machineLedgerMax       = 1024
	machineLedgerTTL       = 24 * time.Hour
	machineExitInvalid     = 2
	machineExitUnavailable = 69
)

var operationIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.@-]{1,128}$`)

type machineRequest struct {
	SchemaVersion int    `json:"schema_version"`
	Operation     string `json:"operation"`
	OperationID   string `json:"operation_id"`
}

type machineOperation struct {
	ID        string   `json:"id"`
	UpdatedAt int64    `json:"updated_at"`
	Profiles  []string `json:"profiles"`
}

type machineState struct {
	Generation uint64             `json:"generation"`
	Operations []machineOperation `json:"operations"`
}

func machineFailure(op, id, code, message string, retryable bool) map[string]any {
	return map[string]any{"schema_version": 1, "ok": false, "operation": op, "operation_id": id,
		"error": map[string]any{"code": code, "message": message, "retryable": retryable}}
}

func runMachine(args []string, in io.Reader, out io.Writer) int {
	req, response, exit := decodeMachineRequest(args, in)
	if exit == 0 {
		switch req.Operation {
		case "credential.select":
			response, exit = selectMachineProfile(req)
		case "diagnostics.status":
			response, exit = machineStatus(req)
		default:
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_operation", "unsupported machine operation", false), machineExitInvalid
		}
	}
	data, _ := json.Marshal(response)
	data = append(data, '\n')
	if len(data) > machineMaxOutputBytes {
		data, _ = json.Marshal(machineFailure(req.Operation, req.OperationID, "output_too_large", "machine response exceeds 16 KiB", false))
		data = append(data, '\n')
		exit = 74
	}
	if _, err := out.Write(data); err != nil {
		return 74
	}
	return exit
}

func decodeMachineRequest(args []string, in io.Reader) (machineRequest, map[string]any, int) {
	if len(args) != 2 {
		return machineRequest{}, machineFailure("", "", "unsupported_version", "expected machine protocol v1", false), machineExitInvalid
	}
	op := args[1]
	if args[0] != "v1" {
		return machineRequest{}, machineFailure(op, "", "unsupported_version", "expected machine protocol v1", false), machineExitInvalid
	}
	data, err := io.ReadAll(io.LimitReader(in, machineMaxInputBytes+1))
	if err != nil || len(data) > machineMaxInputBytes {
		return machineRequest{}, machineFailure(op, "", "invalid_request", "machine request exceeds 64 KiB", false), machineExitInvalid
	}
	var req machineRequest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return req, machineFailure(op, "", "invalid_request", "expected one strict JSON request", false), machineExitInvalid
	}
	if req.SchemaVersion != 1 {
		return req, machineFailure(op, "", "unsupported_version", "unsupported schema version", false), machineExitInvalid
	}
	if req.Operation != op || !operationIDPattern.MatchString(req.OperationID) {
		return req, machineFailure(op, "", "invalid_request", "operation or operation_id is invalid", false), machineExitInvalid
	}
	return req, nil, 0
}

func selectMachineProfile(req machineRequest) (response map[string]any, exit int) {
	err := withMachineState(func() error {
		state, err := loadMachineState()
		if err != nil {
			return err
		}
		timestamp := time.Now().Unix()
		state.Operations = pruneMachineLedger(state.Operations, timestamp)
		record := machineOperation{ID: req.OperationID}
		for i, candidate := range state.Operations {
			if candidate.ID == req.OperationID {
				record = candidate
				state.Operations = append(state.Operations[:i], state.Operations[i+1:]...)
				break
			}
		}
		name, dir, err := nextMachineProfile(tools["claude"], record.Profiles)
		if err != nil {
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_profile_path", "ACM profile path is unsafe", false), machineExitInvalid
			return nil
		}
		if name == "" {
			response, exit = machineFailure(req.Operation, req.OperationID, "no_available_profile", "no unattempted ACM profile is available", false), machineExitUnavailable
			return nil
		}
		record.Profiles, record.UpdatedAt = append(record.Profiles, name), timestamp
		state.Operations = pruneMachineLedger(append(state.Operations, record), timestamp)
		state.Generation++
		if err = saveMachineState(state); err != nil {
			return err
		}
		response = map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation, "operation_id": req.OperationID,
			"profile": name, "config_dir": dir, "generation": state.Generation}
		return nil
	})
	if err == nil {
		return response, exit
	}
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return machineFailure(req.Operation, req.OperationID, "state_busy", "ACM state is busy", true), 75
	}
	return machineFailure(req.Operation, req.OperationID, "state_unavailable", "ACM state is unavailable", false), 74
}

func machineStatus(req machineRequest) (map[string]any, int) {
	state, err := loadMachineState()
	if err != nil {
		return machineFailure(req.Operation, req.OperationID, "state_unavailable", "ACM state is unavailable", false), 74
	}
	return map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation,
		"operation_id": req.OperationID, "generation": state.Generation}, 0
}

func nextMachineProfile(t *tool, attempted []string) (string, string, error) {
	for _, name := range orderedProfiles(t, false) {
		dir, err := canonicalMachineProfile(t, name)
		if dir == "" && err == nil {
			continue
		}
		if err != nil {
			return "", "", err
		}
		if !inCooldown(t, name) && !slices.Contains(attempted, name) {
			return name, dir, nil
		}
	}
	return "", "", nil
}

func canonicalMachineProfile(t *tool, name string) (string, error) {
	if !profileNamePattern.MatchString(name) {
		return "", errors.New("invalid profile name")
	}
	base, baseErr := filepath.EvalSymlinks(filepath.Join(profDir, t.name))
	dir, dirErr := filepath.EvalSymlinks(profilePath(t, name))
	if baseErr != nil || dirErr != nil {
		return "", errors.Join(baseErr, dirErr)
	}
	if !pathWithin(base, dir) {
		principal, _ := filepath.EvalSymlinks(t.defaultHome)
		if name != "principal" || dir != principal {
			return "", errors.New("profile escapes root")
		}
	}
	credential, err := filepath.EvalSymlinks(filepath.Join(dir, t.credFile))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil || !pathWithin(dir, credential) {
		return "", errors.New("credential escapes profile")
	}
	info, err := os.Stat(credential)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("credential is not regular")
	}
	return dir, nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && filepath.IsLocal(rel)
}

func pruneMachineLedger(records []machineOperation, timestamp int64) []machineOperation {
	cutoff := timestamp - int64(machineLedgerTTL/time.Second)
	records = slices.DeleteFunc(records, func(record machineOperation) bool {
		return record.UpdatedAt <= cutoff
	})
	if len(records) > machineLedgerMax {
		records = records[len(records)-machineLedgerMax:]
	}
	return records
}

func machineStateFile() string { return filepath.Join(stateDir, "opencode-machine-v1.json") }

func loadMachineState() (machineState, error) {
	var state machineState
	data, err := os.ReadFile(machineStateFile())
	if os.IsNotExist(err) {
		return state, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &state)
	}
	return state, err
}

func saveMachineState(state machineState) error {
	data, _ := json.Marshal(state)
	path := machineStateFile()
	if err := os.WriteFile(path+".tmp", append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

func withMachineState(fn func() error) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(stateDir, ".machine.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn()
}

func cmdMachine(args []string) int { return runMachine(args, os.Stdin, os.Stdout) }
