#!/usr/bin/env bash
# acm — gestor de cuentas para agentes CLI (Claude Code y Codex CLI)
#
# Cada "perfil" es un directorio de configuración aislado con su propio login:
#   claude : CLAUDE_CONFIG_DIR  (credenciales en <dir>/.credentials.json)
#   codex  : CODEX_HOME         (credenciales en <dir>/auth.json)
#
# El perfil "principal" es un symlink al directorio por defecto (~/.claude, ~/.codex),
# y en ese caso NO se exporta la variable (en claude, exportarla movería ~/.claude.json).
#
# Failover: `acm run <tool> ...` ejecuta en modo no interactivo (claude -p / codex exec);
# si la salida indica límite de uso agotado, marca la cuenta en espera (cooldown) y
# reintenta con la siguiente. `acm <tool>` lanza interactivo en el primer perfil libre.
#
# Plataformas: Linux/WSL (probado) y macOS (best-effort). En macOS:
#   - fechas: usa gdate si existe (brew install coreutils, recomendado) o date BSD;
#   - claude guarda credenciales en el Keychain (item "Claude Code-credentials[-hash]"),
#     acm las lee vía `security` para `quota`; el aislamiento por CLAUDE_CONFIG_DIR
#     depende del build de Claude Code (verifica emails con `acm ls`);
#   - codex usa auth.json en archivo salvo que config tenga cli_auth_credentials_store=keyring.
set -u

ACM_DIR="${ACM_DIR:-$HOME/.acm}"
PROF_DIR="$ACM_DIR/profiles"
STATE_DIR="$ACM_DIR/state"
COOL_DIR="$STATE_DIR/cooldown"
DEFAULT_COOLDOWN_MIN="${ACM_DEFAULT_COOLDOWN_MIN:-60}"
TOOLS="claude codex"

mkdir -p "$COOL_DIR"
for _t in $TOOLS; do mkdir -p "$PROF_DIR/$_t"; done

# brew (linuxbrew/homebrew) puede no estar en PATH en shells no interactivos
for _d in /home/linuxbrew/.linuxbrew/bin /opt/homebrew/bin /usr/local/bin; do
  [ -d "$_d" ] && case ":$PATH:" in *":$_d:"*) ;; *) PATH="$PATH:$_d";; esac
done

# ---------- plataforma ----------
OS_NAME=$(uname -s 2>/dev/null || echo Linux)
# motor de fechas: gdate (coreutils) > date GNU > date BSD (macOS sin coreutils)
if command -v gdate >/dev/null 2>&1; then DATE_BIN=gdate; DATE_GNU=1
elif date -d @0 +%s >/dev/null 2>&1; then DATE_BIN=date; DATE_GNU=1
else DATE_BIN=date; DATE_GNU=0; fi

die() { echo "acm: $*" >&2; exit 2; }
now() { date +%s; }

# ---------- tabla de herramientas ----------
tool_valid()  { case "$1" in claude|codex) return 0;; *) return 1;; esac; }
tool_envvar() { case "$1" in claude) echo CLAUDE_CONFIG_DIR;; codex) echo CODEX_HOME;; esac; }
tool_bin() {
  # ACM_BIN_claude / ACM_BIN_codex permiten sustituir el binario (tests)
  local ov=""
  case "$1" in
    claude) ov="${ACM_BIN_claude:-}";;
    codex)  ov="${ACM_BIN_codex:-}";;
  esac
  [ -n "$ov" ] && echo "$ov" || echo "$1"
}
tool_default_home() { case "$1" in claude) echo "$HOME/.claude";; codex) echo "$HOME/.codex";; esac; }
tool_credfile()     { case "$1" in claude) echo ".credentials.json";; codex) echo "auth.json";; esac; }
# patrón (grep -Ei) en stdout+stderr que indica "límite de uso agotado"
# (anclado a las frases reales de error para no confundirlo con texto de una respuesta)
tool_limit_re() {
  case "$1" in
    claude) echo "usage limit reached|hit your (session|weekly|usage|opus).{0,4}limit|(session|weekly|5-hour) limit reached";;
    codex)  echo "hit your usage limit|usage_limit_reached|rate_limit_reached|workspace_[a-z_]*usage_limit";;
  esac
}
# patrón que indica "sesión caducada / sin autenticación" (rotamos sin marcar límite largo)
tool_auth_re() {
  case "$1" in
    claude) echo "failed to authenticate|oauth (session|access )?token.{0,20}(expired|revoked)|re-authenticate|please run /login|not logged in|invalid bearer token|authentication_error";;
    codex)  echo "not logged in|login required|token refresh failed|401 unauthorized|reauthenticate";;
  esac
}

