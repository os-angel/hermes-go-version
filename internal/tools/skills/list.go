package toolskills

import (
	"context"
	"encoding/json"

	"hermes-go/internal/skills"
	"hermes-go/internal/tools"
)

// RegisterListTool registra la tool "skills_list" en el registry.
func RegisterListTool(reg *tools.Registry, loader *skills.Loader) {
	reg.MustRegister(&tools.Tool{
		Name:        "skills_list",
		Description: "Lista los skills disponibles (solo metadata). Usar antes de skill_view para descubrir que skills existen.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"category": map[string]any{
					"type":        "string",
					"description": "Filtrar por categoria (opcional).",
				},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			all := loader.List(ctx)
			var result []map[string]any
			cat, _ := args["category"].(string)
			for _, s := range all {
				if cat != "" && s.Category != cat {
					continue
				}
				result = append(result, map[string]any{
					"name":        s.Name,
					"description": s.Description,
					"category":    s.Category,
				})
			}
			b, _ := json.Marshal(map[string]any{"skills": result})
			return string(b), nil
		},
	})
}
