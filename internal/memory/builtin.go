package memory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const entryDelimiter = "\n§\n"

// BuiltinProvider implementa Provider usando MEMORY.md y USER.md en disco.
type BuiltinProvider struct {
	dir           string
	memCharLimit  int
	userCharLimit int
	memEntries    []string
	userEntries   []string
	snapshot      struct {
		memory string
		user   string
	}
	mu sync.Mutex
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
	}
}

func (p *BuiltinProvider) Name() string      { return "builtin" }
func (p *BuiltinProvider) IsAvailable() bool { return true }

// Initialize carga los archivos de disco y captura el snapshot frozen para esta sesion.
func (p *BuiltinProvider) Initialize(_ context.Context, _ InitOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	p.memEntries, err = readMemFile(p.pathFor("memory"))
	if err != nil {
		return fmt.Errorf("builtin memory init: %w", err)
	}
	p.userEntries, err = readMemFile(p.pathFor("user"))
	if err != nil {
		return fmt.Errorf("builtin user init: %w", err)
	}

	p.snapshot.memory = strings.Join(p.memEntries, entryDelimiter)
	p.snapshot.user = strings.Join(p.userEntries, entryDelimiter)
	return nil
}

// SystemPromptBlock retorna el snapshot frozen capturado al inicio de la sesion.
func (p *BuiltinProvider) SystemPromptBlock() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var parts []string
	if p.snapshot.memory != "" {
		parts = append(parts, "## Memoria del agente\n\n"+p.snapshot.memory)
	}
	if p.snapshot.user != "" {
		parts = append(parts, "## Perfil del usuario\n\n"+p.snapshot.user)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

func (p *BuiltinProvider) Prefetch(_ context.Context, _, _ string) string { return "" }
func (p *BuiltinProvider) QueuePrefetch(_ context.Context, _, _ string)   {}
func (p *BuiltinProvider) SyncTurn(_ context.Context, _, _, _ string)     {}

func (p *BuiltinProvider) ToolSchemas() []map[string]any {
	return []map[string]any{
		{
			"name":        "memory",
			"description": "Leer y escribir memoria persistente del agente. Usa 'memory' para notas del agente y 'user' para el perfil del usuario.",
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
					"content":  map[string]any{"type": "string", "description": "Contenido a agregar o nuevo contenido para replace"},
					"old_text": map[string]any{"type": "string", "description": "Texto a buscar para replace/remove"},
				},
				"required": []string{"action", "target"},
			},
		},
	}
}

// HandleToolCall despacha las acciones de la tool "memory".
func (p *BuiltinProvider) HandleToolCall(_ context.Context, name string, args map[string]any) (string, error) {
	if name != "memory" {
		return "", fmt.Errorf("builtin provider no maneja la tool %q", name)
	}

	action, _ := args["action"].(string)
	target, _ := args["target"].(string)
	content, _ := args["content"].(string)
	oldText, _ := args["old_text"].(string)

	if target != "memory" && target != "user" {
		return "", fmt.Errorf("target invalido: %q (debe ser 'memory' o 'user')", target)
	}

	switch action {
	case "add":
		result, err := p.Add(target, content)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		return fmt.Sprintf("ok: entrada agregada a %s (%d chars total)", target, result["total_chars"]), nil

	case "replace":
		result, err := p.Replace(target, oldText, content)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		return fmt.Sprintf("ok: entrada reemplazada en %s (%d entradas)", target, result["count"]), nil

	case "remove":
		result, err := p.Remove(target, oldText)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		return fmt.Sprintf("ok: entrada eliminada de %s (%d entradas restantes)", target, result["count"]), nil

	case "read":
		result, err := p.Read(target)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		if result["content"] == "" {
			return fmt.Sprintf("(%s esta vacio)", target), nil
		}
		return result["content"].(string), nil

	default:
		return "", fmt.Errorf("accion invalida: %q", action)
	}
}

func (p *BuiltinProvider) Shutdown(_ context.Context) error { return nil }

