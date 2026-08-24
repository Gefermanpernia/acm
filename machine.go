package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
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
	machineDiagnosticMax   = 256
	machineExitInvalid     = 2
	machineExitUnavailable = 69
)

var operationIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.@-]{1,128}$`)
var machineSync = (*os.File).Sync
var machinePersist = atomicWriteMachineFile
var errMachineStaleGeneration = errors.New("machine generation changed")

type machineRequest struct {
	SchemaVersion int    `json:"schema_version"`
	Operation     string `json:"operation"`
	OperationID   string `json:"operation_id"`
	Profile       string `json:"profile"`
	Generation    uint64 `json:"generation"`
	LeaseID       string `json:"lease_id"`
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	ExpiresAt     int64  `json:"expires_at"`
	ResetAt       int64  `json:"reset_at"`
	Reason        string `json:"reason"`
	Component     string `json:"component"`
	Event         string `json:"event"`
	Outcome       string `json:"outcome"`
	Retryable     bool   `json:"retryable"`
}

type machineOperation struct {
	ID        string   `json:"id"`
	UpdatedAt int64    `json:"updated_at"`
	Profiles  []string `json:"profiles"`
	Exhausted []string `json:"exhausted,omitempty"`
}

type machineState struct {
	Generation  uint64              `json:"generation"`
	Operations  []machineOperation  `json:"operations"`
	Leases      []machineLease      `json:"leases,omitempty"`
	Quarantined []string            `json:"quarantined,omitempty"`
	Cooling     map[string]int64    `json:"cooling,omitempty"`
	Diagnostics []machineDiagnostic `json:"diagnostics,omitempty"`
}

type machineDiagnostic struct {
	Time      int64  `json:"time"`
	Component string `json:"component"`
	Event     string `json:"event"`
	Outcome   string `json:"outcome"`
	Retryable bool   `json:"retryable"`
}

type machineLease struct {
	ID        string `json:"id"`
	Profile   string `json:"profile"`
	ExpiresAt int64  `json:"expires_at"`
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
		case "diagnostics.record":
			response, exit = recordMachineDiagnostic(req)
		case "oauth.refresh.begin":
			response, exit = beginMachineRefresh(req)
		case "oauth.refresh.commit":
			response, exit = commitMachineRefresh(req)
		case "oauth.refresh.abort":
			response, exit = abortMachineRefresh(req)
		case "quota.exhaust":
			response, exit = exhaustMachineQuota(req)
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
	if req.Operation != op || !operationIDPattern.MatchString(req.OperationID) || !validMachineRequest(req) {
		return req, machineFailure(op, "", "invalid_request", "operation or operation_id is invalid", false), machineExitInvalid
	}
	return req, nil, 0
}

func validMachineRequest(req machineRequest) bool {
	if req.Operation == "diagnostics.record" {
		return req.Profile == "" && req.Generation == 0 && req.LeaseID == "" && req.AccessToken == "" && req.RefreshToken == "" && req.ExpiresAt == 0 && req.ResetAt == 0 && req.Reason == "" && req.Component != "" && req.Event != "" && req.Outcome != ""
	}
	if req.Operation == "oauth.refresh.commit" {
		return profileNamePattern.MatchString(req.Profile) && req.Generation > 0 && req.LeaseID != "" && req.AccessToken != "" && req.RefreshToken != "" && req.ExpiresAt > 0 && req.ResetAt == 0 && req.Reason == ""
	}
	if req.Operation == "quota.exhaust" {
		return profileNamePattern.MatchString(req.Profile) && req.Generation > 0 && req.LeaseID == "" && req.AccessToken == "" && req.RefreshToken == "" && req.ExpiresAt == 0 && req.Reason == ""
	}
	return req.AccessToken == "" && req.RefreshToken == "" && req.ResetAt == 0 && req.Component == "" && req.Event == "" && req.Outcome == "" && !req.Retryable
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
		name, dir, err := nextMachineProfile(tools["claude"], record.Profiles, state.Quarantined, state.Cooling, timestamp)
		if err != nil {
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_profile_path", "ACM profile path is unsafe", false), machineExitInvalid
			return nil
		}
		if name == "" {
			response, exit = machineUnavailableResponse(req, state, timestamp)
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

func machineUnavailableResponse(req machineRequest, state machineState, timestamp int64) (map[string]any, int) {
	t, profileCount, quarantinedCount, resetAt := tools["claude"], 0, 0, int64(0)
	for _, profile := range orderedProfiles(t, false) {
		dir, err := canonicalMachineProfile(t, profile)
		if err != nil {
			return machineFailure(req.Operation, req.OperationID, "invalid_profile_path", "ACM profile path is unsafe", false), machineExitInvalid
		}
		if dir == "" {
			continue
		}
		profileCount++
		if slices.Contains(state.Quarantined, profile) {
			quarantinedCount++
			continue
		}
		until := state.Cooling[profile]
		if legacy := cooldownUntil(t, profile); legacy > until {
			until = legacy
		}
		if until > timestamp && (resetAt == 0 || until < resetAt) {
			resetAt = until
		}
	}
	if resetAt > 0 {
		response := machineFailure(req.Operation, req.OperationID, "no_available_profile", "ACM profiles are cooling", true)
		response["reset_at"] = resetAt
		return response, 75
	}
	if profileCount > 0 && quarantinedCount == profileCount {
		return machineFailure(req.Operation, req.OperationID, "credential_quarantined", "credential requires acm login", false), machineExitUnavailable
	}
	return machineFailure(req.Operation, req.OperationID, "no_available_profile", "no unattempted ACM profile is available", false), machineExitUnavailable
}

func machineStatus(req machineRequest) (map[string]any, int) {
	state, err := loadMachineState()
	if err != nil {
		return machineFailure(req.Operation, req.OperationID, "state_unavailable", "ACM state is unavailable", false), 74
	}
	diagnostics, active := machineDiagnosticSnapshot(state, time.Now())
	if diagnostics == nil {
		diagnostics = []machineDiagnostic{}
	}
	if len(diagnostics) > 64 {
		diagnostics = diagnostics[len(diagnostics)-64:]
	}
	return map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation,
		"operation_id": req.OperationID, "generation": state.Generation, "diagnostics": diagnostics, "active_leases": active}, 0
}

func recordMachineDiagnostic(req machineRequest) (response map[string]any, exit int) {
	err := updateMachineState(func(state *machineState) error {
		appendMachineDiagnostic(state, req.Component, req.Event, req.Outcome, req.Retryable)
		response = map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation, "operation_id": req.OperationID, "outcome": "recorded"}
		return nil
	})
	return finishMachineState(req, response, exit, err)
}

func appendMachineDiagnostic(state *machineState, component, event, outcome string, retryable bool) {
	record := machineDiagnostic{Time: time.Now().UnixMilli(), Component: safeDiagnostic(component, "quota", "oauth", "adapter"), Event: safeDiagnostic(event, "transition", "refresh", "compatibility", "recovery"), Outcome: safeDiagnostic(outcome, "cooling", "quarantined", "unavailable", "failed", "recovered"), Retryable: retryable}
	state.Diagnostics = pruneMachineDiagnostics(append(state.Diagnostics, record), record.Time)
}

func safeDiagnostic(value string, allowed ...string) string {
	if slices.Contains(allowed, value) {
		return value
	}
	return "unknown"
}

func pruneMachineDiagnostics(events []machineDiagnostic, timestamp int64) []machineDiagnostic {
	cutoff := timestamp - machineLedgerTTL.Milliseconds()
	events = slices.DeleteFunc(events, func(event machineDiagnostic) bool { return event.Time <= cutoff })
	if len(events) > machineDiagnosticMax {
		events = events[len(events)-machineDiagnosticMax:]
	}
	return events
}

func machineDiagnosticSnapshot(state machineState, timestamp time.Time) ([]machineDiagnostic, int) {
	diagnostics := pruneMachineDiagnostics(state.Diagnostics, timestamp.UnixMilli())
	active := 0
	for _, lease := range state.Leases {
		if lease.ExpiresAt > timestamp.Unix() {
			active++
		}
	}
	return diagnostics, active
}

func nextMachineProfile(t *tool, attempted, quarantined []string, cooling map[string]int64, timestamp int64) (string, string, error) {
	for _, name := range orderedProfiles(t, false) {
		dir, err := canonicalMachineProfile(t, name)
		if dir == "" && err == nil {
			continue
		}
		if err != nil {
			return "", "", err
		}
		if !inCooldown(t, name) && cooling[name] <= timestamp && !slices.Contains(attempted, name) && !slices.Contains(quarantined, name) {
			return name, dir, nil
		}
	}
	return "", "", nil
}

func finishMachineState(req machineRequest, response map[string]any, exit int, err error) (map[string]any, int) {
	if err == nil || exit == 74 && response != nil {
		return response, exit
	}
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return machineFailure(req.Operation, req.OperationID, "state_busy", "ACM state is busy", true), 75
	}
	return machineFailure(req.Operation, req.OperationID, "state_unavailable", "ACM state is unavailable", false), 74
}

func updateMachineState(fn func(*machineState) error) error {
	return withMachineState(func() error {
		state, err := loadMachineState()
		if err != nil {
			return err
		}
		if err = fn(&state); err != nil {
			return err
		}
		return saveMachineState(state)
	})
}

func beginMachineRefresh(req machineRequest) (response map[string]any, exit int) {
	err := updateMachineState(func(state *machineState) error {
		now := time.Now().Unix()
		state.Leases = slices.DeleteFunc(state.Leases, func(lease machineLease) bool { return lease.ExpiresAt <= now })
		if state.Generation != req.Generation {
			response, exit = machineFailure(req.Operation, req.OperationID, "stale_generation", "refresh generation is stale", true), 75
			return nil
		}
		if slices.Contains(state.Quarantined, req.Profile) {
			response, exit = machineFailure(req.Operation, req.OperationID, "credential_quarantined", "credential requires acm login", false), machineExitUnavailable
			return nil
		}
		if dir, pathErr := canonicalMachineProfile(tools["claude"], req.Profile); pathErr != nil || dir == "" {
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_profile_path", "ACM profile path is unsafe", false), machineExitInvalid
			return nil
		}
		if slices.ContainsFunc(state.Leases, func(lease machineLease) bool { return lease.Profile == req.Profile }) {
			response, exit = machineFailure(req.Operation, req.OperationID, "lease_busy", "refresh lease is busy", true), 75
			return nil
		}
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return err
		}
		lease := machineLease{ID: hex.EncodeToString(token[:]), Profile: req.Profile, ExpiresAt: now + 120}
		state.Leases = append(state.Leases, lease)
		response = map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation, "operation_id": req.OperationID, "lease_id": lease.ID, "expires_at": lease.ExpiresAt, "generation": state.Generation}
		return nil
	})
	return finishMachineState(req, response, exit, err)
}

func commitMachineRefresh(req machineRequest) (response map[string]any, exit int) {
	err := updateMachineState(func(state *machineState) error {
		index := slices.IndexFunc(state.Leases, func(lease machineLease) bool { return lease.ID == req.LeaseID && lease.Profile == req.Profile })
		if index < 0 {
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_lease", "refresh lease is invalid", true), 75
			return nil
		}
		lease := state.Leases[index]
		if lease.ExpiresAt <= time.Now().Unix() || state.Generation != req.Generation {
			state.Leases = append(state.Leases[:index], state.Leases[index+1:]...)
			code := "stale_generation"
			if lease.ExpiresAt <= time.Now().Unix() {
				code = "lease_expired"
			}
			response, exit = machineFailure(req.Operation, req.OperationID, code, "refresh lease is stale", true), 75
			return nil
		}
		dir, err := canonicalMachineProfile(tools["claude"], req.Profile)
		if err != nil || dir == "" {
			return errors.New("unsafe credential path")
		}
		path := filepath.Join(dir, tools["claude"].credFile)
		data, err := os.ReadFile(path)
		var document map[string]any
		if err != nil || json.Unmarshal(data, &document) != nil {
			return errors.New("invalid credential state")
		}
		oauth, _ := document["claudeAiOauth"].(map[string]any)
		if oauth == nil {
			oauth = map[string]any{}
		}
		oauth["accessToken"], oauth["refreshToken"], oauth["expiresAt"] = req.AccessToken, req.RefreshToken, req.ExpiresAt
		document["claudeAiOauth"] = oauth
		data, err = json.Marshal(document)
		if err == nil {
			err = machinePersist(path, append(data, '\n'))
		}
		if err != nil {
			response, exit = machineFailure(req.Operation, req.OperationID, "persistence_failed", "credential persistence failed", false), 74
			return err
		}
		state.Leases = append(state.Leases[:index], state.Leases[index+1:]...)
		state.Generation++
		response = map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation, "operation_id": req.OperationID, "outcome": "committed", "generation": state.Generation}
		return nil
	})
	return finishMachineState(req, response, exit, err)
}

func abortMachineRefresh(req machineRequest) (response map[string]any, exit int) {
	err := updateMachineState(func(state *machineState) error {
		index := slices.IndexFunc(state.Leases, func(lease machineLease) bool { return lease.ID == req.LeaseID && lease.Profile == req.Profile })
		if index < 0 {
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_lease", "refresh lease is invalid", true), 75
			return nil
		}
		state.Leases = append(state.Leases[:index], state.Leases[index+1:]...)
		terminal := req.Reason == "invalid_grant" || req.Reason == "revoked" || req.Reason == "unrecoverable"
		outcome := "aborted"
		if terminal && mutateMachineProfileAvailability(state, req.Profile, true, 0) {
			state.Generation++
			outcome = "quarantined"
		}
		response = map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation, "operation_id": req.OperationID, "outcome": outcome, "generation": state.Generation}
		return nil
	})
	return finishMachineState(req, response, exit, err)
}

func exhaustMachineQuota(req machineRequest) (response map[string]any, exit int) {
	err := withMachineState(func() error {
		state, err := loadMachineState()
		if err != nil {
			return err
		}
		timestamp := time.Now().Unix()
		operations := pruneMachineLedger(state.Operations, timestamp)
		index := slices.IndexFunc(operations, func(operation machineOperation) bool { return operation.ID == req.OperationID })
		if index < 0 {
			response, exit = machineFailure(req.Operation, req.OperationID, "unknown_operation", "logical operation is unknown", false), machineExitInvalid
			return nil
		}
		record := operations[index]
		if slices.Contains(record.Exhausted, req.Profile) {
			response, exit = machineQuotaResponse(req, state, record, timestamp, state.Cooling[req.Profile])
			return nil
		}
		if dir, pathErr := canonicalMachineProfile(tools["claude"], req.Profile); pathErr != nil || dir == "" || !slices.Contains(record.Profiles, req.Profile) {
			response, exit = machineFailure(req.Operation, req.OperationID, "invalid_profile_path", "ACM profile path is unsafe", false), machineExitInvalid
			return nil
		}
		if state.Generation != req.Generation {
			response, exit = machineFailure(req.Operation, req.OperationID, "stale_generation", "quota generation is stale", true), 75
			return nil
		}
		resetAt := req.ResetAt
		if resetAt <= timestamp {
			resetAt = timestamp + int64(defaultCooldownMin)*60
		}
		mutateMachineProfileAvailability(&state, req.Profile, false, resetAt)
		record.Exhausted = append(record.Exhausted, req.Profile)
		record.UpdatedAt = timestamp
		operations[index] = record
		state.Operations = operations
		state.Generation++
		response, exit = machineQuotaResponse(req, state, record, timestamp, resetAt)
		if exit != 0 {
			return nil
		}
		if err = saveMachineState(state); err != nil {
			return err
		}
		return nil
	})
	return finishMachineState(req, response, exit, err)
}

func mutateMachineProfileAvailability(state *machineState, profile string, quarantine bool, resetAt int64) bool {
	if resetAt > 0 {
		if state.Cooling == nil {
			state.Cooling = make(map[string]int64)
		}
		changed := state.Cooling[profile] != resetAt
		state.Cooling[profile] = resetAt
		return changed
	}
	index := slices.Index(state.Quarantined, profile)
	if quarantine {
		if index >= 0 {
			return false
		}
		state.Quarantined = append(state.Quarantined, profile)
		return true
	}
	if index < 0 {
		return false
	}
	state.Quarantined = slices.Delete(state.Quarantined, index, index+1)
	return true
}

func machineLoginState(profile string) (generation uint64, quarantined bool, exit int) {
	err := withMachineState(func() error {
		state, err := loadMachineState()
		if err == nil {
			generation, quarantined = state.Generation, slices.Contains(state.Quarantined, profile)
		}
		return err
	})
	return generation, quarantined, machineStateExit(err)
}

func recoverMachineProfile(profile string, generation uint64) int {
	err := withMachineState(func() error {
		state, err := loadMachineState()
		if err != nil {
			return err
		}
		if state.Generation != generation {
			return errMachineStaleGeneration
		}
		if !mutateMachineProfileAvailability(&state, profile, false, 0) {
			return nil
		}
		state.Operations = pruneMachineLedger(state.Operations, time.Now().Unix())
		state.Generation++
		appendMachineDiagnostic(&state, "oauth", "recovery", "recovered", false)
		return saveMachineState(state)
	})
	return machineStateExit(err)
}

func machineStateExit(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, errMachineStaleGeneration) {
		return 75
	}
	return 74
}

func machineQuotaResponse(req machineRequest, state machineState, record machineOperation, timestamp, resetAt int64) (map[string]any, int) {
	replacement, _, err := nextMachineProfile(tools["claude"], record.Profiles, state.Quarantined, state.Cooling, timestamp)
	if err != nil {
		return machineFailure(req.Operation, req.OperationID, "invalid_profile_path", "ACM profile path is unsafe", false), machineExitInvalid
	}
	return map[string]any{"schema_version": 1, "ok": true, "operation": req.Operation, "operation_id": req.OperationID,
		"outcome": "cooling", "generation": state.Generation, "reset_at": resetAt, "replacement_available": replacement != ""}, 0
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
	return machinePersist(machineStateFile(), append(data, '\n'))
}

func atomicWriteMachineFile(path string, data []byte) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { file.Close(); os.Remove(temporary) }()
	_, err = file.Write(data)
	if err == nil {
		err = machineSync(file)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(temporary, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return machineSync(directory)
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