# ---------- perfiles ----------
abspath() { # readlink -f portable (macOS viejos no lo tienen)
  readlink -f "$1" 2>/dev/null && return 0
  local t="$1"
  [ -L "$t" ] && t=$(readlink "$t")
  (cd "$t" 2>/dev/null && pwd -P)
}
profiles_of()    { ls -1 "$PROF_DIR/$1" 2>/dev/null | sort; }
profile_dir()    { echo "$PROF_DIR/$1/$2"; }
profile_exists() { [ -e "$(profile_dir "$1" "$2")" ]; }
resolved_dir()   { abspath "$(profile_dir "$1" "$2")"; }
is_default_profile() {
  [ "$(resolved_dir "$1" "$2")" = "$(abspath "$(tool_default_home "$1")" 2>/dev/null)" ]
}
logged_in() {
  [ -f "$(resolved_dir "$1" "$2")/$(tool_credfile "$1")" ] && return 0
  # macOS: claude guarda credenciales en el Keychain, no en archivo;
  # el .claude.json del perfil (escrito al hacer login) sirve de señal
  if [ "$1" = claude ] && [ "$OS_NAME" = Darwin ]; then
    local d; d=$(resolved_dir claude "$2")
    [ -f "$d/.claude.json" ] && return 0
    is_default_profile claude "$2" && [ -f "$HOME/.claude.json" ] && return 0
  fi
  return 1
}
# JSON de credenciales de claude: archivo, o Keychain en macOS
# (builds recientes añaden sufijo sha256(dir)[0:8] al nombre del item)
claude_cred_json() {
  local dir="$1" h
  local f="$dir/.credentials.json"
  [ -f "$f" ] && { cat "$f"; return 0; }
  if [ "$OS_NAME" = Darwin ] && command -v security >/dev/null 2>&1; then
    h=$(printf '%s' "$dir" | shasum -a 256 2>/dev/null | cut -c1-8)
    [ -n "$h" ] && security find-generic-password -w -s "Claude Code-credentials-$h" 2>/dev/null && return 0
    security find-generic-password -w -s "Claude Code-credentials" 2>/dev/null && return 0
  fi
  return 1
}

current_file() { echo "$STATE_DIR/$1.current"; }
get_current() {
  local f; f=$(current_file "$1")
  if [ -f "$f" ] && profile_exists "$1" "$(cat "$f")"; then cat "$f"; else profiles_of "$1" | head -1; fi
}
set_current() { echo "$2" > "$(current_file "$1")"; }

# ---------- cooldown (límite alcanzado) ----------
cool_file() { echo "$COOL_DIR/$1.$2"; }
cooldown_until() {
  local f u; f=$(cool_file "$1" "$2")
  [ -f "$f" ] || { echo 0; return; }
  u=$(cat "$f" 2>/dev/null || echo 0)
  case "$u" in ''|*[!0-9]*) u=0;; esac
  if [ "$u" -le "$(now)" ]; then rm -f "$f"; echo 0; else echo "$u"; fi
}
in_cooldown()   { [ "$(cooldown_until "$1" "$2")" -gt 0 ]; }
mark_cooldown() { echo "$3" > "$(cool_file "$1" "$2")"; }

# ---------- fechas portables (GNU date / gdate / BSD date de macOS) ----------
fmt_epoch() {
  if [ "$DATE_GNU" = 1 ]; then "$DATE_BIN" -d "@$1" '+%H:%M (%d/%m)' 2>/dev/null || echo "$1"
  else date -r "$1" '+%H:%M (%d/%m)' 2>/dev/null || echo "$1"; fi
}

