package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

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
		// Si no hay transport pero hay command, usar stdio
		if cfg.Command != "" {
			env := FilterEnv(cfg.EnvAllowList)
			t, err = newStdioTransport(cfg.Command, cfg.Args, env)
		} else if cfg.URL != "" {
			t = newHTTPTransport(cfg.URL)
		} else {
			return nil, fmt.Errorf("unknown mcp transport: %s", cfg.Transport)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("mcp connect %s: %w", cfg.Name, err)
	}
	return &ServerClient{name: cfg.Name, cfg: cfg, transport: t}, nil
}

// Initialize llama al metodo MCP initialize para obtener capacidades del servidor.
func (c *ServerClient) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"clientInfo": map[string]any{
			"name":    "hermes-go",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{},
	}
	_, err := c.transport.Call(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("mcp initialize %s: %w", c.name, err)
	}
	slog.Info("mcp server initialized", "server", c.name)
	return nil
}

// ListTools obtiene los tools expuestos por el servidor y los registra en reg.
func (c *ServerClient) ListTools(ctx context.Context, reg *tools.Registry) error {
	result, err := c.transport.Call(ctx, "tools/list", nil)
	if err != nil {
		return fmt.Errorf("mcp list_tools %s: %w", c.name, err)
	}

	var resp struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return fmt.Errorf("mcp list_tools decode %s: %w", c.name, err)
	}

	serverSlug := sanitizeMCPName(c.name)
	for _, t := range resp.Tools {
		toolSlug := sanitizeMCPName(t.Name)
		toolName := "mcp_" + serverSlug + "_" + toolSlug

		// Capturar variables para la closure
		remoteName := t.Name
		client := c

		if err := reg.Register(&tools.Tool{
			Name:        toolName,
			Description: t.Description,
			Schema:      t.InputSchema,
			Parallel:    c.cfg.SupportsParallelToolCalls,
			Handler: func(ctx context.Context, args map[string]any) (string, error) {
				return client.CallTool(ctx, remoteName, args)
			},
		}); err != nil {
			slog.Warn("mcp tool registration skipped", "tool", toolName, "server", c.name, "err", err)
		}
	}

	slog.Info("mcp tools registered", "server", c.name, "count", len(resp.Tools))
	return nil
}

// CallTool invoca un tool en el servidor MCP remoto.
func (c *ServerClient) CallTool(ctx context.Context, toolName string, args map[string]any) (string, error) {
	timeout := c.cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	tCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	params := map[string]any{
		"name":      toolName,
		"arguments": args,
	}
	result, err := c.transport.Call(tCtx, "tools/call", params)
	if err != nil {
		return "", fmt.Errorf("mcp call %s/%s: %w", c.name, toolName, err)
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		// Si no se puede parsear, devolver raw
		return string(result), nil
	}

	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	text := strings.Join(parts, "\n")
	if resp.IsError {
		return "", fmt.Errorf("mcp tool error %s/%s: %s", c.name, toolName, text)
	}
	return text, nil
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

// Connect crea e inicializa un ServerClient. Si falla la conexion inicial,
// lanza un reconnectLoop en background.
func (m *Manager) Connect(ctx context.Context, cfg config.MCPServerConfig) error {
	client, err := NewServerClient(cfg)
	if err != nil {
		return fmt.Errorf("mcp new server client %s: %w", cfg.Name, err)
	}

	connectFn := func() error {
		tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		if err := client.Initialize(tCtx); err != nil {
			return err
		}
		return client.ListTools(tCtx, m.reg)
	}

	if err := connectFn(); err != nil {
		slog.Warn("mcp initial connect failed, retrying in background", "server", cfg.Name, "err", err)
		go reconnectLoop(ctx, cfg.Name, connectFn)
	}

	m.servers = append(m.servers, client)
	return nil
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

// sanitizeMCPName convierte un nombre MCP en un identificador Go valido,
// reemplazando caracteres no alfanumericos con guion bajo.
func sanitizeMCPName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
