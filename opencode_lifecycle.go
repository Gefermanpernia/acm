package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const openCodeManifest = ".acm-opencode-backup.json"

func cmdOpenCode(args []string) int { return runOpenCodeLifecycle(args, os.Stdout, os.Stderr) }

func runOpenCodeLifecycle(args []string, stdout, stderr io.Writer) int {
	replaceUpstream := len(args) == 3 && args[0] == "enable" && args[1] == "--confirm" && args[2] == "--replace-upstream"
	if !replaceUpstream && (len(args) != 2 || args[1] != "--confirm") {
		fmt.Fprintln(stderr, "acm: usa 'acm opencode enable --confirm [--replace-upstream]' o 'acm opencode rollback --confirm'")
		return 2
	}
	home := os.Getenv("ACM_OPENCODE_CONFIG_HOME")
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return 2
		}
		home = filepath.Join(h, ".config", "opencode")
	}
	var err error
	if args[0] == "enable" {
		err = enableOpenCode(home, replaceUpstream)
	} else if args[0] == "rollback" {
		err = rollbackOpenCode(home)
	} else {
		err = fmt.Errorf("operación desconocida")
	}
	if err != nil {
		fmt.Fprintln(stderr, "acm: "+err.Error())
		return 2
	}
	fmt.Fprintln(stdout, "✓ Configuración actualizada. Reinicia OpenCode para aplicar el cambio.")
	return 0
}

func enableOpenCode(home string, replaceUpstream bool) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("la integración OpenCode solo admite Linux")
	}
	plugin := os.Getenv("ACM_OPENCODE_PLUGIN_PATH")
	if plugin == "" {
		h, _ := os.UserHomeDir()
		plugin = filepath.Join(h, ".local", "share", "acm", "opencode", "index.js")
	}
	info, err := os.Lstat(plugin)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("el adaptador OpenCode de ACM no está instalado")
	}
	plugin, _ = filepath.Abs(plugin)
	pluginURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(plugin)}).String()
	path, err := findOpenCodeConfig(home)
	if err != nil {
		return err
	}
	original, err := readOpenCode(path)
	if err != nil {
		return err
	}
	upstream, acm, err := detectOpenCodePlugins(original, pluginURL)
	if err != nil {
		return fmt.Errorf("configuración JSON/JSONC ambigua o inválida")
	}
	if upstream && !replaceUpstream {
		if acm {
			return fmt.Errorf("conflicto de plugins; revisa la configuración y repite con --replace-upstream")
		}
		return fmt.Errorf("la migración requiere --replace-upstream")
	}
	if replaceUpstream && !upstream {
		return fmt.Errorf("no existe un plugin upstream que reemplazar")
	}
	updated, err := editOpenCode(original, pluginURL, true)
	if err != nil || !validateOpenCode(updated, pluginURL) {
		return fmt.Errorf("configuración JSON/JSONC ambigua o inválida")
	}
	manifest, backup := filepath.Join(home, openCodeManifest), path+".acm-backup"
	if _, err = os.Lstat(manifest); !os.IsNotExist(err) {
		return fmt.Errorf("ya existe un respaldo; revierte antes de habilitar")
	}
	if replaceUpstream {
		rollback, _ := editOpenCode(original, pluginURL, false)
		record := []byte(filepath.Base(path) + ":" + checksumOpenCode(rollback))
		if atomicWriteMachineFile(backup, rollback) != nil || atomicWriteMachineFile(manifest, record) != nil {
			os.Remove(backup)
			os.Remove(manifest)
			return fmt.Errorf("no se pudo crear el respaldo")
		}
	}
	if err = atomicWriteMachineFile(path, updated); err == nil {
		current, readErr := readOpenCode(path)
		if readErr != nil || !validateOpenCode(current, pluginURL) {
			err = fmt.Errorf("validation failed")
		}
	}
	if err != nil {
		atomicWriteMachineFile(path, original)
		os.Remove(backup)
		os.Remove(manifest)
		return fmt.Errorf("la migración falló y se restauró la configuración")
	}
	return nil
}

func rollbackOpenCode(home string) error {
	manifest := filepath.Join(home, openCodeManifest)
	record, err := readOpenCode(manifest)
	parts := strings.Split(string(record), ":")
	if err != nil || len(parts) != 2 || (parts[0] != "opencode.json" && parts[0] != "opencode.jsonc") || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(parts[1]) {
		return fmt.Errorf("no existe un respaldo válido")
	}
	path, backup := filepath.Join(home, parts[0]), filepath.Join(home, parts[0])+".acm-backup"
	data, err := readOpenCode(backup)
	if err != nil || checksumOpenCode(data) != parts[1] {
		return fmt.Errorf("el checksum del respaldo no es válido")
	}
	if _, _, _, err = parseOpenCode(data); err != nil || atomicWriteMachineFile(path, data) != nil {
		return fmt.Errorf("no se pudo restaurar el respaldo")
	}
	restored, err := readOpenCode(path)
	if err != nil || checksumOpenCode(restored) != parts[1] {
		return fmt.Errorf("no se pudo validar la restauración")
	}
	os.Remove(backup)
	os.Remove(manifest)
	return nil
}

