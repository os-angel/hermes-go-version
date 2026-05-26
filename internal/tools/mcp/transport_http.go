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
func (t *httpTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
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
		return nil, fmt.Errorf("mcp http marshal %s: %w", method, err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp http build request %s: %w", method, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcp http do %s: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mcp http %s status %d", method, resp.StatusCode)
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("mcp http decode %s: %w", method, err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcp rpc %s error %d: %s", method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (t *httpTransport) Close() error { return nil }