epoch_today_time() { # "11:33 pm" | "3:00 pm" | "15:00" -> epoch de hoy (0 si falla)
  local s="$1" e S
  if [ "$DATE_GNU" = 1 ]; then
    "$DATE_BIN" -d "today $s" +%s 2>/dev/null || echo 0
  else
    S=$(printf '%s' "$s" | sed 's/ am$/ AM/; s/ pm$/ PM/')
    e=$(date -j -f '%Y-%m-%d %I:%M %p' "$(date +%Y-%m-%d) $S" +%s 2>/dev/null) \
      || e=$(date -j -f '%Y-%m-%d %H:%M' "$(date +%Y-%m-%d) $s" +%s 2>/dev/null) \
      || e=0
    echo "$e"
  fi
}

epoch_from_dated() { # "aug 7, 2026 12:39 am" -> epoch (0 si falla)
  local s="$1" f e
  if [ "$DATE_GNU" = 1 ]; then "$DATE_BIN" -d "$s" +%s 2>/dev/null || echo 0; return; fi
  s=$(printf '%s' "$s" | awk '{ $1=toupper(substr($1,1,1)) substr($1,2) } 1' | sed 's/ am$/ AM/; s/ pm$/ PM/')
  for f in '%b %d, %Y %I:%M %p' '%b %d %Y %I:%M %p' '%b %d, %Y %H:%M' '%b %d %Y %H:%M'; do
    e=$(date -j -f "$f" "$s" +%s 2>/dev/null) && { echo "$e"; return; }
  done
  echo 0
}

# hora de reinicio embebida en la salida:
#   claude -p : "Claude AI usage limit reached|1755640800"       (epoch)
#   codex     : "...try again at 11:33 PM."                      (hora local)
#   interactivo: "resets 3pm" / "resets at 15:00"
parse_reset_epoch() {
  local out="$1" e t
  e=$(printf '%s' "$out" | grep -oE '\|[0-9]{10,13}' | head -1 | tr -d '|')
  if [ -n "$e" ]; then
    [ "${#e}" -ge 13 ] && e=$((e/1000))
    echo "$e"; return
  fi
  # fecha completa: "try again at Aug 7th, 2026 12:39 AM" / "wait until Aug 6, 2026 at 12:43 PM"
  t=$(printf '%s' "$out" | grep -oiE '(try again at|wait until|resets( at)?) [a-z]{3,9} [0-9]{1,2}(st|nd|rd|th)?,? [0-9]{4}( at)? [0-9]{1,2}(:[0-9]{2})? ?(am|pm)?' \
      | head -1 | tr '[:upper:]' '[:lower:]' \
      | sed -E 's/^(try again at|wait until|resets( at)?) //; s/([0-9])(st|nd|rd|th)/\1/g; s/ at / /')
  if [ -n "$t" ]; then
    e=$(epoch_from_dated "$t")
    if [ "$e" -gt 0 ]; then echo "$e"; return; fi
  fi
  # solo hora: "try again at 11:33 PM" / "resets 3pm"
  t=$(printf '%s' "$out" | grep -oiE '(try again at|resets( at)?) [0-9]{1,2}(:[0-9]{2})? ?(am|pm)?' \
      | head -1 | tr '[:upper:]' '[:lower:]' \
      | sed -E 's/^(try again at|resets( at)?) //; s/^([0-9]{1,2}) ?(am|pm)$/\1:00 \2/')
  if [ -n "$t" ]; then
    e=$(epoch_today_time "$t")
    if [ "$e" -gt 0 ] && [ "$e" -le "$(now)" ]; then e=$((e+86400)); fi
    echo "$e"; return
  fi
  echo 0
}

parse_when() { # "+30m" | "+2h" | "HH:MM" | epoch  ->  epoch
  local w="$1" n e
  case "$w" in
    +*m) n=${w#+}; n=${n%m}; echo $(( $(now) + n*60 ));;
    +*h) n=${w#+}; n=${n%h}; echo $(( $(now) + n*3600 ));;
    [0-9]:[0-9][0-9]|[0-9][0-9]:[0-9][0-9])
      e=$(epoch_today_time "$w")
      [ "$e" -gt 0 ] || die "hora inválida: $w"
      [ "$e" -le "$(now)" ] && e=$((e+86400))
      echo "$e";;
    [0-9][0-9][0-9][0-9][0-9]*) echo "$w";;
    *) die "tiempo inválido: $w (usa +30m, +2h, HH:MM o epoch)";;
  esac
}

