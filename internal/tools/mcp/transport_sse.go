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

// sseTransport implementa Transport sobre SSE (Server-Sent Events).
// Protocolo MCP legacy (pre-2025-03-26). Mantenido por compatibilidad.
// Envia POST directamente al endpoint base; los servidores SSE que implementan
// el protocolo MCP aceptan tanto SSE como POST directos.
// Fase 13.
type sseTransport struct {
	baseURL string
	http    *http.Client
	mu      sync.Mutex
}

func newSSETransport(baseURL string) *sseTransport {
	return &sseTransport{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 180 * time.Second},
	}
}

// Call envia una solicitud JSON-RPC via POST al endpoint SSE.
func (t *sseTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mcp sse marshal %s: %w", method, err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp sse build request %s: %w", method, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcp sse do %s: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mcp sse %s status %d", method, resp.StatusCode)
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("mcp sse decode %s: %w", method, err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcp rpc %s error %d: %s", method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (t *sseTransport) Close() error { return nil }
