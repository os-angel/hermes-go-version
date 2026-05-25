package mcp

import (
	"context"
	"encoding/json"
)

// sseTransport implementa Transport sobre SSE (Server-Sent Events).
// Protocolo MCP legacy (pre-2025-03-26). Mantenido por compatibilidad.
// Fase 13.
type sseTransport struct {
	baseURL string
}

func newSSETransport(baseURL string) *sseTransport {
	return &sseTransport{baseURL: baseURL}
}

// Call envia una solicitud MCP via SSE.
// Pendiente - Fase 13.
func (t *sseTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	panic("not implemented: Phase 13")
}

func (t *sseTransport) Close() error { return nil }
