package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// httpTransport implementa Transport sobre StreamableHTTP (MCP spec 2025-03-26).
// Un mutex serializa las llamadas RPC por servidor, igual que Hermes Python.
// Fase 13.
type httpTransport struct {
	baseURL string
	http    *http.Client
	mu      sync.Mutex
}

func newHTTPTransport(baseURL string) *httpTransport {
	return &httpTransport{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Call envia una solicitud JSON-RPC via POST y espera la respuesta.
// Pendiente - Fase 13.
func (t *httpTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp http build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	_ = httpReq
	panic("not implemented: Phase 13")
}

func (t *httpTransport) Close() error { return nil }
