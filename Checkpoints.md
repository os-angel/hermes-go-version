# Checkpoints: hermes-go

Tracker de progreso del proyecto. Indica que fases tienen codigo implementado,
cuales estan probadas y cuales estan pendientes.

Si eres Claude leyendo este archivo: revisa la tabla antes de implementar cualquier cosa.
Cuando termines una fase, marca su checkbox y agrega fecha y nota breve.
No marques una fase como completa si los tests no pasan o si hay panics en produccion.

---

## Estado del proyecto

| Fase | Descripcion | Dev | Test | Prod | Notas |
|------|-------------|:---:|:----:|:----:|-------|
| Scaffold | Estructura, stubs, configs, bridge.js | x | - | - | Generado 2026-05-22 |
| 1 - Config | config.go, paths.go, observability | x | - | - | Implementado en scaffold |
| 2 - Memory Builtin | BuiltinProvider: MEMORY.md + USER.md, flock, scan | | | | |
| 3 - LLM Client | ChatCompletion, retry, classify_error | | | | |
| 4 - Session | FileSessionStore, SessionCache LRU+TTL | x | - | - | Implementado en scaffold |
| 5 - Prompt Builder | 3 capas: stable/context/volatile | x | - | - | Implementado en scaffold (basico) |
| 6 - Conv. Loop | Run(): LLM -> tools -> respuesta final | | | | |
| 7 - Skills | Loader: discover, load, system prompt block | x | - | - | Implementado en scaffold |
| 8 - Platform Router | Router + workers + Sender/Receiver interfaces | x | - | - | Implementado en scaffold |
| 9 - WhatsApp | Bridge.js (Baileys), Poller, Sender, Identity | x | - | - | bridge.js funcional |
| 10 - Email | IMAPPoller, SMTPSender, Parser, Threading | | | | |
| 11 - Webhook | Server HMAC, SubscriptionStore | | | | |
| 12 - REST API | Server, BearerAuth, POST /v1/chat | | | | |
| 13 - MCP Client | stdio/http/sse transports, Manager, reconnect | | | | |
| 14 - Send Tool | send_message tool para mensajes proactivos | x | - | - | Stub funcional en scaffold |
| 15 - Cron | Store, Scanner, Runner, Scheduler (robfig/cron) | | | | |
| 16 - Plugins | Plugin interface, Registry in-tree | x | - | - | Implementado en scaffold |
| 17 - Stress Tests | concurrent_test.go, memory_test.go, k6 | | | | |
| 18 - CI/CD | Makefile, Dockerfile, GitHub Actions | x | - | - | Makefile en scaffold |

Referencia columnas: x = completo, en blanco = pendiente, - = no aplica

---

## Como implementar cada fase con Claude

Abre una conversacion nueva de Claude Code en la raiz de `hermes-go/`.
Claude leera este archivo y el PLAN.md para entender el contexto completo.

### Fase 2 — Memory Builtin

> "Lee PLAN.md y Checkpoints.md. La Fase 2 esta pendiente.
> Implementa BuiltinProvider en internal/memory/builtin.go:
> metodos Initialize, SystemPromptBlock, HandleToolCall, Add, Replace, Remove, Read.
> Usa flock (gofrs/flock) para locks de escritura y atomic write (temp + os.Rename).
> El delimiter entre entries es el literal '\n§\n'.
> Char limits: memory 2200 / user 1375. Truncar entries mas antiguas al superar el limite.
> Anti-injection scan en Add/Replace (ya existe en scan.go).
> Snapshot frozen al Initialize: capturar estado inicial sin mutarlo durante la sesion.
> Implementa los tests en tests/memory/ con go test -race.
> Actualiza Checkpoints.md cuando termines."

### Fase 3 — LLM Client

> "Lee PLAN.md y Checkpoints.md. La Fase 3 esta pendiente.
> Implementa ChatCompletion en internal/llm/client.go usando github.com/openai/openai-go.
> Retry con backoff exponencial (cenkalti/backoff/v4): max 5 intentos, solo para errores Transient.
> classify_error.go ya existe para clasificar errores de la API.
> El modelo y base_url vienen del LLMConfig inyectado en NewClient.
> Implementa tests mockeando el servidor HTTP.
> Actualiza Checkpoints.md cuando termines."

### Fase 6 — Conversation Loop

> "Lee PLAN.md y Checkpoints.md. Las Fases 2-5 estan completas. La Fase 6 esta pendiente.
> Implementa ConversationLoop.Run() en internal/agent/loop.go:
> - Llamar al LLM con el historial de sesion + system prompt
> - Parsear tool calls de la respuesta
> - Ejecutar tools via ToolExecutor (ya implementado)
> - Agregar resultados al historial y repetir hasta respuesta final o MaxIter
> - Llamar memory.SyncTurn al terminar cada turno
> ToolExecutor.executeOne() esta en tool_executor.go: completar el parse de JSON args.
> Implementa tests con mocks del LLM.
> Actualiza Checkpoints.md cuando termines."

### Fase 10 — Email

> "Lee PLAN.md y Checkpoints.md. La Fase 10 esta pendiente.
> Implementa IMAPPoller en internal/platforms/email/imap.go usando github.com/emersion/go-imap/v2.
> Implementa SMTPSender en internal/platforms/email/smtp.go usando github.com/wneessen/go-mail.
> Implementa Parse() en parser.go para extraer cuerpo, adjuntos y headers de mensajes MIME.
> Implementa ThreadIndex.Resolve() en threading.go para mantener session IDs de hilos.
> Wiring en cmd/agent/main.go: agregar el bloque de email junto al de WhatsApp.
> Actualiza Checkpoints.md cuando termines."