is_when() { case "$1" in +*m|+*h|[0-9]:[0-9][0-9]|[0-9][0-9]:[0-9][0-9]|[0-9][0-9][0-9][0-9][0-9]*) return 0;; *) return 1;; esac; }

# ---------- ejecución con el entorno del perfil ----------
with_env_run() { # tool perfil cmd args...
  local tool="$1" prof="$2"; shift 2
  local dir; dir=$(resolved_dir "$tool" "$prof")
  if is_default_profile "$tool" "$prof"; then
    "$@"
  else
    env "$(tool_envvar "$tool")=$dir" "$@"
  fi
}

identity_of() { # best-effort: email de la cuenta del perfil
  local tool="$1" prof="$2" dir j a
  dir=$(resolved_dir "$tool" "$prof")
  case "$tool" in
    claude)
      j="$dir/.claude.json"
      is_default_profile claude "$prof" && j="$HOME/.claude.json"
      [ -f "$j" ] && grep -o '"emailAddress"[[:space:]]*:[[:space:]]*"[^"]*"' "$j" 2>/dev/null | head -1 | awk -F'"' '{print $(NF-1)}'
      ;;
    codex)
      a="$dir/auth.json"
      [ -f "$a" ] && command -v node >/dev/null 2>&1 && node -e '
        try{const fs=require("fs");const a=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));
        const t=(a.tokens||{}).id_token;if(!t)process.exit(0);
        const p=JSON.parse(Buffer.from(t.split(".")[1],"base64url").toString());
        console.log(p.email||"");}catch(e){}' "$a" 2>/dev/null
      ;;
  esac
}

# rotación empezando en el perfil actual (o después de él, con arg2=after)
ordered_profiles() {
  local tool="$1" mode="${2:-from}" cur
  cur=$(get_current "$tool")
  profiles_of "$tool" | awk -v c="$cur" -v m="$mode" '
    {a[NR]=$0; if($0==c)i=NR}
    END{
      if(NR==0)exit; if(!i)i=1;
      s=(m=="after")?1:0;
      for(j=0;j<NR;j++){k=((i-1+s+j)%NR)+1; print a[k]}
    }'
}

next_available() { # tool [after] -> nombre de perfil o rc=1
  local tool="$1" mode="${2:-from}" p
  for p in $(ordered_profiles "$tool" "$mode"); do
    logged_in "$tool" "$p" || continue
    in_cooldown "$tool" "$p" && continue
    echo "$p"; return 0
  done
  return 1
}

report_all_limited() {
  local tool="$1" p u
  echo "✖ Ninguna cuenta de $tool disponible (todas al límite o sin login)." >&2
  for p in $(profiles_of "$tool"); do
    u=$(cooldown_until "$tool" "$p")
    [ "$u" -gt 0 ] && echo "   · '$p' se libera a las $(fmt_epoch "$u")" >&2
    logged_in "$tool" "$p" || echo "   · '$p' sin login (acm login $tool $p)" >&2
  done
  return 75
}

# ---------- comandos ----------
cmd_init() {
  local t d p
  for t in $TOOLS; do
    d=$(tool_default_home "$t"); p="$PROF_DIR/$t/principal"
    if [ -d "$d" ] && [ ! -e "$p" ]; then
      ln -s "$d" "$p"
      echo "✓ Login existente de $t adoptado como perfil 'principal'."
    fi
  done
  cmd_ls
}

cmd_ls() {
  local t p cur u mark stat id bin
  for t in $TOOLS; do
    bin=$(command -v "$(tool_bin "$t")" 2>/dev/null || echo "NO ENCONTRADO")
    printf '%s  (%s)\n' "$t" "$bin"
    cur=$(get_current "$t")
    if [ -z "$(profiles_of "$t")" ]; then
      printf '   (sin perfiles — crea uno: acm add %s <nombre>)\n' "$t"; continue
    fi
    for p in $(profiles_of "$t"); do
      mark=' '; [ "$p" = "$cur" ] && mark='*'
      u=$(cooldown_until "$t" "$p")
      if ! logged_in "$t" "$p"; then stat="sin login → acm login $t $p"
      elif [ "$u" -gt 0 ]; then stat="límite alcanzado, libre a las $(fmt_epoch "$u")"
      else stat="disponible"; fi
      id=$(identity_of "$t" "$p" 2>/dev/null || true)
      printf ' %s %-12s %s%s\n' "$mark" "$p" "$stat" "${id:+  [$id]}"
    done
  done
}

