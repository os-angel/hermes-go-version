package toolfile

import (
	"context"
	"fmt"
	"os"

	"hermes-go/internal/tools"
)

func init() {
	tools.Default().MustRegister(&tools.Tool{
		Name:        "read_file",
		Description: "Lee el contenido de un archivo del filesystem local.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Ruta absoluta del archivo."},
			},
			"required": []string{"path"},
		},
		Handler: readFile,
	})

	tools.Default().MustRegister(&tools.Tool{
		Name:        "write_file",
		Description: "Escribe contenido en un archivo del filesystem local.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			},
			"required": []string{"path", "content"},
		},
		Handler: writeFile,
	})
}

func readFile(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return `{"error":"path required"}`, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error()), nil
	}
	return string(data), nil
}

func writeFile(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return `{"error":"path required"}`, nil
	}
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error()), nil
	}
	return `{"success":true}`, nil
}
