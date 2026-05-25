package webhook

import (
	"encoding/json"
	"os"
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

// Upsert agrega o actualiza una suscripcion y persiste.
// Pendiente - Fase 11.
func (s *SubscriptionStore) Upsert(sub *Subscription) error {
	panic("not implemented: Phase 11")
}

// Delete elimina una suscripcion y persiste.
// Pendiente - Fase 11.
func (s *SubscriptionStore) Delete(id string) error {
	panic("not implemented: Phase 11")
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
