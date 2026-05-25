package cron

import (
	"context"
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

// Start carga todos los jobs habilitados y arranca el cron.
// Pendiente - Fase 15.
func (s *Scheduler) Start(ctx context.Context) error {
	panic("not implemented: Phase 15")
}

// Reload sincroniza el cron con el estado actual del Store.
// Se llama cuando el usuario agrega, elimina o modifica un job.
// Pendiente - Fase 15.
func (s *Scheduler) Reload() error {
	panic("not implemented: Phase 15")
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
