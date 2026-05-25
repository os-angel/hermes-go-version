package tools

import (
	"context"
	"fmt"
	"sync"
)

// Registry mantiene el catalogo de tools disponibles.
type Registry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

var defaultRegistry = NewRegistry()

// Default retorna el singleton global del registry.
func Default() *Registry { return defaultRegistry }

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]*Tool)}
}

// Register agrega una tool. Retorna error si el nombre ya existe.
func (r *Registry) Register(t *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[t.Name]; ok {
		return fmt.Errorf("tool %q already registered", t.Name)
	}
	r.tools[t.Name] = t
	return nil
}

// MustRegister registra o hace panic. Para uso en init().
func (r *Registry) MustRegister(t *Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// Unregister elimina una tool.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get retorna la tool por nombre. nil si no existe.
func (r *Registry) Get(name string) *Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// List retorna los nombres de todas las tools registradas.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for k := range r.tools {
		names = append(names, k)
	}
	return names
}

// Schemas retorna los schemas de tools disponibles (Available() == true)
// en formato compatible con la API de OpenAI.
func (r *Registry) Schemas() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schemas := make([]map[string]any, 0, len(r.tools))
	for _, t := range r.tools {
		if !t.IsAvailable() {
			continue
		}
		schemas = append(schemas, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Schema,
			},
		})
	}
	return schemas
}

// Execute invoca el handler de la tool. Retorna error si no existe.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return t.Handler(ctx, args)
}
