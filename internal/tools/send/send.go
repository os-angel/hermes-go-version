package send

import (
	"context"
	"fmt"

	"hermes-go/internal/platforms"
	"hermes-go/internal/tools"
)

// RegisterSendTool registra el tool send_message en el registry dado.
// Permite al agente enviar mensajes proactivos a cualquier chat/sesion activa.
// Fase 14.
func RegisterSendTool(reg *tools.Registry, router *platforms.Router) {
	reg.MustRegister(&tools.Tool{
		Name:        "send_message",
		Description: "Envia un mensaje a un chat o sesion especifica.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"platform": map[string]any{
					"type":        "string",
					"description": "Plataforma destino (whatsapp, email, etc.).",
				},
				"chat_id": map[string]any{
					"type":        "string",
					"description": "ID del chat o direccion de destino.",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "Texto del mensaje.",
				},
			},
			"required": []string{"platform", "chat_id", "message"},
		},
		Parallel: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			platform, _ := args["platform"].(string)
			chatID, _ := args["chat_id"].(string)
			message, _ := args["message"].(string)
			if platform == "" || chatID == "" || message == "" {
				return "", fmt.Errorf("send_message: platform, chat_id y message son requeridos")
			}
			if err := router.Send(ctx, platforms.OutgoingMessage{
				Platform: platform,
				ChatID:   chatID,
				Text:     message,
			}); err != nil {
				return "", fmt.Errorf("send_message: %w", err)
			}
			return "message sent", nil
		},
	})
}
