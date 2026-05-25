package memory

import (
	"context"
	"encoding/json"

	"hermes-go/internal/tools"
)

// RegisterMemoryTool registra la tool "memory" en el registry dado.
// Debe llamarse despues de crear el BuiltinProvider.
func RegisterMemoryTool(reg *tools.Registry, p *BuiltinProvider) {
	reg.MustRegister(&tools.Tool{
		Name:        "memory",
		Description: "Leer y escribir memoria persistente. Targets: 'memory' (notas del agente) o 'user' (perfil del usuario). Acciones: add, replace, remove, read.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action":   map[string]any{"type": "string", "enum": []string{"add", "replace", "remove", "read"}},
				"target":   map[string]any{"type": "string", "enum": []string{"memory", "user"}},
				"content":  map[string]any{"type": "string", "description": "Contenido a agregar o reemplazar."},
				"old_text": map[string]any{"type": "string", "description": "Substring de la entry a reemplazar o eliminar."},
			},
			"required": []string{"action", "target"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return p.HandleToolCall(ctx, "memory", args)
		},
	})
}

func jsonResult(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
