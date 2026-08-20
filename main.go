// acm — Agent aCcount Manager
//
// Gestor de cuentas para CLIs de agentes (Claude Code y Codex CLI):
// perfiles aislados por cuenta (CLAUDE_CONFIG_DIR / CODEX_HOME), failover
// automático al agotar límites de uso, y consulta de cuota sin gastar tokens.
//
// Estado en ~/.acm (compatible con la versión bash original):
//
//	profiles/<tool>/<perfil>/   directorio de config de cada cuenta
//	state/<tool>.current        perfil activo
//	state/cooldown/<tool>.<perfil>  epoch hasta el que la cuenta está limitada
//
// El perfil "principal" es un symlink al home por defecto (~/.claude, ~/.codex)
// y en ese caso NO se exporta la variable (en claude, exportarla movería
// ~/.claude.json al interior del directorio).
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const version = "2.0.0"

// ---------- configuración global ----------

var (
	homeDir string
	acmDir  string
	profDir string
	stateDir string
	coolDir string
	defaultCooldownMin = 60
)

type tool struct {
	name        string
	envVar      string
	credFile    string
	defaultHome string
	execArgs    []string // modo no interactivo: claude -p / codex exec
	limitRe     *regexp.Regexp
	authRe      *regexp.Regexp
	seedFiles   []string // config no-secreta heredada al crear un perfil
}

var toolOrder = []string{"claude", "codex"}
var tools map[string]*tool

func initGlobals() {
	var err error
	homeDir, err = os.UserHomeDir()
	if err != nil {
		die("no se pudo determinar el home: " + err.Error())
	}
	acmDir = os.Getenv("ACM_DIR")
	if acmDir == "" {
		acmDir = filepath.Join(homeDir, ".acm")
	}
	profDir = filepath.Join(acmDir, "profiles")
	stateDir = filepath.Join(acmDir, "state")
	coolDir = filepath.Join(stateDir, "cooldown")
	if v := os.Getenv("ACM_DEFAULT_COOLDOWN_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			defaultCooldownMin = n
		}
	}
	tools = map[string]*tool{
		"claude": {
			name:        "claude",
			envVar:      "CLAUDE_CONFIG_DIR",
			credFile:    ".credentials.json",
			defaultHome: filepath.Join(homeDir, ".claude"),
			execArgs:    []string{"-p"},
			limitRe:     regexp.MustCompile(`(?i)usage limit reached|hit your (session|weekly|usage|opus).{0,4}limit|(session|weekly|5-hour) limit reached`),
			authRe:      regexp.MustCompile(`(?i)failed to authenticate|oauth (session |access )?token.{0,20}(expired|revoked)|re-authenticate|please run /login|not logged in|invalid bearer token|authentication_error`),
			seedFiles:   []string{"settings.json", "CLAUDE.md"},
		},
		"codex": {
			name:        "codex",
			envVar:      "CODEX_HOME",
			credFile:    "auth.json",
			defaultHome: filepath.Join(homeDir, ".codex"),
			execArgs:    []string{"exec"},
			limitRe:     regexp.MustCompile(`(?i)hit your usage limit|usage_limit_reached|rate_limit_reached|workspace_[a-z_]*usage_limit`),
			authRe:      regexp.MustCompile(`(?i)not logged in|login required|token refresh failed|401 unauthorized|reauthenticate`),
			seedFiles:   []string{"config.toml", "AGENTS.md", "hooks.json"},
		},
	}
	for _, t := range toolOrder {
		_ = os.MkdirAll(filepath.Join(profDir, t), 0o755)
	}
	_ = os.MkdirAll(coolDir, 0o755)

	// brew (linuxbrew/homebrew) puede no estar en PATH en shells no interactivos
	path := os.Getenv("PATH")
	for _, d := range []string{"/home/linuxbrew/.linuxbrew/bin", "/opt/homebrew/bin", "/usr/local/bin"} {
		if st, err := os.Stat(d); err == nil && st.IsDir() && !strings.Contains(":"+path+":", ":"+d+":") {
			path = path + string(os.PathListSeparator) + d
		}
	}
	_ = os.Setenv("PATH", path)
}

// ---------- utilidades ----------

func die(msg string) {
	fmt.Fprintln(os.Stderr, "acm: "+msg)
	os.Exit(2)
}

func now() int64 { return time.Now().Unix() }

func binFor(t *tool) string {
	if ov := os.Getenv("ACM_BIN_" + t.name); ov != "" {
		return ov
	}
	return t.name
}