seed_profile() { # copia config no-secreta desde el home por defecto al perfil nuevo
  local tool="$1" dst="$2" src f
  src=$(tool_default_home "$tool")
  [ -d "$src" ] || return 0
  case "$tool" in
    claude) for f in settings.json CLAUDE.md; do [ -f "$src/$f" ] && cp "$src/$f" "$dst/"; done;;
    codex)  for f in config.toml AGENTS.md hooks.json; do [ -f "$src/$f" ] && cp "$src/$f" "$dst/"; done;;
  esac
  return 0
}

cmd_add() {
  local tool="${1:-}" name="${2:-}"
  [ -n "$tool" ] && [ -n "$name" ] || die "uso: acm add <claude|codex> <nombre>"
  tool_valid "$tool" || die "herramienta desconocida: $tool"
  profile_exists "$tool" "$name" && die "el perfil '$name' ya existe"
  mkdir -p "$(profile_dir "$tool" "$name")"
  seed_profile "$tool" "$(profile_dir "$tool" "$name")"
  echo "✓ Perfil '$name' creado para $tool (config heredada del perfil por defecto)."
  cmd_login "$tool" "$name"
}

cmd_login() {
  local tool="${1:-}" name="${2:-}"
  [ -n "$tool" ] && [ -n "$name" ] || die "uso: acm login <claude|codex> <perfil>"
  profile_exists "$tool" "$name" || die "no existe el perfil '$name' (créalo: acm add $tool $name)"
  case "$tool" in
    claude)
      echo "→ Abriendo Claude Code con el perfil '$name'."
      echo "  Si ya hay sesión de otra cuenta, usa /login dentro; termina con /exit."
      with_env_run claude "$name" "$(tool_bin claude)"
      ;;
    codex)
      # device-auth: el flujo de navegador (localhost:1455) suele fallar en WSL
      with_env_run codex "$name" "$(tool_bin codex)" login --device-auth \
        || with_env_run codex "$name" "$(tool_bin codex)" login
      ;;
  esac
  # tras el login, la cuenta vuelve a estar operativa
  rm -f "$(cool_file "$tool" "$name")"
}

cmd_use() {
  local tool="${1:-}" name="${2:-}"
  [ -n "$tool" ] && [ -n "$name" ] || die "uso: acm use <claude|codex> <perfil>"
  profile_exists "$tool" "$name" || die "no existe el perfil '$name'"
  set_current "$tool" "$name"
  echo "✓ Perfil activo de $tool: $name"
}

cmd_next() {
  local tool="${1:-}"; [ -n "$tool" ] || die "uso: acm next <claude|codex>"
  tool_valid "$tool" || die "herramienta desconocida: $tool"
  local p
  p=$(next_available "$tool" after) || { report_all_limited "$tool"; return 75; }
  set_current "$tool" "$p"
  echo "✓ Perfil activo de $tool: $p"
}

