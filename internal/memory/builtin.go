package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const entryDelimiter = "\n§\n"

// BuiltinProvider implementa Provider usando MEMORY.md y USER.md en disco.
// Fase 2.
type BuiltinProvider struct {
	dir             string
	memCharLimit    int
	userCharLimit   int
	memEntries      []string
	userEntries     []string
	snapshot        map[string]string // frozen al inicio de sesion
	mu              sync.Mutex
}

func NewBuiltinProvider(dir string, memLimit, userLimit int) *BuiltinProvider {
	if memLimit == 0 {
		memLimit = 2200
	}
	if userLimit == 0 {
		userLimit = 1375
	}
	return &BuiltinProvider{
		dir:           dir,
		memCharLimit:  memLimit,
		userCharLimit: userLimit,
		snapshot:      make(map[string]string),
	}
}

func (p *BuiltinProvider) Name() string        { return "builtin" }
func (p *BuiltinProvider) IsAvailable() bool   { return true }

func (p *BuiltinProvider) Initialize(ctx context.Context, opts InitOptions) error {
	// Fase 2: cargar MEMORY.md y USER.md, capturar snapshot.
	panic("not implemented: Phase 2")
}

func (p *BuiltinProvider) SystemPromptBlock() string {
	// Fase 2: retornar el snapshot frozen.
	panic("not implemented: Phase 2")
}

func (p *BuiltinProvider) Prefetch(_ context.Context, _, _ string) string { return "" }
func (p *BuiltinProvider) QueuePrefetch(_ context.Context, _, _ string)   {}

func (p *BuiltinProvider) SyncTurn(_ context.Context, _, _, _ string) {}

func (p *BuiltinProvider) ToolSchemas() []map[string]any {
	return []map[string]any{
		{
			"name":        "memory",
			"description": "Leer y escribir memoria persistente del agente.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type": "string",
						"enum": []string{"add", "replace", "remove", "read"},
					},
					"target": map[string]any{
						"type": "string",
						"enum": []string{"memory", "user"},
					},
					"content":  map[string]any{"type": "string"},
					"old_text": map[string]any{"type": "string"},
				},
				"required": []string{"action", "target"},
			},
		},
	}
}

func (p *BuiltinProvider) HandleToolCall(ctx context.Context, name string, args map[string]any) (string, error) {
	// Fase 2: despachar a add/replace/remove/read.
	panic("not implemented: Phase 2")
}

func (p *BuiltinProvider) Shutdown(_ context.Context) error { return nil }

// Add agrega una entry al target ("memory" o "user").
func (p *BuiltinProvider) Add(target, content string) (map[string]any, error) {
	// Fase 2.
	panic("not implemented: Phase 2")
}

// Replace reemplaza una entry que contiene oldSubstr.
func (p *BuiltinProvider) Replace(target, oldSubstr, newContent string) (map[string]any, error) {
	// Fase 2.
	panic("not implemented: Phase 2")
}

// Remove elimina la entry que contiene oldSubstr.
func (p *BuiltinProvider) Remove(target, oldSubstr string) (map[string]any, error) {
	// Fase 2.
	panic("not implemented: Phase 2")
}

// Read retorna el estado actual del target.
func (p *BuiltinProvider) Read(target string) (map[string]any, error) {
	// Fase 2.
	panic("not implemented: Phase 2")
}

// pathFor retorna la ruta del archivo para el target.
func (p *BuiltinProvider) pathFor(target string) string {
	if target == "user" {
		return filepath.Join(p.dir, "USER.md")
	}
	return filepath.Join(p.dir, "MEMORY.md")
}

// readFile lee y splitea entries desde disco.
func readMemFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil, nil
	}
	var entries []string
	for _, e := range strings.Split(raw, entryDelimiter) {
		if t := strings.TrimSpace(e); t != "" {
			entries = append(entries, t)
		}
	}
	return entries, nil
}

// writeFile escribe entries con atomic write (temp + rename).
func writeMemFile(path string, entries []string) error {
	content := strings.Join(entries, entryDelimiter)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o640); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	return os.Rename(tmp, path)
}
