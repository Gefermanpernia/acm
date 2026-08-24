# acm — Agent aCcount Manager

Gestor de cuentas en la terminal para CLIs de agentes — **Claude Code** y **Codex CLI** — escrito en **Go** (binario único, sin dependencias):

- **Perfiles aislados por cuenta**: cada login vive en su propio directorio de configuración (`CLAUDE_CONFIG_DIR` / `CODEX_HOME`), sin pisarse entre sí.
- **Failover automático**: en modo no interactivo (`acm run`), si una cuenta agota su límite de uso, acm lo detecta, anota hasta cuándo está bloqueada y reintenta con la siguiente.
- **Cuota en vivo sin gastar tokens**: `acm quota` muestra el % usado y la hora de reinicio de cada ventana (5h, semanal y ventanas por modelo, p. ej. Fable) leyendo los medidores oficiales — el endpoint OAuth de Anthropic y el JSON‑RPC local `codex app-server`.

```
$ acm quota
claude:
  cuenta2      5h: 5% usado (reinicia 01:00 (20/08)) · semana: 17% usado · semana fable: 31% usado
  principal    5h: 7% usado (reinicia 01:20 (20/08)) · semana: 62% usado · semana fable: 100% usado ⚠
codex:
  principal    plan pro · semana: 100% usado (reinicia 23:33 (19/08))
```

## Instalación

Un solo comando, desde **cualquier shell** (bash, zsh, fish, ...) en Linux/WSL y macOS (x86_64 y arm64):

```sh
curl -fsSL https://raw.githubusercontent.com/Gefermanpernia/acm/main/install.sh | sh
```

El instalador descarga el binario correcto para tu sistema, lo deja en `~/.local/bin/acm`, añade los aliases (`cl`, `cx`, `clp`, `cxp`) a bash, zsh **y** fish (solo en los que existan, sin duplicar) y adopta tus logins actuales como perfil `principal`. Variables opcionales: `ACM_VERSION=v2.0.0`, `ACM_BIN_DIR=~/bin`.

Alternativas:

- **Con Go instalado**: `go install github.com/Gefermanpernia/acm@latest`
- **Windows nativo**: descarga `acm_windows_amd64.exe` desde [Releases](https://github.com/Gefermanpernia/acm/releases).
- **Manual**: baja el binario de tu plataforma desde Releases y dale permisos de ejecución.

No requiere bash, node ni ninguna otra dependencia en tiempo de ejecución: solo los CLIs que quieras gestionar (`claude`, `codex`).

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

Aliases que instala el script (misma sintaxis en bash/zsh/fish):

```sh
alias cl="acm claude"        # Claude Code interactivo
alias cx="acm codex"         # Codex interactivo
alias clp="acm run claude"   # claude -p con failover
alias cxp="acm run codex"    # codex exec con failover
```

Flujo típico cuando la TUI anuncia "límite alcanzado": sales, `acm limit claude 16:00` (la hora que te dijo), vuelves a lanzar `cl` y sigues en la otra cuenta. Los cooldowns expiran solos.

## OpenCode transparent failover

El instalador incluye, pero deja **deshabilitado**, el adaptador experimental ES modules de `integrations/opencode/`. Solo admite Linux, perfiles ACM y la matriz fijada OpenCode 1.18.19 / SDK 1.17.12 / Claude CLI 2.1.236.
La frontera estable es `acm machine v1 <operation>` por stdin/stdout, con entrada máxima de 64 KiB y salida máxima de 16 KiB. Implementa `credential.select`, `diagnostics.status`, `oauth.refresh.begin|commit|abort` y `quota.exhaust`. ACM selecciona el perfil y devuelve su directorio de configuración; el adaptador lee allí `.credentials.json`, mantiene el token solo en memoria, nunca escribe `auth.json` de OpenCode y nunca registra credenciales.
La rotación es deliberadamente conservadora:
- Requiere simultáneamente HTTP 429, el error tipado `rate_limit_error` y `anthropic-ratelimit-unified-status: rejected`. Un 429 genérico o cualquier 529 pasa sin cambios.
- Enfriamiento y cuarentena son resultados distintos. ACM aplica su propia política de cooldown cuando no existe un reset válido; una cuarentena exige `acm login`.
- Reintentos, backoff, esperas y continuidad de sesión pertenecen a OpenCode. El adaptador hace una sola llamada al proveedor por intento.
Para habilitarla, cierre OpenCode y ejecute:

```sh
acm opencode enable --confirm
```
El comando valida `opencode.json` o `opencode.jsonc`. Si detecta `opencode-anthropic-login-via-cli`, solo o junto con el adaptador ACM, se detiene sin modificar la configuración ni crear respaldos. Revise el conflicto y confirme la migración de forma explícita:

```sh
acm opencode enable --confirm --replace-upstream
```

Solo esta ruta reemplaza el plugin upstream. Tanto la activación directa como la migración explícita crean un respaldo con checksum antes de modificar la configuración. Después reinicie OpenCode. Para deshacer cualquiera de las dos rutas, cierre OpenCode, ejecute `acm opencode rollback --confirm` y reinícielo; el rollback solo restaura la configuración de OpenCode y no modifica el estado ni las cuentas de ACM.

## Cómo funciona

- **Aislamiento**: cada perfil es un directorio; acm exporta `CLAUDE_CONFIG_DIR`/`CODEX_HOME` al lanzar. El perfil `principal` apunta al home por defecto **sin** exportar la variable (en claude, exportarla movería `~/.claude.json`).
- **Detección de límite**: por las cadenas reales de error (`Claude AI usage limit reached|<epoch>`, `You've hit your usage limit ... try again at ...`). Codex puede salir con código 0 aunque falle: se valida la salida, no el exit code. El reinicio se parsea (epoch, hora, o fecha completa) para aplicar el cooldown exacto.
- **Cuota**: claude → `GET api.anthropic.com/api/oauth/usage` con el token del perfil (array `limits[]`, incluye ventanas por modelo); codex → `codex app-server` JSON‑RPC `account/rateLimits/read`.
- **Estado**: todo en `~/.acm/` (perfiles, cuenta activa, cooldowns). Compatible con la versión bash original (`legacy/acm.sh`).

## Por qué Go

- **Seguridad**: binario estático compilado solo con la librería estándar (cero dependencias externas), sin `eval` ni interpolación de shell; los tokens nunca pasan por pipes ni argumentos de proceso, y las peticiones van directas por HTTPS con verificación de certificados.
- **Velocidad**: arranque instantáneo y parseo nativo de JSON/fechas (la versión bash necesitaba node y GNU date).
- **Portabilidad real**: el mismo código corre en Linux, macOS (Intel y Apple Silicon, con lectura del Keychain vía `security`) y Windows.

## Plataformas

- **Linux / WSL**: probado (suite de regresión + cuentas reales).
- **macOS**: soportado — credenciales de claude leídas del **Keychain**; el aislamiento de `CLAUDE_CONFIG_DIR` depende del build de Claude Code (verifica con `acm ls` que cada perfil muestre su email). Reportes bienvenidos.
- **Windows nativo**: binario disponible; los CLIs guardan credenciales en archivos igual que en Linux.

## Aviso de uso responsable

- Esta herramienta gestiona **tus propias cuentas de pago**. Compartir cuentas viola los términos de ambos proveedores.
- **Anthropic**: el OAuth de suscripción solo puede usarse en Claude Code/Claude.ai; alternar tus propias cuentas dentro de Claude Code no está prohibido por ninguna norma publicada.
- **OpenAI**: sus términos prohíben "circumvent any rate limits"; la rotación automática de cuentas podría interpretarse así, con riesgo para las cuentas involucradas. El failover existe como conveniencia — úsalo bajo tu propio criterio.

## Licencia

MIT