cmd_limit() {
  local tool="${1:-}"; [ -n "$tool" ] || die "uso: acm limit <tool> [perfil] [+1h|HH:MM|epoch]"
  tool_valid "$tool" || die "herramienta desconocida: $tool"; shift
  local name when
  name=$(get_current "$tool"); when="+${DEFAULT_COOLDOWN_MIN}m"
  if [ $# -ge 1 ]; then
    if is_when "$1"; then when="$1"; else name="$1"; [ $# -ge 2 ] && when="$2"; fi
  fi
  profile_exists "$tool" "$name" || die "no existe el perfil '$name'"
  local e; e=$(parse_when "$when")
  mark_cooldown "$tool" "$name" "$e"
  echo "✓ [$tool:$name] marcada al límite hasta las $(fmt_epoch "$e")"
  local n
  if n=$(next_available "$tool"); then
    set_current "$tool" "$n"
    echo "→ Perfil activo ahora: $n"
  else
    report_all_limited "$tool" || true
  fi
}

cmd_free() {
  local tool="${1:-}" name="${2:-}"
  [ -n "$tool" ] && [ -n "$name" ] || die "uso: acm free <tool> <perfil>"
  rm -f "$(cool_file "$tool" "$name")"
  echo "✓ [$tool:$name] disponible de nuevo"
}

cmd_run() {
  local tool="${1:-}"; [ -n "$tool" ] || die "uso: acm run <claude|codex> [args...]"
  tool_valid "$tool" || die "herramienta desconocida: $tool"; shift
  local re are out code p reset tries total
  re=$(tool_limit_re "$tool"); are=$(tool_auth_re "$tool")
  total=$(profiles_of "$tool" | grep -c . || true); tries=0
  while [ "$tries" -lt "$total" ]; do
    p=$(next_available "$tool") || break
    case "$tool" in
      claude) out=$(with_env_run claude "$p" "$(tool_bin claude)" -p "$@" 2>&1); code=$?;;
      codex)  out=$(with_env_run codex  "$p" "$(tool_bin codex)" exec "$@" 2>&1); code=$?;;
    esac
    # codex puede salir con código 0 aunque falle por límite: valida también la salida
    if [ "$code" -eq 0 ] && ! printf '%s' "$out" | grep -qiE "$re|$are"; then
      set_current "$tool" "$p"
      printf '%s\n' "$out"
      return 0
    fi
    if printf '%s' "$out" | grep -qiE "$re"; then
      reset=$(parse_reset_epoch "$out")
      [ "$reset" -le "$(now)" ] && reset=$(( $(now) + DEFAULT_COOLDOWN_MIN*60 ))
      mark_cooldown "$tool" "$p" "$reset"
      echo "⚠ [$tool:$p] límite de uso alcanzado (libre a las $(fmt_epoch "$reset")). Probando siguiente cuenta..." >&2
      tries=$((tries+1))
      continue
    fi
    if printf '%s' "$out" | grep -qiE "$are"; then
      mark_cooldown "$tool" "$p" $(( $(now) + 15*60 ))
      echo "⚠ [$tool:$p] sesión caducada o sin autenticación. Reactívala con: acm login $tool $p" >&2
      tries=$((tries+1))
      continue
    fi
    printf '%s\n' "$out" >&2
    return "$code"
  done
  report_all_limited "$tool"
}

cmd_launch() {
  local tool="$1"; shift
  local p id
  p=$(next_available "$tool") || { report_all_limited "$tool"; return 75; }
  set_current "$tool" "$p"
  id=$(identity_of "$tool" "$p" 2>/dev/null || true)
  echo "→ $tool · perfil '$p'${id:+ [$id]}" >&2
  with_env_run "$tool" "$p" "$(tool_bin "$tool")" "$@"
}

# ---------- quota: cuota restante SIN gastar tokens ----------
# codex : JSON-RPC local `codex app-server` -> account/rateLimits/read
# claude: GET https://api.anthropic.com/api/oauth/usage con el token OAuth del perfil
read -r -d '' JS_QUOTA_CODEX <<'JSEOF' || true
const {spawn}=require("child_process");
const [bin,home,raw]=process.argv.slice(1);
const env={...process.env}; if(home) env.CODEX_HOME=home;
const p=spawn(bin,["app-server"],{env,stdio:["pipe","pipe","ignore"]});
let buf="",done=false;
const finish=m=>{if(done)return;done=true;try{p.kill()}catch(e){};console.log(m);process.exit(0)};
const P=n=>String(n).padStart(2,"0");
const fmt=e=>{if(!e)return "?";const d=new Date(e*1000);return `${P(d.getDate())}/${P(d.getMonth()+1)} ${P(d.getHours())}:${P(d.getMinutes())}`};
const wname=m=>m===300?"5h":(m===10080?"semana":m+"m");
const send=o=>p.stdin.write(JSON.stringify(o)+"\n");
p.stdout.on("data",d=>{buf+=d;let i;while((i=buf.indexOf("\n"))>=0){const line=buf.slice(0,i);buf=buf.slice(i+1);
  if(!line.trim())continue;let m;try{m=JSON.parse(line)}catch(e){continue}
  if(m.id===0&&m.result){send({jsonrpc:"2.0",method:"initialized"});send({jsonrpc:"2.0",id:1,method:"account/rateLimits/read",params:{}})}
  if(m.id===0&&m.error)finish("error init: "+(m.error.message||""));
  if(m.id===1){
    if(m.error)return finish("error: "+(m.error.message||JSON.stringify(m.error)));
    if(raw==="1")return finish(JSON.stringify(m.result,null,1));
    const r=(m.result||{}).rateLimits||{};const parts=[];
    if(r.planType)parts.push("plan "+r.planType);
    for(const l of [r.primary,r.secondary]) if(l)
      parts.push(`${wname(l.windowDurationMins)}: ${Math.round(l.usedPercent)}% usado (reinicia ${fmt(l.resetsAt)})`);
    if(r.credits&&r.credits.hasCredits)parts.push("créditos: "+r.credits.balance);
    finish(parts.length?parts.join(" · "):"sin datos (prueba --raw)");
  }
}});
p.on("error",e=>finish("no se pudo lanzar codex app-server: "+e.message));
send({jsonrpc:"2.0",id:0,method:"initialize",params:{clientInfo:{name:"acm",title:"acm",version:"1.0"}}});
setTimeout(()=>finish("timeout consultando codex app-server"),15000);
JSEOF

