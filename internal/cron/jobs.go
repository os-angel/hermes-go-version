package cron

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Job define una tarea periodica configurada por el usuario.
// Equivalente a los jobs de Hermes Python (cron/jobs.py).
type Job struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Schedule    string    `json:"schedule"`    // expresion cron: "*/5 * * * *"
	Prompt      string    `json:"prompt"`      // prompt a enviar al agente
	Platform    string    `json:"platform"`    // plataforma destino
	ChatID      string    `json:"chat_id"`     // chat destino
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	LastRunAt   time.Time `json:"last_run_at,omitempty"`
	NextRunAt   time.Time `json:"next_run_at,omitempty"`
}

// Store persiste los jobs en un archivo JSON con flock para evitar corrupcion.
type Store struct {
	mu   sync.RWMutex
	path string
	jobs map[string]*Job
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, jobs: make(map[string]*Job)}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

// Upsert agrega o actualiza un job y persiste.
// Pendiente - Fase 15.
func (s *Store) Upsert(j *Job) error {
	panic("not implemented: Phase 15")
}

// Delete elimina un job y persiste.
// Pendiente - Fase 15.
func (s *Store) Delete(id string) error {
	panic("not implemented: Phase 15")
}

// UpdateLastRun actualiza el timestamp de ultima ejecucion.
// Pendiente - Fase 15.
func (s *Store) UpdateLastRun(id string, t time.Time) error {
	panic("not implemented: Phase 15")
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.jobs)
}
