# hermes-go

Reimplementacion de Hermes (Python) en Go con alta concurrencia.
Agente AI con loop de conversacion, memoria persistente, skills, MCP, WhatsApp, Email, Webhook y REST API.

## Antes de cualquier tarea

1. Lee `Checkpoints.md` para saber en que fase esta el proyecto.
2. Lee `PLAN.md` (en el directorio padre `../PLAN.md`) para contexto completo.
3. No avanzar una fase sin pasar sus criterios de aceptacion.

## Comandos de desarrollo

```bash
make build        # compila el binario
make test         # tests rapidos
make race         # tests con race detector (obligatorio antes de merge)
make bench        # benchmarks
make stress       # tests de carga
make lint         # go vet + staticcheck
make run          # build + run
make tidy         # go mod tidy
```

## Arquitectura

```
IncomingMessage (WhatsApp/Email/Webhook/REST)
    -> PlatformRouter (channel + worker pool)
        -> SessionCache (LRU + TTL + disk)
            -> ConversationLoop (per-session goroutine)
                -> PromptBuilder (3 capas: stable/context/volatile)
                -> LLM Client (OpenAI-compatible)
                -> ToolExecutor (errgroup + semaforo max 8)
                    -> Tool Registry (nativos + MCP)
                    -> Memory Manager (builtin + externo opcional)
                -> Sender (por plataforma)
```

## Convenciones

- Sin emojis en codigo, logs ni documentacion.
- Errores con `fmt.Errorf("...: %w", err)`. NUNCA panic en produccion.
- `context.Context` como primer parametro en funciones con I/O.
- Logging estructurado con `slog`. Nunca `fmt.Println`.
- Campos minimos de log: `session_id`, `platform`, `tool`, `duration_ms`.
- Todos los tests con `-race`.
- Cobertura > 70% por paquete.
- Sin globals mutables salvo `tools.Default()` y `plugins.Default()`.
- Todo lo demas se inyecta por constructor.
- `go vet ./...` y `staticcheck ./...` deben pasar en CI.

## Modulo Go

`module hermes-go` — todos los imports internos son `hermes-go/internal/...`

## Directorio de datos

`~/.hermes-go/` (override: `HERMES_GO_HOME` env var)

```
~/.hermes-go/
├── memories/MEMORY.md          # notas del agente (delimiter: §)
├── memories/USER.md            # perfil del usuario
├── sessions/                   # historial JSON por sesion (PII hasheado)
├── skills/                     # skills de usuario
├── cron/jobs.json              # jobs del scheduler
├── cron/output/                # outputs de jobs
├── whatsapp/bridge.pid         # PID del bridge Node.js
├── whatsapp/auth_state/        # credenciales Baileys
└── logs/                       # logs estructurados
```

## Cuando termines de implementar una fase

1. Marca su checkbox en `Checkpoints.md`.
2. Agrega fecha y nota breve.
3. Confirma que `make race` pasa antes de marcar como completa.
