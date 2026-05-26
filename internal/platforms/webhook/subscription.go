package webhook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Subscription define una fuente de webhooks entrantes con su secreto HMAC.
// Fase 11.
type Subscription struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`    // ruta HTTP: /webhooks/{path}
	Secret  string `json:"secret"`  // secreto HMAC-SHA256
	Enabled bool   `json:"enabled"`
}

// SubscriptionStore persiste las suscripciones en un archivo JSON.
type SubscriptionStore struct {
	mu   sync.RWMutex
	path string
	subs map[string]*Subscription
}

func NewSubscriptionStore(path string) (*SubscriptionStore, error) {
	s := &SubscriptionStore{path: path, subs: make(map[string]*Subscription)}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *SubscriptionStore) Get(id string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subs[id]
	return sub, ok
}

func (s *SubscriptionStore) List() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Subscription, 0, len(s.subs))
	for _, v := range s.subs {
		out = append(out, v)
	}
	return out
}

// Upsert agrega o actualiza una suscripcion y persiste atomicamente.
func (s *SubscriptionStore) Upsert(sub *Subscription) error {
	s.mu.Lock()
	s.subs[sub.ID] = sub
	data, err := json.Marshal(s.subs)
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("webhook store marshal: %w", err)
	}
	return webhookAtomicWrite(s.path, data)
}

// Delete elimina una suscripcion y persiste atomicamente.
func (s *SubscriptionStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.subs, id)
	data, err := json.Marshal(s.subs)
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("webhook store marshal: %w", err)
	}
	return webhookAtomicWrite(s.path, data)
}

func (s *SubscriptionStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.subs)
}

// webhookAtomicWrite escribe data al path usando write-to-temp-then-rename.
func webhookAtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	return os.Rename(tmpPath, path)
}
