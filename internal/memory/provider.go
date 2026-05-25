package memory

import (
	"context"
)

// InitOptions son los parametros de inicializacion de un provider.
type InitOptions struct {
	HermesHome    string
	Platform      string
	SessionID     string
	AgentIdentity string
	UserID        string
}

// Provider es la interfaz de un proveedor de memoria.
// El MemoryManager admite siempre el builtin y adicionalmente UN proveedor externo.
type Provider interface {
	Name() string
	IsAvailable() bool

	Initialize(ctx context.Context, opts InitOptions) error

	// SystemPromptBlock retorna texto estatico para inyectar en la capa stable del prompt.
	SystemPromptBlock() string

	// Prefetch retorna contexto relevante para el turno actual (debe ser rapido, consulta cache).
	Prefetch(ctx context.Context, query, sessionID string) string

	// QueuePrefetch lanza recall en background para el SIGUIENTE turno.
	QueuePrefetch(ctx context.Context, query, sessionID string)

	// SyncTurn persiste un turno completado (no bloqueante).
	SyncTurn(ctx context.Context, user, assistant, sessionID string)

	// ToolSchemas retorna los schemas de tools que este provider expone al LLM.
	ToolSchemas() []map[string]any

	// HandleToolCall despacha una tool call a este provider.
	HandleToolCall(ctx context.Context, name string, args map[string]any) (string, error)

	Shutdown(ctx context.Context) error
}

// Hooks opcionales — implementar para opt-in.

// SessionEndHook se llama cuando termina una sesion.
type SessionEndHook interface {
	OnSessionEnd(ctx context.Context, messages []map[string]any)
}

// PreCompressHook se llama antes de comprimir el contexto.
type PreCompressHook interface {
	OnPreCompress(ctx context.Context, messages []map[string]any) string
}

// MemoryWriteHook se llama cuando el builtin escribe una entry.
type MemoryWriteHook interface {
	OnMemoryWrite(ctx context.Context, action, target, content string, meta map[string]any)
}

// DelegationHook se llama cuando un subagente completa una tarea.
type DelegationHook interface {
	OnDelegation(ctx context.Context, task, result, childSessionID string)
}
