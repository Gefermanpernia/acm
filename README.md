# acm — Agent aCcount Manager

Gestor de cuentas en la terminal para CLIs de agentes — **Claude Code** y **Codex CLI** — con:

- **Perfiles aislados por cuenta**: cada login vive en su propio directorio de configuración (`CLAUDE_CONFIG_DIR` / `CODEX_HOME`), sin pisarse entre sí.
- **Failover automático**: en modo no interactivo (`acm run`), si una cuenta agota su límite de uso, acm lo detecta, anota hasta cuándo está bloqueada y reintenta con la siguiente.
- **Cuota en vivo sin gastar tokens**: `acm quota` muestra el % usado y la hora de reinicio de cada ventana (5h, semanal y ventanas por modelo, p. ej. Fable) leyendo los medidores oficiales — el endpoint OAuth de Anthropic y el JSON‑RPC local `codex app-server`.

```
$ acm quota
claude:
  cuenta2      5h: 3% usado (reinicia 20/08 00:59) · semana: 17% usado · semana fable: 30% usado
  principal    5h: 3% usado (reinicia 20/08 01:20) · semana: 61% usado · semana fable: 100% usado ⚠
codex:
  principal    plan pro · semana: 100% usado (reinicia 19/08 23:33)
```

## Instalación

Requisitos: `bash`, `node` (para `quota` y mostrar emails), y los CLIs que quieras gestionar (`claude`, `codex`).

```bash
mkdir -p ~/.local/bin
curl -fsSL https://raw.githubusercontent.com/Gefermanpernia/acm/main/acm -o ~/.local/bin/acm
chmod +x ~/.local/bin/acm
acm init   # adopta tus logins actuales como perfil "principal"
```

Aliases recomendados (bash/zsh en `~/.bashrc`/`~/.zshrc`, fish en `~/.config/fish/config.fish`):

```bash
alias cl="acm claude"        # Claude Code interactivo en el primer perfil disponible
alias cx="acm codex"         # Codex interactivo
alias clp="acm run claude"   # claude -p con failover automático
alias cxp="acm run codex"    # codex exec con failover automático
```

## Uso

| Comando | Qué hace |
|---|---|
| `acm ls` | Perfiles, cuenta activa `*`, emails y quién está al límite (y hasta cuándo) |
| `acm quota [tool] [--raw]` | Cuota restante por cuenta **sin gastar tokens** |
| `acm add <tool> <perfil>` | Crea un perfil nuevo (hereda tu config no‑secreta) y abre su login |
| `acm <tool> [args...]` | Lanza interactivo en el primer perfil disponible (banner con la cuenta) |
| `acm run <tool> [args...]` | No interactivo (`claude -p` / `codex exec`) con failover entre cuentas |
| `acm limit <tool> [perfil] [+1h\|HH:MM]` | Marca un límite visto en la TUI y activa la siguiente cuenta |
| `acm use / next / free` | Fijar cuenta, rotar, desbloquear |
| `acm login <tool> <perfil>` | (Re)abrir el login de un perfil (codex usa device‑auth, ideal en WSL) |
| `acm doctor` | Versiones, plataforma y estado |

Flujo típico cuando la TUI anuncia "límite alcanzado": sales, `acm limit claude 16:00` (la hora que te dijo), vuelves a lanzar `cl` y sigues en la otra cuenta. Los cooldowns expiran solos.

## Cómo funciona

- **Aislamiento**: cada perfil es un directorio; acm exporta `CLAUDE_CONFIG_DIR`/`CODEX_HOME` al lanzar. El perfil `principal` apunta al home por defecto **sin** exportar la variable (en claude, exportarla movería `~/.claude.json`).
- **Detección de límite**: por las cadenas reales de error (`Claude AI usage limit reached|<epoch>`, `You've hit your usage limit ... try again at ...`). Codex puede salir con código 0 aunque falle: se valida la salida, no el exit code. El reinicio se parsea (epoch, hora, o fecha completa) para el cooldown exacto.
- **Cuota**: claude → `GET api.anthropic.com/api/oauth/usage` con el token del perfil (array `limits[]`, incluye ventanas por modelo); codex → `codex app-server` JSON‑RPC `account/rateLimits/read`.

## Plataformas

- **Linux / WSL**: probado (suite de regresión + cuentas reales).
- **macOS**: soportado best‑effort — usa `gdate` si existe (`brew install coreutils`, recomendado) o `date` BSD; las credenciales de claude se leen del **Keychain** (`security`). El aislamiento de `CLAUDE_CONFIG_DIR` en macOS depende del build de Claude Code (verifica con `acm ls` que cada perfil muestre su email). Reportes bienvenidos.

## Aviso de uso responsable

- Esta herramienta gestiona **tus propias cuentas de pago**. Compartir cuentas viola los términos de ambos proveedores.
- **Anthropic**: el OAuth de suscripción solo puede usarse en Claude Code/Claude.ai; alternar tus propias cuentas dentro de Claude Code no está prohibido por ninguna norma publicada.
- **OpenAI**: sus términos prohíben "circumvent any rate limits"; la rotación automática de cuentas podría interpretarse así, con riesgo para las cuentas involucradas. El failover existe como conveniencia — úsalo bajo tu propio criterio.

## Licencia

MIT
