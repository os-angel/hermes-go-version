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

## Proveedores LLM

hermes-go soporta los mismos proveedores que hermes-agent: claves de API, suscripciones
(Nous Portal, OpenAI Codex, GitHub Copilot), OAuth y modelos locales.

### Tipos de autenticacion

| Tipo | Descripcion |
|------|-------------|
| `api_key` | Clave de API via variable de entorno o config |
| `oauth_device_code` | Login via navegador con device code flow (Nous Portal) |
| `oauth_external` | Login via OAuth con redirect en navegador |
| `copilot` | GitHub token — intercambiado por token de Copilot automaticamente |
| `aws_sdk` | Credenciales IAM de AWS — sin clave explicita |

### Proveedores disponibles

#### Suscripciones (sin API key manual)

| Nombre | Tipo | Modelo recomendado | Requisito |
|--------|------|--------------------|-----------|
| `nous` | oauth_device_code | `hermes-3-405b` | Cuenta Nous Research |
| `openai-codex` | oauth_external | `codex-mini-latest` | Suscripcion ChatGPT Pro |
| `copilot` | copilot | `gpt-4o` | Suscripcion GitHub Copilot |
| `xai-oauth` | oauth_external | `grok-3` | Cuenta xAI |
| `minimax-oauth` | oauth_external | `MiniMax-M2.7` | Cuenta MiniMax |
| `qwen-oauth` | oauth_external | `qwen-max` | Cuenta Alibaba Cloud |

#### API Key

| Nombre | Variable de entorno | Modelo recomendado | Nivel gratis |
|--------|--------------------|--------------------|:------------:|
| `openrouter` | `OPENROUTER_API_KEY` | `meta-llama/llama-3.3-70b-instruct` | Si |
| `anthropic` | `ANTHROPIC_API_KEY` | `claude-opus-4-5` | No |
| `openai` | `OPENAI_API_KEY` | `gpt-4o` | No |
| `gemini` | `GOOGLE_API_KEY` | `gemini-2.0-flash` | Si |
| `groq` | `GROQ_API_KEY` | `llama-3.3-70b-versatile` | Si |
| `deepseek` | `DEEPSEEK_API_KEY` | `deepseek-chat` | No |
| `xai` | `XAI_API_KEY` | `grok-3` | No |
| `together` | `TOGETHER_API_KEY` | `meta-llama/Llama-3-70b-chat-hf` | No |
| `nvidia` | `NVIDIA_API_KEY` | `meta/llama-3.3-70b-instruct` | Si |
| `minimax` | `MINIMAX_API_KEY` | `MiniMax-M2.7` | No |
| `kimi` | `KIMI_API_KEY` | `moonshot-v1-8k` | No |
| `huggingface` | `HF_TOKEN` | *(el que elijas)* | Si |
| `bedrock` | Credenciales IAM | `anthropic.claude-opus-4-5-v1:0` | No |
| `azure` | `AZURE_OPENAI_API_KEY` | *(el que configures)* | No |
| `nous-api` | `NOUS_API_KEY` | `hermes-3-405b` | No |

#### Locales (sin clave)

| Nombre | URL | Modelo recomendado |
|--------|-----|--------------------|
| `ollama` | `http://localhost:11434/v1` | `llama3.3` |
| `lmstudio` | `http://localhost:1234/v1` | *(el que cargues)* |

---

## Configuracion

### Opcion A — usando el registro de proveedores (recomendado)

Con el campo `provider`, hermes-go resuelve credenciales automaticamente:

```yaml
llm:
  provider: "openrouter"           # nombre del proveedor del registro
  model: "meta-llama/llama-3.3-70b-instruct"
  timeout: 120s
  max_retries: 5
```

Las credenciales se toman de las variables de entorno declaradas en el proveedor
(`OPENROUTER_API_KEY` en este caso). Para proveedores OAuth, se leen de
`~/.hermes-go/auth.json` (ver seccion de autorizacion mas abajo).

### Opcion B — configuracion manual (base_url + api_key)

Si prefieres controlar los endpoints directamente:

```yaml
llm:
  base_url: "https://openrouter.ai/api/v1"
  api_key: "${OPENROUTER_API_KEY}"
  model: "meta-llama/llama-3.3-70b-instruct"
```

Todas las variables `${VAR}` se expanden desde el entorno al arrancar.

### Ejemplos

**Nous Portal — suscripcion (OAuth device code)**
```yaml
# Primero autoriza: hermes-go auth add nous
llm:
  provider: "nous"
  model: "hermes-3-405b"
```

**OpenAI Codex — suscripcion ChatGPT Pro**
```yaml
# Primero autoriza: hermes-go auth add openai-codex
llm:
  provider: "openai-codex"
  model: "codex-mini-latest"
```

**GitHub Copilot — suscripcion**
```yaml
# Requiere: export GH_TOKEN=ghp_xxx  o  COPILOT_GITHUB_TOKEN=ghp_xxx
llm:
  provider: "copilot"
  model: "gpt-4o"
```

**OpenRouter — API key o modelos gratis**
```yaml
# export OPENROUTER_API_KEY=sk-or-xxx
llm:
  provider: "openrouter"
  model: "meta-llama/llama-3.3-70b-instruct"
```

**Ollama — modelo local, sin clave**
```yaml
llm:
  provider: "ollama"
  model: "llama3.3"
```

---

## Autorizar proveedores OAuth

Para proveedores con suscripcion (Nous, Codex, xAI, MiniMax, Qwen):

```bash
# Ver todos los proveedores y su estado actual
hermes-go auth list

# Autorizar un proveedor (abre el navegador o muestra un codigo)
hermes-go auth add nous
hermes-go auth add openai-codex

# Eliminar credencial guardada
hermes-go auth remove nous
```

Los tokens se guardan en `~/.hermes-go/auth.json` con refresh automatico
(2 minutos antes de vencer).

---

## Configuracion completa

Despues de instalar, edita `~/.hermes-go/config.yaml`:

```yaml
llm:
  provider: "openrouter"
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
- 25+ proveedores integrados: API keys, suscripciones OAuth, modelos locales
- Gestion de credenciales OAuth con refresh automatico en `~/.hermes-go/auth.json`
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
