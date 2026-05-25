package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// Manager orquesta el proveedor builtin y opcionalmente uno externo.
// Solo se admite UN proveedor externo a la vez.
type Manager struct {
	builtin   Provider
	external  Provider
	toolIndex map[string]Provider
	mu        sync.RWMutex
}

func NewManager(builtin Provider) *Manager {
	m := &Manager{
		builtin:   builtin,
		toolIndex: make(map[string]Provider),
	}
	for _, schema := range builtin.ToolSchemas() {
		if name, ok := schema["name"].(string); ok {
			m.toolIndex[name] = builtin
		}
	}
	return m
}

// AddExternal registra un proveedor externo. Retorna error si ya hay uno.
func (m *Manager) AddExternal(p Provider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.external != nil {
		return fmt.Errorf("external provider %q already registered; only one allowed", m.external.Name())
	}
	m.external = p
	for _, schema := range p.ToolSchemas() {
		if name, ok := schema["name"].(string); ok {
			if _, exists := m.toolIndex[name]; !exists {
				m.toolIndex[name] = p
			} else {
				slog.Warn("memory tool name conflict", "tool", name, "provider", p.Name())
			}
		}
	}
	return nil
}

func (m *Manager) providers() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.external != nil {
		return []Provider{m.builtin, m.external}
	}
	return []Provider{m.builtin}
}

// BuildSystemPrompt concatena los bloques estaticos de todos los providers.
func (m *Manager) BuildSystemPrompt() string {
	var parts []string
	for _, p := range m.providers() {
		if b := p.SystemPromptBlock(); strings.TrimSpace(b) != "" {
			parts = append(parts, b)
		}
	}
	return strings.Join(parts, "\n\n")
}

// Prefetch agrupa el contexto relevante de todos los providers.
func (m *Manager) Prefetch(ctx context.Context, query, sessionID string) string {
	var parts []string
	for _, p := range m.providers() {
		if r := p.Prefetch(ctx, query, sessionID); strings.TrimSpace(r) != "" {
			parts = append(parts, r)
		}
	}
	return strings.Join(parts, "\n\n")
}

// QueuePrefetch lanza recall en background en todos los providers.
func (m *Manager) QueuePrefetch(ctx context.Context, query, sessionID string) {
	for _, p := range m.providers() {
		p.QueuePrefetch(ctx, query, sessionID)
	}
}

// SyncTurn sincroniza el turno completado en todos los providers.
func (m *Manager) SyncTurn(ctx context.Context, user, assistant, sessionID string) {
	for _, p := range m.providers() {
		p.SyncTurn(ctx, user, assistant, sessionID)
	}
}

// ToolSchemas retorna los schemas de todos los providers.
func (m *Manager) ToolSchemas() []map[string]any {
	seen := make(map[string]bool)
	var schemas []map[string]any
	for _, p := range m.providers() {
		for _, s := range p.ToolSchemas() {
			if name, ok := s["name"].(string); ok && !seen[name] {
				schemas = append(schemas, s)
				seen[name] = true
			}
		}
	}
	return schemas
}

// HandleToolCall enruta al provider correcto.
func (m *Manager) HandleToolCall(ctx context.Context, name string, args map[string]any) (string, error) {
	m.mu.RLock()
	p, ok := m.toolIndex[name]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("no memory provider handles tool %q", name)
	}
	return p.HandleToolCall(ctx, name, args)
}

// Shutdown cierra todos los providers.
func (m *Manager) Shutdown(ctx context.Context) error {
	for _, p := range m.providers() {
		if err := p.Shutdown(ctx); err != nil {
			slog.Warn("memory provider shutdown error", "provider", p.Name(), "err", err)
		}
	}
	return nil
}