// Add agrega una entry al target. Aplica scan anti-injection, char limit y escribe a disco.
func (p *BuiltinProvider) Add(target, content string) (map[string]any, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content no puede estar vacio")
	}
	if err := Scan(content); err != nil {
		return nil, fmt.Errorf("scan blocked: %w", err)
	}

	path := p.pathFor(target)
	lock, err := AcquireLock(path)
	if err != nil {
		return nil, fmt.Errorf("add lock: %w", err)
	}
	defer lock.Release()

	entries, err := readMemFile(path)
	if err != nil {
		return nil, err
	}
	entries = append(entries, strings.TrimSpace(content))
	entries = p.enforceCharLimit(entries, p.limitFor(target))

	if err := writeMemFile(path, entries); err != nil {
		return nil, err
	}

	p.mu.Lock()
	if target == "user" {
		p.userEntries = entries
	} else {
		p.memEntries = entries
	}
	p.mu.Unlock()

	total := len(strings.Join(entries, entryDelimiter))
	slog.Debug("memory add", "target", target, "entries", len(entries), "total_chars", total)
	return map[string]any{"total_chars": total, "count": len(entries)}, nil
}

// Replace reemplaza la primera entry que contiene oldSubstr.
func (p *BuiltinProvider) Replace(target, oldSubstr, newContent string) (map[string]any, error) {
	if strings.TrimSpace(oldSubstr) == "" {
		return nil, fmt.Errorf("old_text no puede estar vacio")
	}
	if err := Scan(newContent); err != nil {
		return nil, fmt.Errorf("scan blocked: %w", err)
	}

	path := p.pathFor(target)
	lock, err := AcquireLock(path)
	if err != nil {
		return nil, fmt.Errorf("replace lock: %w", err)
	}
	defer lock.Release()

	entries, err := readMemFile(path)
	if err != nil {
		return nil, err
	}

	replaced := false
	for i, e := range entries {
		if strings.Contains(e, oldSubstr) {
			entries[i] = strings.TrimSpace(newContent)
			replaced = true
			break
		}
	}
	if !replaced {
		return nil, fmt.Errorf("old_text %q no encontrado en %s", oldSubstr, target)
	}

	if err := writeMemFile(path, entries); err != nil {
		return nil, err
	}

	p.mu.Lock()
	if target == "user" {
		p.userEntries = entries
	} else {
		p.memEntries = entries
	}
	p.mu.Unlock()

	return map[string]any{"count": len(entries)}, nil
}

// Remove elimina la primera entry que contiene oldSubstr.
func (p *BuiltinProvider) Remove(target, oldSubstr string) (map[string]any, error) {
	if strings.TrimSpace(oldSubstr) == "" {
		return nil, fmt.Errorf("old_text no puede estar vacio")
	}

	path := p.pathFor(target)
	lock, err := AcquireLock(path)
	if err != nil {
		return nil, fmt.Errorf("remove lock: %w", err)
	}
	defer lock.Release()

	entries, err := readMemFile(path)
	if err != nil {
		return nil, err
	}

	var filtered []string
	removed := false
	for _, e := range entries {
		if !removed && strings.Contains(e, oldSubstr) {
			removed = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !removed {
		return nil, fmt.Errorf("old_text %q no encontrado en %s", oldSubstr, target)
	}

	if err := writeMemFile(path, filtered); err != nil {
		return nil, err
	}

	p.mu.Lock()
	if target == "user" {
		p.userEntries = filtered
	} else {
		p.memEntries = filtered
	}
	p.mu.Unlock()

	return map[string]any{"count": len(filtered)}, nil
}

// Read retorna el estado actual del target desde disco (no el snapshot).
func (p *BuiltinProvider) Read(target string) (map[string]any, error) {
	p.mu.Lock()
	var entries []string
	if target == "user" {
		entries = p.userEntries
	} else {
		entries = p.memEntries
	}
	p.mu.Unlock()

	content := strings.Join(entries, entryDelimiter)
	return map[string]any{
		"content": content,
		"count":   len(entries),
	}, nil
}

// pathFor retorna la ruta del archivo para el target.
func (p *BuiltinProvider) pathFor(target string) string {
	if target == "user" {
		return filepath.Join(p.dir, "USER.md")
	}
	return filepath.Join(p.dir, "MEMORY.md")
}

func (p *BuiltinProvider) limitFor(target string) int {
	if target == "user" {
		return p.userCharLimit
	}
	return p.memCharLimit
}

// enforceCharLimit trunca las entries mas antiguas si se supera el limite.
func (p *BuiltinProvider) enforceCharLimit(entries []string, limit int) []string {
	for {
		total := len(strings.Join(entries, entryDelimiter))
		if total <= limit || len(entries) <= 1 {
			break
		}
		slog.Debug("memory char limit: removing oldest entry", "total", total, "limit", limit)
		entries = entries[1:] // drop la mas antigua
	}
	return entries
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
