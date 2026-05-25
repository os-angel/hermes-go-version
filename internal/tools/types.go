package tools

import "context"

// ToolHandler es la firma de cualquier handler de tool.
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool representa una capacidad registrada en el Registry.
type Tool struct {
	Name        string
	Description string
	// Schema es el JSON Schema de los parametros (campo "parameters" del OpenAI function calling).
	Schema    map[string]any
	Handler   ToolHandler
	// Parallel indica si esta tool puede ejecutarse junto a otras en el mismo turno.
	Parallel  bool
	// Available retorna false si la tool no esta lista (env vars faltantes, etc.).
	// nil significa siempre disponible.
	Available func() bool
}

// IsAvailable retorna true si la tool esta lista para usar.
func (t *Tool) IsAvailable() bool {
	if t.Available == nil {
		return true
	}
	return t.Available()
}