read -r -d '' JS_QUOTA_CLAUDE <<'JSEOF' || true
const fs=require("fs"),https=require("https");
const [credPath,raw]=process.argv.slice(1);
let done=false;
const out=m=>{if(done)return;done=true;console.log(m);process.exit(0)};
let tok,exp;
try{
  // "env": el JSON llega en ACM_CRED_JSON (evita la carrera EAGAIN de leer un pipe con readFileSync(0))
  const txt=credPath==="env"?(process.env.ACM_CRED_JSON||""):fs.readFileSync(credPath,"utf8");
  const c=JSON.parse(txt).claudeAiOauth;tok=c.accessToken;exp=c.expiresAt;
  if(!tok)out("sin credenciales legibles");
}catch(e){out("sin credenciales legibles")}
if(exp!=null&&exp<Date.now())out("token caducado — reactiva con: acm login claude <perfil>");
const P=n=>String(n).padStart(2,"0");
const fmt=s=>{if(s==null)return "?";const d=new Date(typeof s==="number"?(s>1e12?s:s*1000):s);
  if(isNaN(d))return String(s);return `${P(d.getDate())}/${P(d.getMonth()+1)} ${P(d.getHours())}:${P(d.getMinutes())}`};
const req=https.request({host:"api.anthropic.com",path:"/api/oauth/usage",method:"GET",
  headers:{Authorization:"Bearer "+tok,"anthropic-beta":"oauth-2025-04-20","Content-Type":"application/json"}},res=>{
  let b="";res.on("data",d=>b+=d);res.on("end",()=>{
    let j=null;try{j=JSON.parse(b)}catch(e){}
    if(raw==="1")out("HTTP "+res.statusCode+"\n"+(j?JSON.stringify(j,null,1):String(b).slice(0,400)));
    if(res.statusCode===401)out("sesión caducada (401) — reactiva con: acm login claude <perfil>");
    if(res.statusCode===429)out("endpoint saturado o token inválido (429) — reintenta en unos minutos");
    if(!j)out("HTTP "+res.statusCode+" respuesta no JSON");
    // fuente primaria: array moderno `limits` (igual que /usage en la TUI,
    // incluye ventanas por modelo, p.ej. scope.model.display_name = "Fable")
    if(j&&Array.isArray(j.limits)&&j.limits.length){
      const lab=e=>{
        let b=e.kind==="session"?"5h":(e.kind==="weekly_all"?"semana":(e.group==="weekly"?"semana":e.kind));
        const m=e.scope&&e.scope.model&&e.scope.model.display_name;
        if(m)b+=" "+String(m).toLowerCase();
        const s=e.scope&&e.scope.surface;
        if(s)b+="/"+s;
        return b;
      };
      const ps=j.limits.filter(e=>e&&e.percent!=null).map(e=>
        `${lab(e)}: ${Math.round(e.percent)}% usado${e.severity&&e.severity!=="normal"?" ⚠":""} (reinicia ${fmt(e.resets_at)})`);
      if(ps.length)out(ps.join(" · "));
    }
    const label=k=>({five_hour:"5h",seven_day:"semana",seven_day_opus:"semana opus",
      seven_day_sonnet:"semana sonnet",seven_day_cowork:"semana cowork"})[k]||k;
    const parts=[];
    const walk=(o,pre)=>{for(const [k,v] of Object.entries(o||{})){
      if(!v||typeof v!=="object")continue;
      const pct=v.utilization??v.used_percent??v.usedPercent;
      if(pct!=null){
        const rst=v.resets_at??v.resetsAt;
        if(pct===0&&rst==null)continue; // ventana inactiva
        parts.push(`${pre?pre+".":""}${label(k)}: ${Math.round(pct)}% usado (reinicia ${fmt(rst)})`);
      } else walk(v,(pre?pre+".":"")+k);
    }};
    walk(j,"");
    if(parts.length)out(parts.join(" · "));
    if(j&&j.error)out("HTTP "+res.statusCode+": "+(j.error.type||"")+" — "+(j.error.message||""));
    out("HTTP "+res.statusCode+" formato desconocido — mira: acm quota claude --raw");
  })});
