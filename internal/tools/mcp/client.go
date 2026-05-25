package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hermes-go/internal/config"
	"hermes-go/internal/tools"
)

// Transport abstrae los tres modos de transporte MCP.
type Transport interface {
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)
	Close() error
}

// ServerClient gestiona la conexion a un servidor MCP y expone sus tools.
// Un mutex en el transport serializa todas las llamadas RPC al mismo servidor,
// replicando el patron del cliente MCP de Hermes Python (event loop dedicado).
// Fase 13.
type ServerClient struct {
	name      string
	cfg       config.MCPServerConfig
	transport Transport
}

// NewServerClient crea el cliente segun el tipo de transporte configurado.
// Pendiente - Fase 13.
func NewServerClient(cfg config.MCPServerConfig) (*ServerClient, error) {
	var t Transport
	var err error

	switch cfg.Transport {
	case "stdio":
		env := FilterEnv(cfg.EnvAllowList)
		t, err = newStdioTransport(cfg.Command, cfg.Args, env)
	case "http", "streamable_http":
		t = newHTTPTransport(cfg.URL)
	case "sse":
		t = newSSETransport(cfg.URL)
	default:
		return nil, fmt.Errorf("unknown mcp transport: %s", cfg.Transport)
	}
	if err != nil {
		return nil, fmt.Errorf("mcp connect %s: %w", cfg.Name, err)
	}
	return &ServerClient{name: cfg.Name, cfg: cfg, transport: t}, nil
}

// Initialize llama al metodo MCP initialize para obtener capacidades.
// Pendiente - Fase 13.
func (c *ServerClient) Initialize(ctx context.Context) error {
	panic("not implemented: Phase 13")
}

// ListTools obtiene los tools expuestos por el servidor y los registra en reg.
// Pendiente - Fase 13.
func (c *ServerClient) ListTools(ctx context.Context, reg *tools.Registry) error {
	panic("not implemented: Phase 13")
}

// CallTool invoca un tool en el servidor MCP remoto.
// Pendiente - Fase 13.
func (c *ServerClient) CallTool(ctx context.Context, toolName string, args map[string]any) (string, error) {
	panic("not implemented: Phase 13")
}

// Close cierra el transporte.
func (c *ServerClient) Close() error {
	slog.Info("mcp server close", "server", c.name)
	return c.transport.Close()
}

// Manager gestiona multiples ServerClient y los registra en el Registry de tools.
// Fase 13.
type Manager struct {
	servers []*ServerClient
	reg     *tools.Registry
}

func NewManager(reg *tools.Registry) *Manager {
	return &Manager{reg: reg}
}

// Connect crea e inicializa un ServerClient, con reconexion automatica en background.
// Pendiente - Fase 13.
func (m *Manager) Connect(ctx context.Context, cfg config.MCPServerConfig) error {
	panic("not implemented: Phase 13")
}

// Shutdown cierra todos los servidores MCP.
func (m *Manager) Shutdown(_ context.Context) error {
	var lastErr error
	for _, s := range m.servers {
		if err := s.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
