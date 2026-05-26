package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// Store persiste los jobs en un archivo JSON.
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

// Upsert agrega o actualiza un job y persiste atomicamente.
func (s *Store) Upsert(j *Job) error {
	s.mu.Lock()
	s.jobs[j.ID] = j
	data, err := json.Marshal(s.jobs)
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("cron store marshal: %w", err)
	}
	return cronAtomicWrite(s.path, data)
}

// Delete elimina un job y persiste atomicamente.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	delete(s.jobs, id)
	data, err := json.Marshal(s.jobs)
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("cron store marshal: %w", err)
	}
	return cronAtomicWrite(s.path, data)
}

// UpdateLastRun actualiza el timestamp de ultima ejecucion y persiste.
func (s *Store) UpdateLastRun(id string, t time.Time) error {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok {
		j.LastRunAt = t
	}
	data, err := json.Marshal(s.jobs)
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("cron store marshal: %w", err)
	}
	return cronAtomicWrite(s.path, data)
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

// cronAtomicWrite escribe data al path usando write-to-temp-then-rename.
func cronAtomicWrite(path string, data []byte) error {
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