req.on("error",e=>out("error de red: "+e.message));
req.end();
setTimeout(()=>out("timeout"),15000);
JSEOF

cmd_quota() {
  command -v node >/dev/null 2>&1 || die "acm quota requiere node"
  local sel="" raw="0" a
  for a in "$@"; do
    case "$a" in
      --raw) raw="1";;
      *) tool_valid "$a" && sel="$a" || die "herramienta desconocida: $a";;
    esac
  done
  local list="${sel:-$TOOLS}" t p dir cj
  for t in $list; do
    echo "$t:"
    if [ -z "$(profiles_of "$t")" ]; then echo "   (sin perfiles)"; continue; fi
    for p in $(profiles_of "$t"); do
      if ! logged_in "$t" "$p"; then printf '  %-12s sin login\n' "$p"; continue; fi
      printf '  %-12s ' "$p"
      case "$t" in
        claude)
          cj=$(claude_cred_json "$(resolved_dir claude "$p")" 2>/dev/null || true)
          ACM_CRED_JSON="$cj" node -e "$JS_QUOTA_CLAUDE" env "$raw"
          ;;
        codex)
          dir=""; is_default_profile codex "$p" || dir=$(resolved_dir codex "$p")
          node -e "$JS_QUOTA_CODEX" "$(command -v "$(tool_bin codex)")" "$dir" "$raw"
          ;;
      esac
    done
  done
}

cmd_doctor() {
  local t v
  for t in $TOOLS; do
    v=$("$(tool_bin "$t")" --version 2>/dev/null | head -1 || echo "no ejecutable")
    printf '%-7s %s\n' "$t:" "$v"
  done
  echo "estado : $ACM_DIR"
  echo "cooldown por defecto: ${DEFAULT_COOLDOWN_MIN}m"
  echo "plataforma: $OS_NAME · fechas: $DATE_BIN $([ "$DATE_GNU" = 1 ] && echo '(GNU)' || echo '(BSD)')"
  if [ "$OS_NAME" = Darwin ] && [ "$DATE_GNU" = 0 ]; then
    echo "sugerencia macOS: 'brew install coreutils' (gdate) usa la vía de fechas más probada"
  fi
  command -v node >/dev/null 2>&1 || echo "aviso: sin node no funcionan 'acm quota' ni los emails de acm ls"
  echo
  cmd_ls
}

usage() {
  cat <<'EOF'
acm — gestor de cuentas para Claude Code y Codex CLI

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
El failover automático aplica a `acm run`; en sesiones interactivas usa
`acm limit` + volver a lanzar `acm <tool>` cuando la TUI anuncie el límite.
EOF
}

case "${1:-}" in
  init|ls|doctor|add|login|use|next|limit|free|run|quota)
    c=$1; shift; "cmd_$c" "$@";;
  claude|codex)
    t=$1; shift; cmd_launch "$t" "$@";;
  ""|-h|--help|help) usage;;
  *) die "comando desconocido: $1 (mira: acm help)";;
esac
