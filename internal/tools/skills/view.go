package toolskills

import (
	"context"
	"encoding/json"
	"fmt"

	"hermes-go/internal/skills"
	"hermes-go/internal/tools"
)

// RegisterViewTool registra la tool "skill_view" en el registry.
func RegisterViewTool(reg *tools.Registry, loader *skills.Loader) {
	reg.MustRegister(&tools.Tool{
		Name:        "skill_view",
		Description: "Carga el contenido completo de un skill por nombre. Incluye instrucciones y archivos enlazados.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Nombre del skill a cargar.",
				},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name, _ := args["name"].(string)
			if name == "" {
				return `{"error":"name is required"}`, nil
			}
			ls, err := loader.Load(ctx, name)
			if err != nil {
				return "", fmt.Errorf("load skill: %w", err)
			}
			if ls == nil {
				return fmt.Sprintf(`{"error":"skill %q not found"}`, name), nil
			}
			b, _ := json.Marshal(map[string]any{
				"name":           ls.Name,
				"content":        ls.Content,
				"description":    ls.Description,
				"ready_status":   ls.ReadyStatus,
				"setup_note":     ls.SetupNote,
				"linked_files":   ls.LinkedData,
			})
			return string(b), nil
		},
	})
}
