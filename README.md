# hermes-go

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)
![Status](https://img.shields.io/badge/Status-En%20desarrollo-orange?style=flat)
![WhatsApp](https://img.shields.io/badge/WhatsApp-Baileys-25D366?style=flat)

Reimplementacion de [Hermes](https://github.com/NousResearch/hermes-agent) en Go.  
Alta concurrencia sin GIL — hasta 2000 sesiones simultaneas en un solo proceso.

---

## Instalacion

**Linux / macOS / WSL**
```bash
curl -fsSL https://raw.githubusercontent.com/os-angel/hermes-go-version/main/scripts/install.sh | bash
```

**Windows**
```powershell
iex (irm https://raw.githubusercontent.com/os-angel/hermes-go-version/main/scripts/install.ps1)
```

Requisito previo: **Node.js >= 18** para el bridge de WhatsApp.  
El script lo verifica antes de continuar e indica donde descargarlo si falta.

---

## Configuracion del proveedor LLM

hermes-go funciona con cualquier endpoint compatible con la API de OpenAI.  
Soporta tanto **claves de API** (pago por token) como **tokens de suscripcion** (plan mensual).

### Proveedores compatibles

| Proveedor | Tipo | base_url | Modelo recomendado |
|-----------|------|----------|--------------------|
| **Nous Portal** | Suscripcion | `https://portal.nousresearch.com/api/v1` | `hermes-3-llama-3.1-405b` |
| **OpenRouter** | API key / gratis | `https://openrouter.ai/api/v1` | `meta-llama/llama-3.3-70b-instruct` |
| **OpenAI** | API key | `https://api.openai.com/v1` | `gpt-4o` |
| **OpenAI Codex** | Suscripcion ChatGPT Pro | `https://api.openai.com/v1` | `codex-mini-latest` |
| **GitHub Copilot** | Suscripcion Copilot | `https://api.githubcopilot.com` | `gpt-4o` |
| **Anthropic** | API key | `https://api.anthropic.com/v1` | `claude-opus-4-5` |
| **Groq** | API key / tier gratis | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile` |
| **Together AI** | API key | `https://api.together.xyz/v1` | `meta-llama/Llama-3-70b-chat-hf` |
| **Ollama** (local) | Sin clave | `http://localhost:11434/v1` | `llama3.3` |
| **LM Studio** (local) | Sin clave | `http://localhost:1234/v1` | *(el que cargues)* |

### Ejemplos de configuracion

**Nous Portal — suscripcion**
```yaml
llm:
  base_url: "https://portal.nousresearch.com/api/v1"
  api_key: "${NOUS_PORTAL_TOKEN}"
  model: "hermes-3-llama-3.1-405b"
```

**OpenRouter — API key o modelos gratis**
```yaml
llm:
  base_url: "https://openrouter.ai/api/v1"
  api_key: "${OPENROUTER_API_KEY}"
  model: "meta-llama/llama-3.3-70b-instruct"
```

**OpenAI Codex — suscripcion ChatGPT Pro**
```yaml
llm:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  model: "codex-mini-latest"
```

**GitHub Copilot — suscripcion**
```yaml
llm:
  base_url: "https://api.githubcopilot.com"
  api_key: "${GITHUB_COPILOT_TOKEN}"
  model: "gpt-4o"
```

**Ollama — modelo local, sin clave**
```yaml
llm:
  base_url: "http://localhost:11434/v1"
  api_key: "ollama"
  model: "llama3.3"
```

Todas las variables `${VAR}` se expanden automaticamente desde el entorno al arrancar.

---

## Configuracion completa

Despues de instalar, edita `~/.hermes-go/config.yaml`:

```yaml
llm:
  base_url: "https://openrouter.ai/api/v1"
  api_key: "${OPENROUTER_API_KEY}"
  model: "meta-llama/llama-3.3-70b-instruct"
  timeout: 120s
  max_retries: 5

agent:
  identity: |
    Eres un asistente util y directo.
    Respondes en el idioma del usuario.
  max_iterations: 12
  workers: 16
  session_cache_size: 256
  session_ttl: 1h

platforms:
  whatsapp:
    enabled: true
    bridge_port: 3001

memory:
  builtin:
    memory_char_limit: 2200
    user_char_limit: 1375

server:
  listen_addr: "0.0.0.0:8080"

logging:
  level: "info"
  format: "json"
  prometheus_enabled: false
```

---

## Arrancar

```bash
hermes-go --config ~/.hermes-go/config.yaml
```

Con WhatsApp habilitado, el bridge imprime un QR en la terminal al primer arranque.  
Escanea con tu telefono en **WhatsApp > Dispositivos vinculados**.  
Las credenciales se guardan en `~/.hermes-go/whatsapp/` — no necesitas volver a escanear.

---

## Caracteristicas

- Loop de conversacion con cualquier LLM OpenAI-compatible
- Alta concurrencia: goroutines + worker pool, sin GIL (hasta 2000 sesiones)
- Memoria persistente: `MEMORY.md` + `USER.md` por sesion, con file locking
- Skills: instrucciones reutilizables cargadas desde archivos `.md` bajo demanda
- Cliente MCP: stdio, StreamableHTTP, SSE con reconexion automatica
- WhatsApp via bridge Node.js Baileys
- Email IMAP/SMTP *(Fase 10)*
- Webhook con HMAC-SHA256 *(Fase 11)*
- REST API con Bearer auth *(Fase 12)*
- Scheduler cron para tareas periodicas *(Fase 15)*
- Sistema de plugins in-tree *(Fase 16)*

---

## Desarrollo

```bash
git clone https://github.com/os-angel/hermes-go-version
cd hermes-go-version

go mod tidy
make bridge          # npm install del bridge de WhatsApp

make run             # compila y corre con ./config.yaml
make race            # tests con race detector (obligatorio antes de merge)
make release         # compila para todos los targets en dist/
```

---

## Estado del proyecto

Ver [`Checkpoints.md`](Checkpoints.md) para el progreso fase por fase e instrucciones para implementar cada fase con Claude.

Ver [`../PLAN.md`](../PLAN.md) para la arquitectura completa y contratos de cada modulo.
