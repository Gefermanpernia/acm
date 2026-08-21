#!/bin/sh
# Instalador de acm — https://github.com/Gefermanpernia/acm
#
#   curl -fsSL https://raw.githubusercontent.com/Gefermanpernia/acm/main/install.sh | sh
#
# Funciona desde cualquier shell (bash, zsh, fish, ...): es POSIX sh.
# Descarga el binario correcto para tu SO/arquitectura desde GitHub Releases,
# lo instala en ~/.local/bin/acm y añade los aliases (cl, cx, clp, cxp) a
# bash, zsh y fish — solo en los que existan, y sin duplicarlos.
#
# Variables opcionales:
#   ACM_VERSION=v2.0.0   instala una versión concreta (por defecto: latest)
#   ACM_BIN_DIR=~/bin    directorio de instalación (por defecto: ~/.local/bin)
set -eu

REPO="Gefermanpernia/acm"
VERSION="${ACM_VERSION:-latest}"
BIN_DIR="${ACM_BIN_DIR:-$HOME/.local/bin}"
SHARE_DIR="${ACM_SHARE_DIR:-$HOME/.local/share/acm}"

say() { printf '%s\n' "$*"; }

os=$(uname -s)
arch=$(uname -m)
case "$os" in
  Linux)  goos=linux ;;
  Darwin) goos=darwin ;;
  *)
    say "SO no soportado por este instalador: $os"
    say "En Windows usa WSL, o descarga acm_windows_amd64.exe del release."
    exit 1 ;;
esac
case "$arch" in
  x86_64|amd64)  goarch=amd64 ;;
  aarch64|arm64) goarch=arm64 ;;
  *) say "arquitectura no soportada: $arch"; exit 1 ;;
esac

asset="acm_${goos}_${goarch}"
if [ "$VERSION" = "latest" ]; then
  url="https://github.com/$REPO/releases/latest/download/$asset"
else
  url="https://github.com/$REPO/releases/download/$VERSION/$asset"
fi

mkdir -p "$BIN_DIR"
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

say "→ Descargando $asset ($VERSION)..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp" "$url"
else
  say "necesito curl o wget para descargar"; exit 1
fi
chmod +x "$tmp"
mv "$tmp" "$BIN_DIR/acm"
trap - EXIT
say "✓ acm instalado: $BIN_DIR/acm ($("$BIN_DIR/acm" version 2>/dev/null || echo ok))"

plugin_tmp=$(mktemp -d)
trap 'rm -rf "$plugin_tmp"' EXIT
ref=main; [ "$VERSION" = latest ] || ref="$VERSION"
for file in index.js machine.js oauth.js compat.js quota.js diagnostics.js compatibility.json package.json; do
  raw="https://raw.githubusercontent.com/$REPO/$ref/integrations/opencode/$file"
  if command -v curl >/dev/null 2>&1; then curl -fsSL "$raw" -o "$plugin_tmp/$file"
  else wget -qO "$plugin_tmp/$file" "$raw"; fi
done
mkdir -p "$SHARE_DIR"
rm -rf "$SHARE_DIR/.opencode-old"
[ ! -d "$SHARE_DIR/opencode" ] || mv "$SHARE_DIR/opencode" "$SHARE_DIR/.opencode-old"
mv "$plugin_tmp" "$SHARE_DIR/opencode"
rm -rf "$SHARE_DIR/.opencode-old"
trap - EXIT
say "✓ adaptador OpenCode incluido (deshabilitado): $SHARE_DIR/opencode"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) say "⚠ $BIN_DIR no está en tu PATH — añádelo en la config de tu shell." ;;
esac

# ---------- aliases (bash / zsh / fish usan la misma sintaxis) ----------
MARK="acm (gestor de cuentas Claude/Codex)"
write_block() {
  cat >> "$1" <<'EOF'

# >>> acm (gestor de cuentas Claude/Codex) >>>
alias cl="acm claude"        # Claude Code interactivo en el primer perfil disponible
alias cx="acm codex"         # Codex interactivo
alias clp="acm run claude"   # claude -p con failover automático de cuenta
alias cxp="acm run codex"    # codex exec con failover automático de cuenta
# <<< acm <<<
EOF
}
add_aliases() {
  f="$1"
  [ -f "$f" ] || return 0
  if grep -q "$MARK" "$f" 2>/dev/null; then
    say "· aliases ya presentes en $f"
  else
    write_block "$f"
    say "✓ aliases añadidos a $f"
  fi
}

add_aliases "$HOME/.bashrc"
add_aliases "$HOME/.zshrc"
if command -v fish >/dev/null 2>&1 || [ -d "$HOME/.config/fish" ]; then
  mkdir -p "$HOME/.config/fish"
  [ -f "$HOME/.config/fish/config.fish" ] || : > "$HOME/.config/fish/config.fish"
  add_aliases "$HOME/.config/fish/config.fish"
fi

say ""
say "→ Adoptando logins existentes (acm init)..."
"$BIN_DIR/acm" init || true
say ""
say "Listo. Abre una terminal nueva (o recarga tu config de shell) y usa:"
say "  cl · cx · clp · cxp · acm quota · acm help"
