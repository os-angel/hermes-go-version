package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hermes-go/internal/config"
)

// Store persiste credenciales OAuth en ~/.hermes-go/auth.json.
type Store struct {
	mu   sync.RWMutex
	path string
	data map[string]*Credential // provider name -> credential
}

var (
	defaultStore *Store
	storeOnce   sync.Once
)

// Default retorna el store global, inicializado desde disco.
func Default() *Store {
	storeOnce.Do(func() {
		s := &Store{
			path: filepath.Join(config.Home(), "auth.json"),
			data: make(map[string]*Credential),
		}
		_ = s.load() // ignorar error si no existe
		defaultStore = s
	})
	return defaultStore
}

// Get retorna la credencial para un proveedor, nil si no existe.
func (s *Store) Get(providerName string) *Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[providerName]
}

// Save guarda o actualiza la credencial de un proveedor.
func (s *Store) Save(cred *Credential) error {
	cred.SavedAt = time.Now()
	s.mu.Lock()
	s.data[cred.ProviderName] = cred
	s.mu.Unlock()
	return s.persist()
}

// Delete elimina la credencial de un proveedor.
func (s *Store) Delete(providerName string) error {
	s.mu.Lock()
	delete(s.data, providerName)
	s.mu.Unlock()
	return s.persist()
}

// List retorna todas las credenciales almacenadas.
func (s *Store) List() []*Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Credential, 0, len(s.data))
	for _, c := range s.data {
		out = append(out, c)
	}
	return out
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth store read: %w", err)
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) persist() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("auth store marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("auth store mkdir: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("auth store write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("auth store rename: %w", err)
	}
	return nil
}