func findOpenCodeConfig(home string) (string, error) {
	var found string
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		path := filepath.Join(home, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || found != "" {
			return "", fmt.Errorf("orígenes JSON/JSONC ambiguos o inseguros; no se cambió ninguno")
		}
		found = path
	}
	if found == "" {
		return "", fmt.Errorf("no existe opencode.json ni opencode.jsonc")
	}
	return found, nil
}

func readOpenCode(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, fmt.Errorf("archivo ausente, inseguro o demasiado grande")
	}
	return os.ReadFile(path)
}

var pluginArray = regexp.MustCompile(`"plugin"\s*:\s*(\[[^]]*\])`)
var jsoncComment = regexp.MustCompile(`(?ms)(^|[ \t])//[^\n]*|/\*.*?\*/`)
var trailingComma = regexp.MustCompile(`,\s*[]}]`)
var validateOpenCode = validEnabledOpenCode

func parseOpenCode(data []byte) (map[string]json.RawMessage, []string, []int, error) {
	clean, err := sanitizeJSONC(data)
	var document map[string]json.RawMessage
	if err != nil || json.Unmarshal(clean, &document) != nil {
		return nil, nil, nil, fmt.Errorf("invalid config")
	}
	matches := pluginArray.FindAllSubmatchIndex(clean, -1)
	raw, present := document["plugin"]
	if present != (len(matches) == 1) {
		return nil, nil, nil, fmt.Errorf("ambiguous plugin")
	}
	var plugins []string
	if present && json.Unmarshal(raw, &plugins) != nil {
		return nil, nil, nil, fmt.Errorf("invalid plugin")
	}
	if !present {
		return document, plugins, nil, nil
	}
	return document, plugins, matches[0], nil
}

func editOpenCode(data []byte, pluginURL string, enable bool) ([]byte, error) {
	document, plugins, match, err := parseOpenCode(data)
	if err != nil {
		return nil, err
	}
	result := plugins[:0]
	for _, value := range plugins {
		upstream, acm := classifyOpenCodePlugin(value, pluginURL)
		if !acm && (!enable || !upstream) {
			result = append(result, value)
		}
	}
	if enable {
		result = append(result, pluginURL)
	}
	encoded, _ := json.Marshal(result)
	if match != nil {
		start, end := match[2], match[3]
		return append(append(append([]byte{}, data[:start]...), encoded...), data[end:]...), nil
	}
	if !enable {
		return data, nil
	}
	clean, _ := sanitizeJSONC(data)
	close, prefix := bytes.LastIndexByte(clean, '}'), `,"plugin":`
	if len(document) == 0 {
		prefix = `"plugin":`
	}
	return append(append(append([]byte{}, data[:close]...), append([]byte(prefix), encoded...)...), data[close:]...), nil
}

func detectOpenCodePlugins(data []byte, pluginURL string) (upstream, acm bool, err error) {
	_, plugins, _, err := parseOpenCode(data)
	if err != nil {
		return false, false, err
	}
	for _, value := range plugins {
		isUpstream, isACM := classifyOpenCodePlugin(value, pluginURL)
		upstream, acm = upstream || isUpstream, acm || isACM
	}
	return upstream, acm, nil
}

func classifyOpenCodePlugin(value, pluginURL string) (upstream, acm bool) {
	upstream = value == "opencode-anthropic-login-via-cli" || strings.HasPrefix(value, "opencode-anthropic-login-via-cli@")
	acm = value == pluginURL || strings.HasSuffix(value, "/acm/opencode/index.js")
	return upstream, acm
}

func validEnabledOpenCode(data []byte, pluginURL string) bool {
	normalized, err := editOpenCode(data, pluginURL, true)
	return err == nil && bytes.Equal(normalized, data)
}

func sanitizeJSONC(data []byte) ([]byte, error) {
	out := append([]byte(nil), data...)
	for _, match := range jsoncComment.FindAllIndex(out, -1) {
		for i := match[0]; i < match[1]; i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	for _, match := range trailingComma.FindAllIndex(out, -1) {
		out[match[0]] = ' '
	}
	if !json.Valid(out) {
		return nil, fmt.Errorf("invalid JSONC")
	}
	return out, nil
}

func checksumOpenCode(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
