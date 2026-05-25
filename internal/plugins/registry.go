package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"hermes-go/internal/tools"
)

// Registry mantiene todos los plugins registrados.
// El singleton Default() se usa en los init() de cada plugin.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

var defaultRegistry = &Registry{plugins: make(map[string]Plugin)}

// Default retorna el registry global de plugins.
func Default() *Registry { return defaultRegistry }

// Register agrega un plugin. Llamado desde init() de cada plugin.
func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plugins[p.Name()]; exists {
		panic(fmt.Sprintf("plugin already registered: %s", p.Name()))
	}
	r.plugins[p.Name()] = p
	slog.Debug("plugin registered", "name", p.Name())
}

// InitAll inicializa todos los plugins registrados.
func (r *Registry) InitAll(ctx context.Context, reg *tools.Registry) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, p := range r.plugins {
		if err := p.Init(ctx, reg); err != nil {
			return fmt.Errorf("plugin %s init: %w", name, err)
		}
		slog.Info("plugin initialized", "name", name)
	}
	return nil
}

// ShutdownAll llama Shutdown en todos los plugins.
func (r *Registry) ShutdownAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var lastErr error
	for _, p := range r.plugins {
		if err := p.Shutdown(ctx); err != nil {
			lastErr = err
			slog.Error("plugin shutdown error", "name", p.Name(), "err", err)
		}
	}
	return lastErr
}

// List retorna los nombres de todos los plugins registrados.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
