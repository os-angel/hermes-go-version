package toolweb

import (
	"context"
	"fmt"

	"hermes-go/internal/tools"
)

func init() {
	tools.Default().MustRegister(&tools.Tool{
		Name:        "web_search",
		Description: "Busca informacion en internet.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Terminos de busqueda."},
			},
			"required": []string{"query"},
		},
		Handler: webSearch,
		Available: func() bool {
			// Fase 3: verificar que haya API key configurada
			return false // stub: deshabilitada hasta Fase 3
		},
	})
}

func webSearch(_ context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	return "", fmt.Errorf("web_search not configured (query: %q)", query)
}