### Fase 11 — Webhook

> "Lee PLAN.md y Checkpoints.md. La Fase 11 esta pendiente.
> Implementa verifyHMAC en internal/platforms/webhook/server.go (HMAC-SHA256 con X-Hub-Signature-256).
> Implementa handleWebhook: validar firma, parsear payload JSON, emitir IncomingMessage.
> Implementa Upsert y Delete en subscription.go con atomic write.
> Actualiza Checkpoints.md cuando termines."

### Fase 12 — REST API

> "Lee PLAN.md y Checkpoints.md. La Fase 12 esta pendiente.
> Implementa handleChat en internal/platforms/restapi/server.go:
> construir IncomingMessage desde el body y enviarlo al router.
> El campo session_id del body se usa directamente como SessionID.
> Si session_id esta vacio, generar uno desde el user_id.
> Actualiza Checkpoints.md cuando termines."

### Fase 13 — MCP Client

> "Lee PLAN.md y Checkpoints.md. La Fase 13 esta pendiente.
> Implementa Call() en transport_stdio.go: serializar JSON-RPC, escribir a stdin, leer linea de stdout.
> Implementa Call() en transport_http.go: POST al endpoint /mcp, parsear respuesta JSON-RPC.
> Implementa NewServerClient, Initialize, ListTools, CallTool en client.go.
> ListTools: llamar 'tools/list', registrar cada tool en el Registry con un handler que llame CallTool.
> Implementa Manager.Connect con reconnectLoop en background.
> Wiring en main.go: iterar cfg.MCP.Servers y llamar mcpMgr.Connect para cada uno.
> Actualiza Checkpoints.md cuando termines."

### Fase 15 — Cron Scheduler

> "Lee PLAN.md y Checkpoints.md. La Fase 15 esta pendiente.
> Implementa Upsert, Delete, UpdateLastRun en internal/cron/jobs.go con atomic write y flock.
> Implementa Runner.Run() en runner.go: validar gracia de 60s, escanear prompt, despachar al agente.
> Implementa Scheduler.Start() y Reload() en scheduler.go usando robfig/cron/v3.
> Agregar tools al Registry: cron_add, cron_list, cron_delete, cron_enable.
> Actualiza Checkpoints.md cuando termines."

---

## Setup para trabajar en el proyecto

```bash
# 1. Instalar dependencias Go
go mod tidy
go mod download

# 2. Instalar dependencias Node.js del bridge
cd bridge && npm install && cd ..

# 3. Verificar que todo compila
go build ./...

# 4. Ejecutar tests con race detector
go test -race ./...

# 5. Configurar el agente
cp config.example.yaml config.yaml
# Editar config.yaml con tu API key, bridge path, etc.

# 6. Correr el agente
go run ./cmd/agent --config config.yaml
```

---

## Estructura del proyecto

```
hermes-go/
├── cmd/agent/main.go              # Entrypoint: wiring de todos los componentes
├── internal/
│   ├── agent/                     # Loop de conversacion, sesiones, prompt
│   │   ├── loop.go                # ConversationLoop.Run() [Fase 6]
│   │   ├── prompt.go              # PromptBuilder 3 capas [Fase 5, parcial]
│   │   ├── session.go             # Session thread-safe
│   │   ├── session_cache.go       # LRU + TTL + disk [Fase 4]
│   │   ├── session_store.go       # FileSessionStore [Fase 4]
│   │   └── tool_executor.go       # errgroup + semaforo max 8 [Fase 6]
│   ├── config/                    # Config YAML + paths
│   ├── cron/                      # Scheduler de jobs periodicos [Fase 15]
│   ├── identity/                  # PII hashing (sha256 v1:salt:value)
│   ├── llm/                       # Cliente OpenAI-compatible [Fase 3]
│   ├── memory/                    # MEMORY.md + USER.md + Manager [Fase 2]
│   ├── observability/             # slog + Prometheus
│   ├── platforms/
│   │   ├── router.go              # Router + worker pool [Fase 8]
│   │   ├── types.go               # IncomingMessage, OutgoingMessage
│   │   ├── email/                 # IMAP + SMTP [Fase 10]
│   │   ├── restapi/               # REST API + BearerAuth [Fase 12]
│   │   ├── webhook/               # Webhook HMAC [Fase 11]
│   │   └── whatsapp/              # Bridge Baileys [Fase 9] - funcional
│   ├── plugins/                   # Plugin interface + Registry [Fase 16]
│   ├── shutdown/                  # Graceful shutdown LIFO
│   ├── skills/                    # SKILL.md loader [Fase 7]
│   └── tools/                     # Registry + types + tools nativos
│       ├── file/                  # read_file, write_file
│       ├── mcp/                   # MCP client [Fase 13]
│       ├── send/                  # send_message [Fase 14]
│       ├── skills/                # skills_list, skill_view
│       └── web/                   # web_search [Fase 3, stub]
├── bridge/
│   ├── bridge.js                  # HTTP bridge Node.js + Baileys (funcional)
│   ├── package.json
│   └── README.md
├── skills/                        # Skills de usuario (.md con frontmatter)
│   ├── responder-mensajes/
│   ├── gestionar-tareas/
│   └── resumir-conversacion/
├── test/stress/                   # Tests de carga
├── config.example.yaml
├── go.mod
├── Makefile
└── CLAUDE.md
```