func toolByName(name string) *tool {
	if t, ok := tools[name]; ok {
		return t
	}
	return nil
}

func profilesOf(t *tool) []string {
	entries, err := os.ReadDir(filepath.Join(profDir, t.name))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func profilePath(t *tool, name string) string { return filepath.Join(profDir, t.name, name) }

func profileExists(t *tool, name string) bool {
	_, err := os.Lstat(profilePath(t, name))
	return err == nil
}

func resolvedDir(t *tool, name string) string {
	p := profilePath(t, name)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func isDefaultProfile(t *tool, name string) bool {
	d, err := filepath.EvalSymlinks(t.defaultHome)
	if err != nil {
		return false
	}
	return resolvedDir(t, name) == d
}

func loggedIn(t *tool, name string) bool {
	if _, err := os.Stat(filepath.Join(resolvedDir(t, name), t.credFile)); err == nil {
		return true
	}
	// macOS: claude guarda credenciales en el Keychain; el .claude.json del
	// perfil (escrito al hacer login) sirve de señal
	if t.name == "claude" && runtime.GOOS == "darwin" {
		if _, err := os.Stat(filepath.Join(resolvedDir(t, name), ".claude.json")); err == nil {
			return true
		}
		if isDefaultProfile(t, name) {
			if _, err := os.Stat(filepath.Join(homeDir, ".claude.json")); err == nil {
				return true
			}
		}
	}
	return false
}

// ---------- perfil activo ----------

func currentFile(t *tool) string { return filepath.Join(stateDir, t.name+".current") }

func getCurrent(t *tool) string {
	if b, err := os.ReadFile(currentFile(t)); err == nil {
		name := strings.TrimSpace(string(b))
		if profileExists(t, name) {
			return name
		}
	}
	ps := profilesOf(t)
	if len(ps) > 0 {
		return ps[0]
	}
	return ""
}

func setCurrent(t *tool, name string) {
	_ = os.MkdirAll(stateDir, 0o755)
	_ = os.WriteFile(currentFile(t), []byte(name+"\n"), 0o644)
}

// ---------- cooldown (límite alcanzado) ----------

func coolFile(t *tool, name string) string { return filepath.Join(coolDir, t.name+"."+name) }

func cooldownUntil(t *tool, name string) int64 {
	b, err := os.ReadFile(coolFile(t, name))
	if err != nil {
		return 0
	}
	u, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || u <= now() {
		_ = os.Remove(coolFile(t, name))
		return 0
	}
	return u
}

func inCooldown(t *tool, name string) bool { return cooldownUntil(t, name) > 0 }

func markCooldown(t *tool, name string, epoch int64) {
	_ = os.MkdirAll(coolDir, 0o755)
	_ = os.WriteFile(coolFile(t, name), []byte(strconv.FormatInt(epoch, 10)+"\n"), 0o644)
}

func fmtEpoch(e int64) string {
	return time.Unix(e, 0).Local().Format("15:04 (02/01)")
}

// ---------- parseo de tiempos ----------

var (
	reEpochPipe = regexp.MustCompile(`\|(\d{10,13})`)
	reDated     = regexp.MustCompile(`(?i)(?:try again at|wait until|resets(?: at)?) ([a-z]{3,9} \d{1,2}(?:st|nd|rd|th)?,? \d{4}(?: at)? \d{1,2}(?::\d{2})? ?(?:am|pm)?)`)
	reTimeOnly  = regexp.MustCompile(`(?i)(?:try again at|resets(?: at)?) (\d{1,2}(?::\d{2})? ?(?:am|pm)?)`)
	reOrdinal   = regexp.MustCompile(`(\d)(st|nd|rd|th)`)
)

var datedLayouts = []string{
	"Jan 2, 2006 3:04 PM", "Jan 2 2006 3:04 PM",
	"Jan 2, 2006 15:04", "Jan 2 2006 15:04",
}

var clockLayouts = []string{"3:04 PM", "3 PM", "3PM", "15:04"}

func normalizeDated(s string) string {
	s = reOrdinal.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, " at ", " ")
	f := strings.Fields(s)
	if len(f) > 0 {
		f[0] = strings.ToUpper(f[0][:1]) + strings.ToLower(f[0][1:])
	}
	if len(f) > 0 {
		last := strings.ToLower(f[len(f)-1])
		if last == "am" || last == "pm" {
			f[len(f)-1] = strings.ToUpper(last)
		}
	}
	return strings.Join(f, " ")
}

func normalizeClock(s string) string {
	s = strings.TrimSpace(s)
	f := strings.Fields(s)
	if len(f) > 0 {
		last := strings.ToLower(f[len(f)-1])
		if last == "am" || last == "pm" {
			f[len(f)-1] = strings.ToUpper(last)
		}
	}
	s = strings.Join(f, " ")
	// "3pm" -> "3PM"
	if m := regexp.MustCompile(`(?i)^(\d{1,2}(?::\d{2})?)(am|pm)$`).FindStringSubmatch(s); m != nil {
		s = m[1] + strings.ToUpper(m[2])
	}
	return s
}

// clockToday interpreta una hora del día y la ancla a hoy (o mañana si ya pasó).
func clockToday(s string) int64 {
	s = normalizeClock(s)
	for _, layout := range clockLayouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err != nil {
			continue
		}
		n := time.Now().Local()
		cand := time.Date(n.Year(), n.Month(), n.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		if !cand.After(n) {
			cand = cand.Add(24 * time.Hour)
		}
		return cand.Unix()
	}
	return 0
}

// parseResetEpoch extrae la hora de reinicio embebida en la salida de error:
//
//	claude -p : "Claude AI usage limit reached|1755640800"        (epoch)
//	codex     : "...try again at Dec 7th, 2026 12:39 AM."         (fecha)
//	codex     : "...try again at 11:33 PM."                       (hora)
//	interactivo: "resets 3pm" / "resets at 15:00"
func parseResetEpoch(out string) int64 {
	if m := reEpochPipe.FindStringSubmatch(out); m != nil {
		e, _ := strconv.ParseInt(m[1], 10, 64)
		if len(m[1]) >= 13 {
			e /= 1000
		}
		return e
	}
	if m := reDated.FindStringSubmatch(out); m != nil {
		s := normalizeDated(m[1])
		for _, layout := range datedLayouts {
			if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return t.Unix()
			}
		}
	}
	if m := reTimeOnly.FindStringSubmatch(out); m != nil {
		if e := clockToday(m[1]); e > 0 {
			return e
		}
	}
	return 0
}

