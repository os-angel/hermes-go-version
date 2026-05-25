package whatsapp

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// Identity resuelve JIDs LID -> phone para mantener consistencia de sesion.
// Fase 9.
type Identity struct {
	mappingPath string
	mapping     map[string]string
	mu          sync.RWMutex
}

func NewIdentity(mappingPath string) (*Identity, error) {
	id := &Identity{mappingPath: mappingPath, mapping: make(map[string]string)}
	if err := id.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return id, nil
}

// Canonical retorna el JID canonico (phone si hay mapping, original si no).
func (id *Identity) Canonical(jid string) string {
	id.mu.RLock()
	defer id.mu.RUnlock()
	if phone, ok := id.mapping[strings.ToLower(jid)]; ok {
		return phone
	}
	return jid
}

// Update agrega o actualiza un mapping lid -> phone y persiste.
func (id *Identity) Update(lid, phone string) error {
	id.mu.Lock()
	id.mapping[strings.ToLower(lid)] = phone
	id.mu.Unlock()
	return id.save()
}

func (id *Identity) load() error {
	data, err := os.ReadFile(id.mappingPath)
	if err != nil {
		return err
	}
	id.mu.Lock()
	defer id.mu.Unlock()
	return json.Unmarshal(data, &id.mapping)
}

func (id *Identity) save() error {
	id.mu.RLock()
	data, err := json.MarshalIndent(id.mapping, "", "  ")
	id.mu.RUnlock()
	if err != nil {
		return err
	}
	tmp := id.mappingPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, id.mappingPath)
}
