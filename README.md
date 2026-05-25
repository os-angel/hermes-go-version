# hermes-go

Agente AI en Go con alta concurrencia. Reimplementacion de Hermes (Python/Nous Research).

~500-2000 sesiones simultaneas sin GIL. Un binario estatico. Sin virtual environments.

## Instalacion

**Linux / macOS / WSL:**
```bash
curl -fsSL https://raw.githubusercontent.com/os-angel/hermes-go-version/main/scripts/install.sh | bash
```

**Windows (PowerShell):**
```powershell
iex (irm https://raw.githubusercontent.com/os-angel/hermes-go-version/main/scripts/install.ps1)
```

Requisito: **Node.js >= 18** (para el bridge de WhatsApp). El script verifica esto antes de continuar.

Lo que instala el script:
- Binario `hermes-go` en `~/.local/bin/` (Linux/macOS) o `%LOCALAPPDATA%\hermes-go\bin\` (Windows)
- Bridge de WhatsApp en `~/.hermes-go/bridge/` con dependencias npm
- Config de ejemplo en `~/.hermes-go/config.yaml`

## Configuracion inicial

```bash
# Editar config antes de arrancar
nano ~/.hermes-go/config.yaml
```

Campos minimos requeridos:
```yaml
llm:
  base_url: "https://openrouter.ai/api/v1"
  api_key: "sk-or-..."
  model: "meta-llama/llama-3.3-70b-instruct"

platforms:
  whatsapp:
    enabled: true
```

## Arrancar el agente

```bash
hermes-go --config ~/.hermes-go/config.yaml
```

Al arrancar con WhatsApp habilitado, el bridge imprime un QR en terminal para escanear con tu telefono (Dispositivos vinculados). Las credenciales se guardan en `~/.hermes-go/whatsapp/` y no tienes que volver a escanear.

## Caracteristicas

- Loop de conversacion con LLM (OpenAI-compatible: OpenRouter, Anthropic, Ollama, etc.)
- Alta concurrencia: goroutines + worker pool, sin GIL
- Memoria persistente: MEMORY.md + USER.md con flock y atomic write
- Skills: instrucciones reutilizables cargadas bajo demanda desde archivos .md
- Cliente MCP: stdio, StreamableHTTP, SSE con reconexion automatica
- WhatsApp via bridge Node.js Baileys (mismo patron que hermes-agent)
- Email IMAP/SMTP (Fase 10)
- Webhook generico con HMAC-SHA256 (Fase 11)
- REST API con Bearer auth (Fase 12)
- Scheduler cron para jobs periodicos (Fase 15)
- Sistema de plugins in-tree (Fase 16)

## Desarrollo (desde el codigo fuente)

```bash
git clone https://github.com/os-angel/hermes-go-version
cd hermes-go

# Instalar dependencias
go mod tidy
make bridge          # npm install del bridge

# Compilar y correr
make run             # usa ./config.yaml como default

# Tests con race detector (obligatorio antes de merge)
make race

# Compilar para todos los targets de release
make release         # genera binarios en dist/
```

## Estado del proyecto

Ver `Checkpoints.md` para el progreso de implementacion fase por fase y las instrucciones exactas para implementar cada fase con Claude.

## Arquitectura

Ver `../PLAN.md` para la arquitectura completa, contratos de cada modulo y orden de implementacion.