// parseWhen: "+30m" | "+2h" | "HH:MM" | epoch -> epoch
func parseWhen(w string) int64 {
	switch {
	case strings.HasPrefix(w, "+") && strings.HasSuffix(w, "m"):
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(w, "+"), "m"))
		if err != nil {
			die("tiempo inválido: " + w)
		}
		return now() + int64(n)*60
	case strings.HasPrefix(w, "+") && strings.HasSuffix(w, "h"):
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(w, "+"), "h"))
		if err != nil {
			die("tiempo inválido: " + w)
		}
		return now() + int64(n)*3600
	case regexp.MustCompile(`^\d{1,2}:\d{2}$`).MatchString(w):
		if e := clockToday(w); e > 0 {
			return e
		}
		die("hora inválida: " + w)
	case regexp.MustCompile(`^\d{5,}$`).MatchString(w):
		e, _ := strconv.ParseInt(w, 10, 64)
		return e
	}
	die("tiempo inválido: " + w + " (usa +30m, +2h, HH:MM o epoch)")
	return 0
}

func isWhen(w string) bool {
	return regexp.MustCompile(`^(\+\d+[mh]|\d{1,2}:\d{2}|\d{5,})$`).MatchString(w)
}

// ---------- rotación de perfiles ----------

func orderedProfiles(t *tool, after bool) []string {
	ps := profilesOf(t)
	if len(ps) == 0 {
		return nil
	}
	cur := getCurrent(t)
	idx := 0
	for i, p := range ps {
		if p == cur {
			idx = i
			break
		}
	}
	if after {
		idx = (idx + 1) % len(ps)
	}
	var out []string
	for i := 0; i < len(ps); i++ {
		out = append(out, ps[(idx+i)%len(ps)])
	}
	return out
}

func nextAvailable(t *tool, after bool) (string, bool) {
	for _, p := range orderedProfiles(t, after) {
		if !loggedIn(t, p) || inCooldown(t, p) {
			continue
		}
		return p, true
	}
	return "", false
}

func reportAllLimited(t *tool) int {
	fmt.Fprintf(os.Stderr, "✖ Ninguna cuenta de %s disponible (todas al límite o sin login).\n", t.name)
	for _, p := range profilesOf(t) {
		if u := cooldownUntil(t, p); u > 0 {
			fmt.Fprintf(os.Stderr, "   · '%s' se libera a las %s\n", p, fmtEpoch(u))
		}
		if !loggedIn(t, p) {
			fmt.Fprintf(os.Stderr, "   · '%s' sin login (acm login %s %s)\n", p, t.name, p)
		}
	}
	return 75
}

