package mcp

import "encoding/json"

// rpcRequest es una solicitud JSON-RPC 2.0.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// rpcResponse es una respuesta JSON-RPC 2.0.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError es el objeto de error en una respuesta JSON-RPC 2.0.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
