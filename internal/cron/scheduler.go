package cron

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler registra los jobs del Store en robfig/cron y los mantiene
// sincronizados cuando el Store cambia.
// Equivalente al Scheduler de Hermes Python (cron/scheduler.py).
// Fase 15.
type Scheduler struct {
	store   *Store
	runner  *Runner
	inner   *cron.Cron
	entryID map[string]cron.EntryID
	mu      sync.Mutex
}

func NewScheduler(store *Store, runner *Runner) *Scheduler {
	return &Scheduler{
		store:   store,
		runner:  runner,
		inner:   cron.New(cron.WithSeconds()),
		entryID: make(map[string]cron.EntryID),
	}
}

// Start carga todos los jobs habilitados, arranca el cron y bloquea hasta ctx.Done().
func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.Reload(); err != nil {
		return fmt.Errorf("cron scheduler reload: %w", err)
	}
	s.inner.Start()
	<-ctx.Done()
	s.Stop()
	return nil
}

// Reload sincroniza el cron con el estado actual del Store.
// Se llama cuando el usuario agrega, elimina o modifica un job.
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := s.store.List()

	// Eliminar todas las entradas existentes
	for _, id := range s.entryID {
		s.inner.Remove(id)
	}
	s.entryID = make(map[string]cron.EntryID)

	// Re-registrar solo los jobs habilitados
	for _, j := range jobs {
		if !j.Enabled {
			continue
		}
		job := j // capturar para closure
		id, err := s.inner.AddFunc(job.Schedule, func() {
			s.runner.Run(context.Background(), job)
		})
		if err != nil {
			slog.Error("cron add job", "job", job.ID, "schedule", job.Schedule, "err", err)
			continue
		}
		s.entryID[job.ID] = id
	}
	slog.Info("cron scheduler reloaded", "jobs", len(s.entryID))
	return nil
}

// Stop detiene el scheduler de forma ordenada.
func (s *Scheduler) Stop() {
	ctx := s.inner.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		slog.Warn("cron scheduler stop timeout")
	}
}

func (s *Scheduler) Shutdown(_ context.Context) error {
	s.Stop()
	return nil
}