// ---------- ejecución con el entorno del perfil ----------

func envFor(t *tool, profile string) []string {
	env := os.Environ()
	if !isDefaultProfile(t, profile) {
		env = append(env, t.envVar+"="+resolvedDir(t, profile))
	}
	return env
}

func runInteractive(t *tool, profile string, args []string) int {
	cmd := exec.Command(binFor(t), args...)
	cmd.Env = envFor(t, profile)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// el TUI hijo gestiona Ctrl-C desde la tty; el padre lo ignora
	signal.Ignore(os.Interrupt)
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	fmt.Fprintln(os.Stderr, "acm: "+err.Error())
	return 1
}

func runCapture(t *tool, profile string, args []string) (string, int) {
	cmd := exec.Command(binFor(t), args...)
	cmd.Env = envFor(t, profile)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	return string(out) + "\nacm: " + err.Error(), 127
}

// ---------- identidad (best-effort) ----------

var reEmail = regexp.MustCompile(`"emailAddress"\s*:\s*"([^"]+)"`)

func identityOf(t *tool, profile string) string {
	dir := resolvedDir(t, profile)
	switch t.name {
	case "claude":
		j := filepath.Join(dir, ".claude.json")
		if isDefaultProfile(t, profile) {
			j = filepath.Join(homeDir, ".claude.json")
		}
		if b, err := os.ReadFile(j); err == nil {
			if m := reEmail.FindSubmatch(b); m != nil {
				return string(m[1])
			}
		}
	case "codex":
		b, err := os.ReadFile(filepath.Join(dir, "auth.json"))
		if err != nil {
			return ""
		}
		var a struct {
			Tokens struct {
				IDToken string `json:"id_token"`
			} `json:"tokens"`
		}
		if json.Unmarshal(b, &a) != nil || a.Tokens.IDToken == "" {
			return ""
		}
		parts := strings.Split(a.Tokens.IDToken, ".")
		if len(parts) < 2 {
			return ""
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
		var p struct {
			Email string `json:"email"`
		}
		if json.Unmarshal(payload, &p) == nil {
			return p.Email
		}
	}
	return ""
}

// ---------- comandos ----------

func cmdInit() int {
	for _, name := range toolOrder {
		t := tools[name]
		link := filepath.Join(profDir, t.name, "principal")
		if _, err := os.Stat(t.defaultHome); err == nil {
			if _, err := os.Lstat(link); os.IsNotExist(err) {
				if err := os.Symlink(t.defaultHome, link); err == nil {
					fmt.Printf("✓ Login existente de %s adoptado como perfil 'principal'.\n", t.name)
				}
			}
		}
	}
	return cmdLs()
}

func cmdLs() int {
	for _, name := range toolOrder {
		t := tools[name]
		bin, err := exec.LookPath(binFor(t))
		if err != nil {
			bin = "NO ENCONTRADO"
		}
		fmt.Printf("%s  (%s)\n", t.name, bin)
		ps := profilesOf(t)
		if len(ps) == 0 {
			fmt.Printf("   (sin perfiles — crea uno: acm add %s <nombre>)\n", t.name)
			continue
		}
		cur := getCurrent(t)
		for _, p := range ps {
			mark := " "
			if p == cur {
				mark = "*"
			}
			var stat string
			switch {
			case !loggedIn(t, p):
				stat = fmt.Sprintf("sin login → acm login %s %s", t.name, p)
			case inCooldown(t, p):
				stat = "límite alcanzado, libre a las " + fmtEpoch(cooldownUntil(t, p))
			default:
				stat = "disponible"
			}
			id := identityOf(t, p)
			if id != "" {
				id = "  [" + id + "]"
			}
			fmt.Printf(" %s %-12s %s%s\n", mark, p, stat, id)
		}
	}
	return 0
}

func cmdDoctor() int {
	for _, name := range toolOrder {
		t := tools[name]
		v := "no ejecutable"
		if out, err := exec.Command(binFor(t), "--version").Output(); err == nil {
			v = strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		}
		fmt.Printf("%-7s %s\n", t.name+":", v)
	}
	fmt.Printf("acm    : v%s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	fmt.Println("estado : " + acmDir)
	fmt.Printf("cooldown por defecto: %dm\n", defaultCooldownMin)
	fmt.Println()
	return cmdLs()
}

func seedProfile(t *tool, dst string) {
	for _, f := range t.seedFiles {
		src := filepath.Join(t.defaultHome, f)
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(dst, f), b, 0o644)
	}
}

func cmdAdd(args []string) int {
	if len(args) < 2 {
		die("uso: acm add <claude|codex> <nombre>")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	name := args[1]
	if profileExists(t, name) {
		die("el perfil '" + name + "' ya existe")
	}
	dst := profilePath(t, name)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		die(err.Error())
	}
	seedProfile(t, dst)
	fmt.Printf("✓ Perfil '%s' creado para %s (config heredada del perfil por defecto).\n", name, t.name)
	return cmdLogin(args)
}

func cmdLogin(args []string) int {
	if len(args) < 2 {
		die("uso: acm login <claude|codex> <perfil>")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	name := args[1]
	if !profileExists(t, name) {
		die("no existe el perfil '" + name + "' (créalo: acm add " + t.name + " " + name + ")")
	}
	var rc int
	switch t.name {
	case "claude":
		fmt.Printf("→ Abriendo Claude Code con el perfil '%s'.\n", name)
		fmt.Println("  Si ya hay sesión de otra cuenta, usa /login dentro; termina con /exit.")
		rc = runInteractive(t, name, nil)
	case "codex":
		// device-auth: el flujo de navegador (localhost:1455) suele fallar en WSL
		rc = runInteractive(t, name, []string{"login", "--device-auth"})
		if rc != 0 {
			rc = runInteractive(t, name, []string{"login"})
		}
	}
	// tras el login, la cuenta vuelve a estar operativa
	_ = os.Remove(coolFile(t, name))
	return rc
}

func cmdUse(args []string) int {
	if len(args) < 2 {
		die("uso: acm use <claude|codex> <perfil>")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	if !profileExists(t, args[1]) {
		die("no existe el perfil '" + args[1] + "'")
	}
	setCurrent(t, args[1])
	fmt.Printf("✓ Perfil activo de %s: %s\n", t.name, args[1])
	return 0
}

func cmdNext(args []string) int {
	if len(args) < 1 {
		die("uso: acm next <claude|codex>")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	p, ok := nextAvailable(t, true)
	if !ok {
		return reportAllLimited(t)
	}
	setCurrent(t, p)
	fmt.Printf("✓ Perfil activo de %s: %s\n", t.name, p)
	return 0
}

func cmdLimit(args []string) int {
	if len(args) < 1 {
		die("uso: acm limit <tool> [perfil] [+1h|HH:MM|epoch]")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	name := getCurrent(t)
	when := fmt.Sprintf("+%dm", defaultCooldownMin)
	rest := args[1:]
	if len(rest) >= 1 {
		if isWhen(rest[0]) {
			when = rest[0]
		} else {
			name = rest[0]
			if len(rest) >= 2 {
				when = rest[1]
			}
		}
	}
	if !profileExists(t, name) {
		die("no existe el perfil '" + name + "'")
	}
	e := parseWhen(when)
	markCooldown(t, name, e)
	fmt.Printf("✓ [%s:%s] marcada al límite hasta las %s\n", t.name, name, fmtEpoch(e))
	if n, ok := nextAvailable(t, false); ok {
		setCurrent(t, n)
		fmt.Println("→ Perfil activo ahora: " + n)
	} else {
		_ = reportAllLimited(t)
	}
	return 0
}

func cmdFree(args []string) int {
	if len(args) < 2 {
		die("uso: acm free <tool> <perfil>")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	_ = os.Remove(coolFile(t, args[1]))
	fmt.Printf("✓ [%s:%s] disponible de nuevo\n", t.name, args[1])
	return 0
}

func cmdRun(args []string) int {
	if len(args) < 1 {
		die("uso: acm run <claude|codex> [args...]")
	}
	t := toolByName(args[0])
	if t == nil {
		die("herramienta desconocida: " + args[0])
	}
	rest := args[1:]
	total := len(profilesOf(t))
	for tries := 0; tries < total; tries++ {
		p, ok := nextAvailable(t, false)
		if !ok {
			break
		}
		out, code := runCapture(t, p, append(append([]string{}, t.execArgs...), rest...))
		// codex puede salir con código 0 aunque falle por límite: valida también la salida
		if code == 0 && !t.limitRe.MatchString(out) && !t.authRe.MatchString(out) {
			setCurrent(t, p)
			fmt.Print(out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Println()
			}
			return 0
		}
		if t.limitRe.MatchString(out) {
			reset := parseResetEpoch(out)
			if reset <= now() {
				reset = now() + int64(defaultCooldownMin)*60
			}
			markCooldown(t, p, reset)
			fmt.Fprintf(os.Stderr, "⚠ [%s:%s] límite de uso alcanzado (libre a las %s). Probando siguiente cuenta...\n", t.name, p, fmtEpoch(reset))
			continue
		}
		if t.authRe.MatchString(out) {
			markCooldown(t, p, now()+15*60)
			fmt.Fprintf(os.Stderr, "⚠ [%s:%s] sesión caducada o sin autenticación. Reactívala con: acm login %s %s\n", t.name, p, t.name, p)
			continue
		}
		fmt.Fprint(os.Stderr, out)
		if !strings.HasSuffix(out, "\n") {
			fmt.Fprintln(os.Stderr)
		}
		return code
	}
	return reportAllLimited(t)
}

func cmdLaunch(toolName string, args []string) int {
	t := toolByName(toolName)
	p, ok := nextAvailable(t, false)
	if !ok {
		return reportAllLimited(t)
	}
	setCurrent(t, p)
	id := identityOf(t, p)
	if id != "" {
		id = " [" + id + "]"
	}
	fmt.Fprintf(os.Stderr, "→ %s · perfil '%s'%s\n", t.name, p, id)
	return runInteractive(t, p, args)
}

// ---------- quota ----------

func cmdQuota(args []string) int {
	var sel string
	raw := false
	for _, a := range args {
		switch {
		case a == "--raw":
			raw = true
		case toolByName(a) != nil:
			sel = a
		default:
			die("herramienta desconocida: " + a)
		}
	}
	names := toolOrder
	if sel != "" {
		names = []string{sel}
	}
	for _, name := range names {
		t := tools[name]
		fmt.Println(t.name + ":")
		ps := profilesOf(t)
		if len(ps) == 0 {
			fmt.Println("   (sin perfiles)")
			continue
		}
		for _, p := range ps {
			if !loggedIn(t, p) {
				fmt.Printf("  %-12s sin login\n", p)
				continue
			}
			fmt.Printf("  %-12s ", p)
			switch t.name {
			case "claude":
				fmt.Println(claudeQuota(t, p, raw))
			case "codex":
				fmt.Println(codexQuota(t, p, raw))
			}
		}
	}
	return 0
}

// claudeCredJSON lee las credenciales del perfil: archivo, o Keychain en macOS
// (builds recientes añaden sufijo sha256(dir)[0:8] al nombre del item).
func claudeCredJSON(dir string) ([]byte, bool) {
	if b, err := os.ReadFile(filepath.Join(dir, ".credentials.json")); err == nil {
		return b, true
	}
	if runtime.GOOS == "darwin" {
		sum := sha256.Sum256([]byte(dir))
		suffix := hex.EncodeToString(sum[:])[:8]
		for _, svc := range []string{"Claude Code-credentials-" + suffix, "Claude Code-credentials"} {
			out, err := exec.Command("security", "find-generic-password", "-w", "-s", svc).Output()
			if err == nil && len(bytes.TrimSpace(out)) > 0 {
				return bytes.TrimSpace(out), true
			}
		}
	}
	return nil, false
}

type claudeLimitEntry struct {
	Kind     string          `json:"kind"`
	Group    string          `json:"group"`
	Percent  *float64        `json:"percent"`
	Severity string          `json:"severity"`
	ResetsAt json.RawMessage `json:"resets_at"`
	Scope    *struct {
		Model *struct {
			DisplayName string `json:"display_name"`
		} `json:"model"`
	} `json:"scope"`
}

func fmtResetRaw(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "?"
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.Local().Format("15:04 (02/01)")
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.Local().Format("15:04 (02/01)")
		}
		return s
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		e := int64(n)
		if e > 1e12 {
			e /= 1000
		}
		return fmtEpoch(e)
	}
	return "?"
}

func claudeQuota(t *tool, profile string, raw bool) string {
	cred, ok := claudeCredJSON(resolvedDir(t, profile))
	if !ok {
		return "sin credenciales legibles"
	}
	var c struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
			ExpiresAt   int64  `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if json.Unmarshal(cred, &c) != nil || c.ClaudeAiOauth.AccessToken == "" {
		return "sin credenciales legibles"
	}
	if c.ClaudeAiOauth.ExpiresAt > 0 && c.ClaudeAiOauth.ExpiresAt < time.Now().UnixMilli() {
		return "token caducado — reactiva con: acm login claude " + profile
	}
	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return "error: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+c.ClaudeAiOauth.AccessToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "error de red: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if raw {
		var pretty bytes.Buffer
		if json.Indent(&pretty, body, "", " ") == nil {
			return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, pretty.String())
		}
		return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body))
	}
	switch resp.StatusCode {
	case 401:
		return "sesión caducada (401) — reactiva con: acm login claude " + profile
	case 429:
		return "endpoint saturado o token inválido (429) — reintenta en unos minutos"
	}
	// fuente primaria: array moderno `limits` (igual que /usage en la TUI,
	// incluye ventanas por modelo, p.ej. scope.model.display_name = "Fable")
	var doc struct {
		Limits []claudeLimitEntry `json:"limits"`
	}
	if json.Unmarshal(body, &doc) == nil && len(doc.Limits) > 0 {
		var parts []string
		for _, e := range doc.Limits {
			if e.Percent == nil {
				continue
			}
			label := e.Kind
			switch {
			case e.Kind == "session":
				label = "5h"
			case e.Kind == "weekly_all" || e.Group == "weekly":
				label = "semana"
			}
			if e.Scope != nil && e.Scope.Model != nil && e.Scope.Model.DisplayName != "" {
				label += " " + strings.ToLower(e.Scope.Model.DisplayName)
			}
			warn := ""
			if e.Severity != "" && e.Severity != "normal" {
				warn = " ⚠"
			}
			parts = append(parts, fmt.Sprintf("%s: %d%% usado%s (reinicia %s)", label, int(*e.Percent+0.5), warn, fmtResetRaw(e.ResetsAt)))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
	}
	// fallback legacy: claves five_hour / seven_day con utilization/resets_at
	var generic map[string]json.RawMessage
	if json.Unmarshal(body, &generic) == nil {
		labels := map[string]string{"five_hour": "5h", "seven_day": "semana", "seven_day_opus": "semana opus", "seven_day_sonnet": "semana sonnet"}
		var keys []string
		for k := range generic {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			var v struct {
				Utilization *float64        `json:"utilization"`
				ResetsAt    json.RawMessage `json:"resets_at"`
			}
			if json.Unmarshal(generic[k], &v) != nil || v.Utilization == nil {
				continue
			}
			if *v.Utilization == 0 && (len(v.ResetsAt) == 0 || string(v.ResetsAt) == "null") {
				continue // ventana inactiva
			}
			label := labels[k]
			if label == "" {
				label = k
			}
			parts = append(parts, fmt.Sprintf("%s: %d%% usado (reinicia %s)", label, int(*v.Utilization+0.5), fmtResetRaw(v.ResetsAt)))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
		if em, ok := generic["error"]; ok {
			var e struct{ Type, Message string }
			_ = json.Unmarshal(em, &e)
			return fmt.Sprintf("HTTP %d: %s — %s", resp.StatusCode, e.Type, e.Message)
		}
	}
	return fmt.Sprintf("HTTP %d formato desconocido — mira: acm quota claude --raw", resp.StatusCode)
}

// codexQuota consulta `codex app-server` por JSON-RPC (account/rateLimits/read).
func codexQuota(t *tool, profile string, raw bool) string {
	cmd := exec.Command(binFor(t), "app-server")
	cmd.Env = envFor(t, profile)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "error: " + err.Error()
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "error: " + err.Error()
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return "no se pudo lanzar codex app-server: " + err.Error()
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	timer := time.AfterFunc(15*time.Second, func() { _ = cmd.Process.Kill() })
	defer timer.Stop()

	send := func(v any) {
		b, _ := json.Marshal(v)
		_, _ = stdin.Write(append(b, '\n'))
	}
	type rpcMsg struct {
		ID     *int            `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	send(map[string]any{"jsonrpc": "2.0", "id": 0, "method": "initialize",
		"params": map[string]any{"clientInfo": map[string]any{"name": "acm", "title": "acm", "version": version}}})

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m rpcMsg
		if json.Unmarshal([]byte(line), &m) != nil || m.ID == nil {
			continue
		}
		switch *m.ID {
		case 0:
			if m.Error != nil {
				return "error init: " + m.Error.Message
			}
			send(map[string]any{"jsonrpc": "2.0", "method": "initialized"})
			send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "account/rateLimits/read", "params": map[string]any{}})
		case 1:
			if m.Error != nil {
				return "error: " + m.Error.Message
			}
			if raw {
				var pretty bytes.Buffer
				if json.Indent(&pretty, m.Result, "", " ") == nil {
					return pretty.String()
				}
				return string(m.Result)
			}
			var r struct {
				RateLimits struct {
					PlanType string `json:"planType"`
					Primary  *struct {
						UsedPercent        float64 `json:"usedPercent"`
						WindowDurationMins int     `json:"windowDurationMins"`
						ResetsAt           int64   `json:"resetsAt"`
					} `json:"primary"`
					Secondary *struct {
						UsedPercent        float64 `json:"usedPercent"`
						WindowDurationMins int     `json:"windowDurationMins"`
						ResetsAt           int64   `json:"resetsAt"`
					} `json:"secondary"`
					Credits *struct {
						HasCredits bool   `json:"hasCredits"`
						Balance    string `json:"balance"`
					} `json:"credits"`
				} `json:"rateLimits"`
			}
			if json.Unmarshal(m.Result, &r) != nil {
				return "respuesta ilegible (usa --raw)"
			}
			wname := func(mins int) string {
				switch mins {
				case 300:
					return "5h"
				case 10080:
					return "semana"
				}
				return fmt.Sprintf("%dm", mins)
			}
			var parts []string
			if r.RateLimits.PlanType != "" {
				parts = append(parts, "plan "+r.RateLimits.PlanType)
			}
			for _, l := range []*struct {
				UsedPercent        float64 `json:"usedPercent"`
				WindowDurationMins int     `json:"windowDurationMins"`
				ResetsAt           int64   `json:"resetsAt"`
			}{r.RateLimits.Primary, r.RateLimits.Secondary} {
				if l == nil {
					continue
				}
				parts = append(parts, fmt.Sprintf("%s: %d%% usado (reinicia %s)", wname(l.WindowDurationMins), int(l.UsedPercent+0.5), fmtEpoch(l.ResetsAt)))
			}
			if cr := r.RateLimits.Credits; cr != nil && cr.HasCredits {
				parts = append(parts, "créditos: "+cr.Balance)
			}
			if len(parts) == 0 {
				return "sin datos (prueba --raw)"
			}
			return strings.Join(parts, " · ")
		}
	}
	return "timeout o cierre inesperado consultando codex app-server"
}

// ---------- ayuda y main ----------

func usage() {
	fmt.Print(`acm — gestor de cuentas para Claude Code y Codex CLI (v` + version + `)

  acm init                    adopta los logins actuales como perfil "principal"
  acm ls                      lista perfiles, cuenta activa (*) y límites
  acm doctor                  versiones + estado
  acm add <tool> <perfil>     crea un perfil nuevo y abre su login
  acm login <tool> <perfil>   (re)abre el login de un perfil
  acm use <tool> <perfil>     fija el perfil activo
  acm next <tool>             rota al siguiente perfil disponible
  acm limit <tool> [perfil] [+1h|HH:MM]   marca límite a mano (p.ej. lo viste en la TUI)
  acm free <tool> <perfil>    quita la marca de límite
  acm quota [tool] [--raw]    cuota restante por cuenta SIN gastar tokens
  acm run <tool> [args...]    no-interactivo con failover (claude -p / codex exec)
  acm <tool> [args...]        interactivo en el primer perfil disponible

tools: claude | codex
El failover automático aplica a 'acm run'; en sesiones interactivas usa
'acm limit' + volver a lanzar 'acm <tool>' cuando la TUI anuncie el límite.
`)
}

func main() {
	initGlobals()
	if len(os.Args) < 2 {
		usage()
		return
	}
	cmd, rest := os.Args[1], os.Args[2:]
	var rc int
	switch cmd {
	case "init":
		rc = cmdInit()
	case "ls":
		rc = cmdLs()
	case "doctor":
		rc = cmdDoctor()
	case "add":
		rc = cmdAdd(rest)
	case "login":
		rc = cmdLogin(rest)
	case "use":
		rc = cmdUse(rest)
	case "next":
		rc = cmdNext(rest)
	case "limit":
		rc = cmdLimit(rest)
	case "free":
		rc = cmdFree(rest)
	case "run":
		rc = cmdRun(rest)
	case "quota":
		rc = cmdQuota(rest)
	case "claude", "codex":
		rc = cmdLaunch(cmd, rest)
	case "help", "-h", "--help":
		usage()
	case "version", "--version", "-v":
		fmt.Println("acm v" + version)
	default:
		die("comando desconocido: " + cmd + " (mira: acm help)")
	}
	os.Exit(rc)
}
