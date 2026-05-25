package shutdown

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Shutdowner es cualquier componente con ciclo de vida.
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// Manager coordina el shutdown ordenado de todos los componentes.
// Se cierra al recibir SIGINT o SIGTERM.
type Manager struct {
	components []namedShutdowner
	timeout    time.Duration
}

type namedShutdowner struct {
	name string
	s    Shutdowner
}

func NewManager(timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Manager{timeout: timeout}
}

// Register agrega un componente al shutdown. El orden de registro
// determina el orden inverso de shutdown (LIFO).
func (m *Manager) Register(name string, s Shutdowner) {
	m.components = append(m.components, namedShutdowner{name: name, s: s})
}

// WaitForSignal bloquea hasta recibir SIGINT o SIGTERM, luego hace shutdown LIFO.
func (m *Manager) WaitForSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	received := <-sig
	slog.Info("shutdown signal received", "signal", received)
	m.doShutdown()
}

// Shutdown ejecuta el shutdown directamente (util en tests).
func (m *Manager) Shutdown() {
	m.doShutdown()
}

func (m *Manager) doShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	for i := len(m.components) - 1; i >= 0; i-- {
		c := m.components[i]
		slog.Info("shutdown component", "name", c.name)
		if err := c.s.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "name", c.name, "err", err)
		}
	}
	slog.Info("shutdown complete")
}
